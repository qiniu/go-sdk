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
		log.Fatal("请设置 Qiniu_API_KEY 环境变量")
	}

	apiURL := os.Getenv("Qiniu_SANDBOX_API_URL")

	c, err := sandbox.NewClient(&sandbox.Config{
		APIKey:   apiKey,
		Endpoint: apiURL,
	})
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// 连接到已有沙箱，需替换为实际的沙箱 ID
	sandboxID := os.Getenv("SANDBOX_ID")
	if sandboxID == "" {
		// 如果未指定，列出现有沙箱并选取第一个
		sandboxes, err := c.List(ctx, nil)
		if err != nil {
			log.Fatalf("列出沙箱失败: %v", err)
		}
		if len(sandboxes) > 0 {
			sandboxID = sandboxes[0].SandboxID
			fmt.Printf("找到运行中的沙箱: %s\n", sandboxID)
		}
	}

	// 如果没有可用沙箱，自行创建一个用于演示连接
	if sandboxID == "" {
		fmt.Println("没有运行中的沙箱，自动创建一个用于演示...")

		templates, err := c.ListTemplates(ctx, nil)
		if err != nil {
			log.Fatalf("列出模板失败: %v", err)
		}

		var templateID string
		for _, tmpl := range templates {
			if tmpl.BuildStatus == apis.TemplateBuildStatusReady || tmpl.BuildStatus == sandbox.TemplateBuildStatusUploaded {
				templateID = tmpl.TemplateID
				break
			}
		}
		if templateID == "" {
			log.Fatal("没有构建成功的模板")
		}

		timeout := int32(60)
		created, _, err := c.CreateAndWait(ctx, apis.CreateSandboxJSONRequestBody{
			TemplateID: templateID,
			Timeout:    &timeout,
		}, 2*time.Second)
		if err != nil {
			log.Fatalf("创建沙箱失败: %v", err)
		}
		sandboxID = created.SandboxID
		fmt.Printf("沙箱已创建: %s\n", sandboxID)

		defer func() {
			killCtx, killCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer killCancel()
			if err := created.Kill(killCtx); err != nil {
				log.Printf("终止沙箱失败: %v", err)
			} else {
				fmt.Printf("沙箱 %s 已终止\n", sandboxID)
			}
		}()
	}

	timeout := int32(300)
	sb, err := c.Connect(ctx, sandboxID, apis.ConnectSandboxJSONRequestBody{
		Timeout: timeout,
	})
	if err != nil {
		log.Fatalf("连接沙箱失败: %v", err)
	}

	fmt.Printf("已连接到沙箱: %s (模板: %s)\n", sb.SandboxID, sb.TemplateID)

	// 获取沙箱详情
	info, err := sb.GetInfo(ctx)
	if err != nil {
		log.Fatalf("获取详情失败: %v", err)
	}
	fmt.Printf("状态: %s, CPU: %d 核, 内存: %d MB\n", info.State, info.CPUCount, info.MemoryMB)
}
