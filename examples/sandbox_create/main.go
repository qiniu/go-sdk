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

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 创建沙箱，templateID 需替换为实际可用的模板 ID
	templateID := "base"
	timeout := int32(300)
	sb, err := c.Create(ctx, apis.CreateSandboxJSONRequestBody{
		TemplateID: templateID,
		Timeout:    &timeout,
	})
	if err != nil {
		log.Fatalf("创建沙箱失败: %v", err)
	}

	fmt.Printf("沙箱已创建: %s (模板: %s)\n", sb.SandboxID, sb.TemplateID)

	// 演示完毕，终止沙箱释放资源
	killCtx, killCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer killCancel()
	if err := sb.Kill(killCtx); err != nil {
		log.Printf("终止沙箱失败: %v", err)
	} else {
		fmt.Printf("沙箱 %s 已终止\n", sb.SandboxID)
	}
}
