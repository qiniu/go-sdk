package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/qiniu/go-sdk/v7/sandbox"
	"github.com/qiniu/go-sdk/v7/sandbox/apis"
)

func main() {
	apiKey := os.Getenv("Qiniu_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("E2B_API_KEY")
	}
	if apiKey == "" {
		log.Fatal("请设置 Qiniu_API_KEY 或 E2B_API_KEY 环境变量")
	}

	apiURL := os.Getenv("Qiniu_SANDBOX_API_URL")
	if apiURL == "" {
		apiURL = os.Getenv("E2B_API_URL")
	}

	c, err := sandbox.NewClient(&sandbox.Config{
		APIKey:   apiKey,
		Endpoint: apiURL,
	})
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// 1. 列出模板，选取第一个可用模板
	templates, err := c.ListTemplates(ctx, nil)
	if err != nil {
		log.Fatalf("列出模板失败: %v", err)
	}

	var templateID string
	for _, tmpl := range templates {
		if tmpl.BuildStatus == apis.TemplateBuildStatusReady || tmpl.BuildStatus == "uploaded" {
			templateID = tmpl.TemplateID
			break
		}
	}
	if templateID == "" {
		log.Fatal("没有构建成功的模板")
	}
	fmt.Printf("使用模板: %s\n", templateID)

	// 2. 创建沙箱并等待就绪
	timeout := int32(120)
	sb, _, err := c.CreateAndWait(ctx, apis.CreateSandboxJSONRequestBody{
		TemplateID: templateID,
		Timeout:    &timeout,
	}, 2*time.Second)
	if err != nil {
		log.Fatalf("创建沙箱失败: %v", err)
	}
	fmt.Printf("沙箱已就绪: %s\n", sb.SandboxID)

	defer func() {
		killCtx, killCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer killCancel()
		if err := sb.Kill(killCtx); err != nil {
			log.Printf("终止沙箱失败: %v", err)
		} else {
			fmt.Printf("沙箱 %s 已终止\n", sb.SandboxID)
		}
	}()

	// 3. 获取端口域名
	host := sb.GetHost(8080)
	fmt.Printf("端口 8080 访问地址: %s\n", host)

	// 4. 文件系统操作
	fmt.Println("\n--- 文件系统操作 ---")

	// 写入文件
	_, err = sb.Files().Write(ctx, "/tmp/hello.txt", []byte("Hello from Go SDK!\n"))
	if err != nil {
		log.Fatalf("写入文件失败: %v", err)
	}
	fmt.Println("文件已写入: /tmp/hello.txt")

	// 读取文件
	content, err := sb.Files().Read(ctx, "/tmp/hello.txt")
	if err != nil {
		log.Fatalf("读取文件失败: %v", err)
	}
	fmt.Printf("文件内容: %s", string(content))

	// 创建目录
	_, err = sb.Files().MakeDir(ctx, "/tmp/mydir")
	if err != nil {
		log.Fatalf("创建目录失败: %v", err)
	}
	fmt.Println("目录已创建: /tmp/mydir")

	// 列出目录
	entries, err := sb.Files().List(ctx, "/tmp")
	if err != nil {
		log.Fatalf("列出目录失败: %v", err)
	}
	fmt.Printf("/tmp 目录内容 (%d 项):\n", len(entries))
	for _, e := range entries {
		fmt.Printf("  %s %s (%s, %d bytes)\n", e.Type, e.Name, e.Permissions, e.Size)
	}

	// 5. 执行命令
	fmt.Println("\n--- 命令执行 ---")

	result, err := sb.Commands().Run(ctx, "echo hello world")
	if err != nil {
		log.Fatalf("执行命令失败: %v", err)
	}
	fmt.Printf("命令: echo hello world\n")
	fmt.Printf("退出码: %d\n", result.ExitCode)
	fmt.Printf("stdout: %s", result.Stdout)

	// 带环境变量的命令
	result, err = sb.Commands().Run(ctx, "echo $MY_VAR",
		sandbox.WithEnvs(map[string]string{"MY_VAR": "sandbox-value"}),
	)
	if err != nil {
		log.Fatalf("执行命令失败: %v", err)
	}
	fmt.Printf("命令: echo $MY_VAR (MY_VAR=sandbox-value)\n")
	fmt.Printf("stdout: %s", result.Stdout)

	// 6. 下载/上传 URL
	fmt.Println("\n--- 文件 URL ---")
	downloadURL := sb.DownloadURL("/tmp/hello.txt")
	fmt.Printf("下载 URL: %s\n", downloadURL)

	uploadURL := sb.UploadURL("/tmp/upload.txt")
	fmt.Printf("上传 URL: %s\n", uploadURL)
}
