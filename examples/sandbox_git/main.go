// sandbox_git 演示如何使用 Sandbox 的 Git 高层接口完成常见 git 操作。
//
// 本示例完全在沙箱内部进行，不依赖外部仓库；如果设置了 QINIU_GIT_REPO_URL（HTTPS）、
// QINIU_GIT_USERNAME、QINIU_GIT_PASSWORD 环境变量，会额外演示带凭证的 Clone。
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

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
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

	timeout := int32(180)
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

	// 2. 初始化仓库
	fmt.Println("\n--- Init ---")
	if _, err := git.Init(ctx, repoPath, &sandbox.InitOptions{InitialBranch: "main"}); err != nil {
		log.Fatalf("Init 失败: %v", err)
	}
	fmt.Printf("已初始化仓库: %s（initial-branch=main）\n", repoPath)

	// 3. 配置提交用户（local scope）
	fmt.Println("\n--- ConfigureUser ---")
	if _, err := git.ConfigureUser(ctx, "Sandbox Demo", "demo@example.com", &sandbox.ConfigOptions{
		Scope: sandbox.GitConfigScopeLocal,
		Path:  repoPath,
	}); err != nil {
		log.Fatalf("ConfigureUser 失败: %v", err)
	}
	name, err := git.GetConfig(ctx, "user.name", &sandbox.ConfigOptions{
		Scope: sandbox.GitConfigScopeLocal,
		Path:  repoPath,
	})
	if err != nil {
		log.Fatalf("GetConfig 失败: %v", err)
	}
	fmt.Printf("user.name = %q\n", name)

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
	if _, err := sb.Files().Write(ctx, repoPath+"/dirty.txt", []byte("dirty\n")); err != nil {
		log.Fatalf("写入脏文件失败: %v", err)
	}
	if _, err := git.Add(ctx, repoPath, nil); err != nil {
		log.Fatalf("Add 失败: %v", err)
	}
	st, _ = git.Status(ctx, repoPath, nil)
	fmt.Printf("Add 后 staged=%d，unstaged=%d\n", st.StagedCount(), st.UnstagedCount())

	// Reset：取消暂存（仅 paths，无 mode）
	if _, err := git.Reset(ctx, repoPath, &sandbox.ResetOptions{
		Paths: []string{"dirty.txt"},
	}); err != nil {
		log.Fatalf("Reset 失败: %v", err)
	}
	st, _ = git.Status(ctx, repoPath, nil)
	fmt.Printf("Reset 后 staged=%d，unstaged=%d\n", st.StagedCount(), st.UnstagedCount())

	// Restore：丢弃工作区改动（这里 dirty.txt 是新文件，restore 不会删除未跟踪文件，仅作 API 演示）
	if _, err := git.Restore(ctx, repoPath, &sandbox.RestoreOptions{
		Paths: []string{"README.md"},
	}); err != nil {
		log.Fatalf("Restore 失败: %v", err)
	}
	fmt.Println("Restore: 已恢复 README.md 工作区版本")

	// 8. Remote 管理
	fmt.Println("\n--- Remote ---")
	if _, err := git.RemoteAdd(ctx, repoPath, "origin", "https://example.com/demo.git", nil); err != nil {
		log.Fatalf("RemoteAdd 失败: %v", err)
	}
	url, err := git.RemoteGet(ctx, repoPath, "origin", nil)
	if err != nil {
		log.Fatalf("RemoteGet 失败: %v", err)
	}
	fmt.Printf("origin URL = %s\n", url)

	// 不存在的 remote 返回空字符串
	missing, err := git.RemoteGet(ctx, repoPath, "nonexistent", nil)
	if err != nil {
		log.Fatalf("RemoteGet(nonexistent) 失败: %v", err)
	}
	fmt.Printf("nonexistent URL = %q（未配置）\n", missing)

	// 9. 可选：带凭证的 Clone（仅在 QINIU_GIT_REPO_URL 等环境变量齐备时执行）
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
		fmt.Println("\n--- Clone 演示已跳过（未设置 QINIU_GIT_REPO_URL / QINIU_GIT_USERNAME / QINIU_GIT_PASSWORD）---")
	}
}
