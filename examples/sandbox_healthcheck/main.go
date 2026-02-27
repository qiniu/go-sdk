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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.HealthCheck(ctx); err != nil {
		log.Fatalf("健康检查失败: %v", err)
	}
	fmt.Println("API 健康检查通过")
}
