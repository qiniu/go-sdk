// sandbox_git 演示如何使用 Sandbox 的 Git 高层接口完成常见 git 操作。
//
// 本示例完全在沙箱内部进行，不依赖外部仓库：通过在沙箱内本地建立一个
// bare 仓库当作 "remote"，串起 Init / Add / Commit / Push / Pull 闭环。
// 如果设置了 QINIU_GIT_REPO_URL（HTTPS）、QINIU_GIT_USERNAME、
// QINIU_GIT_PASSWORD 环境变量，会额外演示带凭证的 Clone。
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/qiniu/go-sdk/v7/sandbox"
)

func main() {
	apiKey := os.Getenv("QINIU_API_KEY")
	if apiKey == "" {
		log.Fatal("请设置 QINIU_API_KEY 环境变量")
	}

	apiURL := os.Getenv("QINIU_SANDBOX_API_URL")

	c, err := sandbox.NewClient(&sandbox.Config{
		APIKey:   apiKey,
		Endpoint: apiURL,
	})
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()

	// 1. 选取可用模板并创建沙箱
	templates, err := c.ListTemplates(ctx, nil)
	if err != nil {
		log.Fatalf("列出模板失败: %v", err)
	}
	var templateID string
	for _, tmpl := range templates {
		if tmpl.BuildStatus == sandbox.BuildStatusReady || tmpl.BuildStatus == sandbox.BuildStatusUploaded {
			templateID = tmpl.TemplateID
			break
		}
	}
	if templateID == "" {
		log.Fatal("没有构建成功的模板")
	}
	fmt.Printf("使用模板: %s\n", templateID)

	timeout := int32(240)
	sb, _, err := c.CreateAndWait(ctx, sandbox.CreateParams{
		TemplateID: templateID,
		Timeout:    &timeout,
	}, sandbox.WithPollInterval(2*time.Second))
	if err != nil {
		log.Fatalf("创建沙箱失败: %v", err)
	}
	fmt.Printf("沙箱已就绪: %s\n", sb.ID())

	defer func() {
		killCtx, killCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer killCancel()
		if err := sb.Kill(killCtx); err != nil {
			log.Printf("终止沙箱失败: %v", err)
		} else {
			fmt.Printf("沙箱 %s 已终止\n", sb.ID())
		}
	}()

	git := sb.Git()
	repoPath := "/tmp/demo-repo"
	bareRepoPath := "/tmp/demo-remote.git"
	consumerPath := "/tmp/demo-consumer"

	// 2. 初始化工作仓库
	fmt.Println("\n--- Init ---")
	if _, err := git.Init(ctx, repoPath, &sandbox.InitOptions{InitialBranch: "main"}); err != nil {
		log.Fatalf("Init 失败: %v", err)
	}
	fmt.Printf("已初始化仓库: %s（initial-branch=main）\n", repoPath)

	// 同时初始化一个 bare 仓库当作 remote
	if _, err := git.Init(ctx, bareRepoPath, &sandbox.InitOptions{Bare: true, InitialBranch: "main"}); err != nil {
		log.Fatalf("Init(bare) 失败: %v", err)
	}
	fmt.Printf("已初始化 bare 仓库: %s\n", bareRepoPath)

	// 3. 配置提交用户（local scope）
	fmt.Println("\n--- ConfigureUser / SetConfig / GetConfig ---")
	if _, err := git.ConfigureUser(ctx, "Sandbox Demo", "demo@example.com", &sandbox.ConfigOptions{
		Scope: sandbox.GitConfigScopeLocal,
		Path:  repoPath,
	}); err != nil {
		log.Fatalf("ConfigureUser 失败: %v", err)
	}
	// SetConfig 演示：单独设置一个非 user 的键
	if _, err := git.SetConfig(ctx, "core.autocrlf", "input", &sandbox.ConfigOptions{
		Scope: sandbox.GitConfigScopeLocal,
		Path:  repoPath,
	}); err != nil {
		log.Fatalf("SetConfig 失败: %v", err)
	}
	for _, key := range []string{"user.name", "user.email", "core.autocrlf"} {
		val, err := git.GetConfig(ctx, key, &sandbox.ConfigOptions{
			Scope: sandbox.GitConfigScopeLocal,
			Path:  repoPath,
		})
		if err != nil {
			log.Fatalf("GetConfig(%s) 失败: %v", key, err)
		}
		fmt.Printf("  %s = %q\n", key, val)
	}
	// 不存在的键返回空字符串
	missing, err := git.GetConfig(ctx, "user.notexist", &sandbox.ConfigOptions{
		Scope: sandbox.GitConfigScopeLocal,
		Path:  repoPath,
	})
	if err != nil {
		log.Fatalf("GetConfig(missing) 失败: %v", err)
	}
	fmt.Printf("  user.notexist = %q（未配置）\n", missing)

	// 4. 写入文件并暂存提交
	fmt.Println("\n--- Add / Commit ---")
	if _, err := sb.Files().Write(ctx, repoPath+"/README.md", []byte("# demo\n")); err != nil {
		log.Fatalf("写入文件失败: %v", err)
	}
	if _, err := git.Add(ctx, repoPath, nil); err != nil { // nil → 默认 -A
		log.Fatalf("Add 失败: %v", err)
	}
	if _, err := git.Commit(ctx, repoPath, "feat: initial commit", &sandbox.CommitOptions{
		AuthorName:  "Sandbox Demo",
		AuthorEmail: "demo@example.com",
	}); err != nil {
		log.Fatalf("Commit 失败: %v", err)
	}
	fmt.Println("已创建初始提交")

	// 5. 查看状态
	fmt.Println("\n--- Status ---")
	st, err := git.Status(ctx, repoPath, nil)
	if err != nil {
		log.Fatalf("Status 失败: %v", err)
	}
	fmt.Printf("CurrentBranch=%s, Detached=%v, Clean=%v, Total=%d, Staged=%d, Unstaged=%d\n",
		st.CurrentBranch, st.Detached, st.IsClean(), st.TotalCount(), st.StagedCount(), st.UnstagedCount())

	// 6. 分支管理：CreateBranch -> 修改 -> Commit -> Branches -> CheckoutBranch -> DeleteBranch
	fmt.Println("\n--- Branches ---")
	if _, err := git.CreateBranch(ctx, repoPath, "feature/x", nil); err != nil {
		log.Fatalf("CreateBranch 失败: %v", err)
	}
	if _, err := sb.Files().Write(ctx, repoPath+"/feature.txt", []byte("hello feature\n")); err != nil {
		log.Fatalf("写入特性文件失败: %v", err)
	}
	if _, err := git.Add(ctx, repoPath, &sandbox.AddOptions{Files: []string{"feature.txt"}}); err != nil {
		log.Fatalf("Add(files) 失败: %v", err)
	}
	if _, err := git.Commit(ctx, repoPath, "feat: add feature.txt", nil); err != nil {
		log.Fatalf("Commit 失败: %v", err)
	}

	branches, err := git.Branches(ctx, repoPath, nil)
	if err != nil {
		log.Fatalf("Branches 失败: %v", err)
	}
	fmt.Printf("分支: %v，当前: %s\n", branches.Branches, branches.CurrentBranch)

	if _, err := git.CheckoutBranch(ctx, repoPath, "main", nil); err != nil {
		log.Fatalf("CheckoutBranch 失败: %v", err)
	}
	if _, err := git.DeleteBranch(ctx, repoPath, "feature/x", &sandbox.DeleteBranchOptions{Force: true}); err != nil {
		log.Fatalf("DeleteBranch 失败: %v", err)
	}
	fmt.Println("已切回 main 并强制删除 feature/x")

	// 7. Reset / Restore 演示
	fmt.Println("\n--- Reset / Restore ---")

	// 7a. Reset paths-only：取消暂存（不带 mode）
	if _, err := sb.Files().Write(ctx, repoPath+"/dirty.txt", []byte("dirty\n")); err != nil {
		log.Fatalf("写入脏文件失败: %v", err)
	}
	if _, err := git.Add(ctx, repoPath, nil); err != nil {
		log.Fatalf("Add 失败: %v", err)
	}
	st, err = git.Status(ctx, repoPath, nil)
	if err != nil {
		log.Fatalf("Status 失败: %v", err)
	}
	fmt.Printf("Add 后 staged=%d，unstaged=%d\n", st.StagedCount(), st.UnstagedCount())
	if _, err := git.Reset(ctx, repoPath, &sandbox.ResetOptions{
		Paths: []string{"dirty.txt"},
	}); err != nil {
		log.Fatalf("Reset(paths) 失败: %v", err)
	}
	st, err = git.Status(ctx, repoPath, nil)
	if err != nil {
		log.Fatalf("Status 失败: %v", err)
	}
	fmt.Printf("Reset(paths) 后 staged=%d，unstaged=%d\n", st.StagedCount(), st.UnstagedCount())

	// 7b. Reset --hard：丢弃工作区改动并把 HEAD 重置到指定提交
	if _, err := sb.Files().Write(ctx, repoPath+"/README.md", []byte("# demo (modified)\n")); err != nil {
		log.Fatalf("修改 README 失败: %v", err)
	}
	if _, err := git.Reset(ctx, repoPath, &sandbox.ResetOptions{
		Mode:   sandbox.GitResetModeHard,
		Target: "HEAD",
	}); err != nil {
		log.Fatalf("Reset(--hard HEAD) 失败: %v", err)
	}
	readme, err := sb.Files().ReadText(ctx, repoPath+"/README.md")
	if err != nil {
		log.Fatalf("读取 README 失败: %v", err)
	}
	fmt.Printf("Reset --hard 后 README.md = %q\n", readme)

	// 7c. Restore --staged：取消暂存（保留工作区）
	if _, err := sb.Files().Write(ctx, repoPath+"/README.md", []byte("# demo (staged change)\n")); err != nil {
		log.Fatalf("修改 README 失败: %v", err)
	}
	if _, err := git.Add(ctx, repoPath, nil); err != nil {
		log.Fatalf("Add 失败: %v", err)
	}
	stagedPtr := boolPtr(true)
	if _, err := git.Restore(ctx, repoPath, &sandbox.RestoreOptions{
		Paths:  []string{"README.md"},
		Staged: stagedPtr, // 仅取消暂存，工作区保留
	}); err != nil {
		log.Fatalf("Restore(--staged) 失败: %v", err)
	}
	st, err = git.Status(ctx, repoPath, nil)
	if err != nil {
		log.Fatalf("Status 失败: %v", err)
	}
	fmt.Printf("Restore --staged 后 staged=%d，unstaged=%d\n", st.StagedCount(), st.UnstagedCount())

	// 7d. Restore --worktree --source=HEAD：把工作区改动恢复到 HEAD 版本
	if _, err := git.Restore(ctx, repoPath, &sandbox.RestoreOptions{
		Paths:  []string{"README.md"},
		Source: "HEAD",
	}); err != nil {
		log.Fatalf("Restore(--source HEAD) 失败: %v", err)
	}
	readme, err = sb.Files().ReadText(ctx, repoPath+"/README.md")
	if err != nil {
		log.Fatalf("读取 README 失败: %v", err)
	}
	fmt.Printf("Restore --source HEAD 后 README.md = %q\n", readme)

	// 8. Remote 管理（含 Overwrite）
	fmt.Println("\n--- Remote ---")
	// 先用一个占位 URL 添加 origin
	if _, err := git.RemoteAdd(ctx, repoPath, "origin", "https://example.com/placeholder.git", nil); err != nil {
		log.Fatalf("RemoteAdd 失败: %v", err)
	}
	// 再用 Overwrite=true 把它改成本地 bare 仓库（用于后面的 push/pull）
	if _, err := git.RemoteAdd(ctx, repoPath, "origin", bareRepoPath, &sandbox.RemoteAddOptions{
		Overwrite: true,
	}); err != nil {
		log.Fatalf("RemoteAdd(overwrite) 失败: %v", err)
	}
	url, err := git.RemoteGet(ctx, repoPath, "origin", nil)
	if err != nil {
		log.Fatalf("RemoteGet 失败: %v", err)
	}
	fmt.Printf("origin URL = %s\n", url)

	missingRemote, err := git.RemoteGet(ctx, repoPath, "nonexistent", nil)
	if err != nil {
		log.Fatalf("RemoteGet(nonexistent) 失败: %v", err)
	}
	fmt.Printf("nonexistent URL = %q（未配置）\n", missingRemote)

	// 9. Push（不带凭证；SetUpstream 默认 true）
	fmt.Println("\n--- Push ---")
	if _, err := git.Push(ctx, repoPath, &sandbox.PushOptions{
		Remote: "origin",
		Branch: "main",
	}); err != nil {
		log.Fatalf("Push 失败: %v", err)
	}
	fmt.Printf("已推送到 %s（main）\n", bareRepoPath)

	// 10. Pull：再 clone 出一个 consumer 仓库，从同一个 bare remote 拉取
	fmt.Println("\n--- Pull（通过本地 clone 演示）---")
	// 用 git clone 本地路径作为 remote 来初始化 consumer（无凭证场景）
	if _, err := git.Clone(ctx, bareRepoPath, &sandbox.CloneOptions{
		Path: consumerPath,
	}); err != nil {
		log.Fatalf("本地 Clone 失败: %v", err)
	}
	fmt.Printf("已 clone 到 %s\n", consumerPath)

	// 在 repoPath 里再做一次提交并 push
	if _, err := sb.Files().Write(ctx, repoPath+"/CHANGELOG.md", []byte("# v1\n")); err != nil {
		log.Fatalf("写入 CHANGELOG 失败: %v", err)
	}
	if _, err := git.Add(ctx, repoPath, nil); err != nil {
		log.Fatalf("Add 失败: %v", err)
	}
	if _, err := git.Commit(ctx, repoPath, "docs: add CHANGELOG", nil); err != nil {
		log.Fatalf("Commit 失败: %v", err)
	}
	if _, err := git.Push(ctx, repoPath, &sandbox.PushOptions{
		Remote: "origin",
		Branch: "main",
	}); err != nil {
		log.Fatalf("Push(2) 失败: %v", err)
	}

	// 在 consumer 里 Pull 拉取最新提交
	if _, err := git.Pull(ctx, consumerPath, &sandbox.PullOptions{
		Remote: "origin",
		Branch: "main",
	}); err != nil {
		log.Fatalf("Pull 失败: %v", err)
	}
	exists, err := sb.Files().Exists(ctx, consumerPath+"/CHANGELOG.md")
	if err != nil {
		log.Fatalf("Exists 失败: %v", err)
	}
	fmt.Printf("Pull 后 consumer/CHANGELOG.md 存在 = %v\n", exists)

	// 11. DangerouslyAuthenticate：演示 API 调用形态（仅写到沙箱内的全局 credential store）
	fmt.Println("\n--- DangerouslyAuthenticate（仅在沙箱内持久化，不影响外部环境）---")
	if _, err := git.DangerouslyAuthenticate(ctx, &sandbox.AuthenticateOptions{
		Username: "demo-user",
		Password: "demo-token",
		Host:     "example.com",
		// Protocol 缺省即 "https"，这里显式写出仅作演示
		Protocol: "https",
	}); err != nil {
		log.Fatalf("DangerouslyAuthenticate 失败: %v", err)
	}
	fmt.Println("已通过 git credential approve 写入 example.com 的凭证")

	// 12. 可选：带凭证的 Clone（仅在 QINIU_GIT_REPO_URL 等环境变量齐备时执行）
	repoURL := os.Getenv("QINIU_GIT_REPO_URL")
	username := os.Getenv("QINIU_GIT_USERNAME")
	password := os.Getenv("QINIU_GIT_PASSWORD")
	if repoURL != "" && username != "" && password != "" {
		fmt.Println("\n--- Clone（HTTPS + token，clone 后会自动剥离 origin URL 中的凭证）---")
		clonePath := "/tmp/cloned-repo"
		if _, err := git.Clone(ctx, repoURL, &sandbox.CloneOptions{
			Path:     clonePath,
			Depth:    1,
			Username: username,
			Password: password,
		}); err != nil {
			log.Fatalf("Clone 失败: %v", err)
		}
		clonedURL, err := git.RemoteGet(ctx, clonePath, "origin", nil)
		if err != nil {
			log.Fatalf("RemoteGet(cloned) 失败: %v", err)
		}
		fmt.Printf("clone 完成，origin URL = %s（不含凭证）\n", clonedURL)
	} else {
		fmt.Println("\n--- 远程 Clone 演示已跳过（未设置 QINIU_GIT_REPO_URL / QINIU_GIT_USERNAME / QINIU_GIT_PASSWORD）---")
	}
}

// boolPtr 返回 bool 值的指针。
func boolPtr(v bool) *bool { return &v }
