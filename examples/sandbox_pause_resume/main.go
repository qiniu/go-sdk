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

	// 1. 创建沙箱
	templateID := "base"
	timeout := int32(120)
	sb, info, err := c.CreateAndWait(ctx, sandbox.CreateParams{
		TemplateID: templateID,
		Timeout:    &timeout,
	}, sandbox.WithPollInterval(2*time.Second))
	if err != nil {
		log.Fatalf("创建沙箱失败: %v", err)
	}
	fmt.Printf("沙箱已就绪: %s (状态: %s)\n", sb.ID(), info.State)

	// 2. 暂停沙箱
	fmt.Println("正在暂停沙箱...")
	if err := sb.Pause(ctx); err != nil {
		log.Fatalf("暂停失败: %v", err)
	}
	fmt.Println("沙箱已暂停")

	// 3. 确认状态
	detail, err := sb.GetInfo(ctx)
	if err != nil {
		log.Fatalf("获取详情失败: %v", err)
	}
	fmt.Printf("当前状态: %s\n", detail.State)

	// 4. 恢复沙箱（通过 Connect）
	fmt.Println("正在恢复沙箱...")
	resumed, err := c.Connect(ctx, sb.ID(), sandbox.ConnectParams{
		Timeout: 120,
	})
	if err != nil {
		log.Fatalf("恢复失败: %v", err)
	}

	// 等待恢复就绪
	readyInfo, err := resumed.WaitForReady(ctx, sandbox.WithPollInterval(2*time.Second))
	if err != nil {
		log.Fatalf("等待就绪失败: %v", err)
	}
	fmt.Printf("沙箱已恢复: %s (状态: %s)\n", resumed.ID(), readyInfo.State)

	// 5. 清理
	if err := resumed.Kill(ctx); err != nil {
		log.Fatalf("终止失败: %v", err)
	}
	fmt.Println("沙箱已终止")
}
