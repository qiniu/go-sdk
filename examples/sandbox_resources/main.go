package main

import (
	"context"
	"log"
	"os"

	"github.com/qiniu/go-sdk/v7/sandbox"
)

func main() {
	// 确保设置了环境变量 QINIU_API_KEY
	apiKey := os.Getenv("QINIU_API_KEY")
	if apiKey == "" {
		log.Fatal("请设置 QINIU_API_KEY 环境变量")
	}

	apiURL := os.Getenv("QINIU_SANDBOX_API_URL")

	// 演示用的 GitHub 仓库与 token，可通过环境变量覆盖
	repoURL := os.Getenv("QINIU_SANDBOX_GIT_REPO_URL")
	if repoURL == "" {
		repoURL = "https://github.com/qiniu/go-sdk.git"
	}
	githubToken := os.Getenv("GITHUB_TOKEN") // 私有仓库必填，公共仓库可留空

	ctx := context.Background()

	// 初始化客户端
	client, err := sandbox.NewClient(&sandbox.Config{
		APIKey:   apiKey,
		Endpoint: apiURL,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	log.Println("Creating sandbox with GitHub repository resource...")

	// 通过 Resources 在沙箱启动前由平台拉取 GitHub 仓库快照并挂载到 /workspace/repo。
	// 通过 Injections 注入 GitHub 凭证，使沙箱内对 github.com / api.github.com 的
	// HTTPS 请求自动鉴权（token 不会以明文形式暴露给沙箱内进程）。
	repoResource := sandbox.GitRepositoryResource{
		Type:      sandbox.GitRepositoryTypeGithub,
		URL:       repoURL,
		MountPath: "/workspace/repo",
	}
	if githubToken != "" {
		repoResource.AuthorizationToken = &githubToken
	}

	params := sandbox.CreateParams{
		TemplateID: "base",
		Resources: &[]sandbox.SandboxResourceSpec{
			{GitRepository: &repoResource},
		},
	}

	// 创建并等待沙箱就绪
	sb, info, err := client.CreateAndWait(ctx, params)
	if err != nil {
		log.Fatalf("Failed to create sandbox: %v", err)
	}
	defer func() {
		log.Println("Killing sandbox...")
		_ = sb.Kill(ctx)
	}()

	log.Printf("Sandbox created successfully! ID: %s, State: %s\n", sb.ID(), info.State)

	// 列出已挂载的仓库内容，验证资源已就位
	listCmd := "ls -la /workspace/repo | head -20"
	log.Printf("Executing command in sandbox:\n$ %s\n", listCmd)

	result, err := sb.Commands().Run(ctx, listCmd)
	if err != nil {
		log.Fatalf("Failed to run command: %v", err)
	}

	log.Printf("ExitCode: %d\n", result.ExitCode)
	if result.Stdout != "" {
		log.Printf("Stdout:\n%s\n", result.Stdout)
	}
	if result.Stderr != "" {
		log.Printf("Stderr:\n%s\n", result.Stderr)
	}
}
