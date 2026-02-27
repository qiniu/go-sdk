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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 连接到已有沙箱，需替换为实际的沙箱 ID
	sandboxID := os.Getenv("SANDBOX_ID")
	if sandboxID == "" {
		// 如果未指定，列出现有沙箱并选取第一个
		sandboxes, err := c.List(ctx, nil)
		if err != nil {
			log.Fatalf("列出沙箱失败: %v", err)
		}
		if len(sandboxes) == 0 {
			log.Fatal("没有运行中的沙箱，请先创建一个")
		}
		sandboxID = sandboxes[0].SandboxID
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
