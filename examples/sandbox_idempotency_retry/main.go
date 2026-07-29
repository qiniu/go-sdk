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
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	apiKey := os.Getenv("QINIU_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("请设置 QINIU_API_KEY 环境变量")
	}

	apiURL := os.Getenv("QINIU_SANDBOX_API_URL")

	c, err := sandbox.NewClient(&sandbox.Config{
		APIKey:   apiKey,
		Endpoint: apiURL,
	})
	if err != nil {
		return fmt.Errorf("创建客户端失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	timeout := int32(300)

	// 指定相同的幂等键连续创建两次，第二次应返回同一沙箱
	idempotencyKey := fmt.Sprintf("sdk-example-%d", time.Now().Unix())
	fmt.Printf("幂等键: %s\n\n", idempotencyKey)

	// 第一次创建
	sb1, err := c.Create(ctx, sandbox.CreateParams{
		TemplateID:     "base",
		Timeout:        &timeout,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return fmt.Errorf("第一次创建失败: %w", err)
	}
	defer cleanupSandbox(sb1)
	fmt.Printf("第一次创建: %s\n", sb1.ID())

	// 第二次创建 — 同一幂等键，应返回同一沙箱
	sb2, err := c.Create(ctx, sandbox.CreateParams{
		TemplateID:     "base",
		Timeout:        &timeout,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return fmt.Errorf("第二次创建失败: %w", err)
	}
	if sb2.ID() != sb1.ID() {
		defer cleanupSandbox(sb2)
	}
	fmt.Printf("第二次创建: %s\n", sb2.ID())

	if sb1.ID() == sb2.ID() {
		fmt.Println("\n幂等重试验证通过：两次创建返回同一沙箱")
	} else {
		fmt.Printf("\nWARNING: 两次创建返回不同沙箱: %s vs %s\n", sb1.ID(), sb2.ID())
	}

	return nil
}

func cleanupSandbox(sb *sandbox.Sandbox) {
	killCtx, killCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer killCancel()
	if err := sb.Kill(killCtx); err != nil {
		log.Printf("清理沙箱 %s 失败: %v", sb.ID(), err)
		return
	}
	log.Printf("沙箱 %s 已清理", sb.ID())
}
