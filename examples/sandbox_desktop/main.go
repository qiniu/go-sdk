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
	client, err := sandbox.NewClient(&sandbox.Config{
		APIKey:   apiKey,
		Endpoint: os.Getenv("QINIU_SANDBOX_API_URL"),
	})
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	timeout := int32(180)
	allowPublicTraffic := true
	desktop, err := client.CreateDesktop(ctx, sandbox.DesktopCreateParams{
		CreateParams: sandbox.CreateParams{
			Timeout: &timeout,
			Network: &sandbox.NetworkConfig{AllowPublicTraffic: &allowPublicTraffic},
		},
		Resolution: sandbox.ScreenSize{Width: 1024, Height: 768},
	})
	if err != nil {
		log.Fatalf("创建桌面沙箱失败: %v", err)
	}
	defer func() {
		streamCtx, streamCancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := desktop.Stream().Stop(streamCtx); err != nil {
			log.Printf("停止桌面流失败: %v", err)
		}
		streamCancel()
		killCtx, killCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer killCancel()
		if err := desktop.Kill(killCtx); err != nil {
			log.Printf("终止桌面沙箱失败: %v", err)
		}
	}()

	if err := desktop.Open(ctx, "https://developer.qiniu.com"); err != nil {
		log.Fatalf("打开浏览器失败: %v", err)
	}
	if err := desktop.Stream().Start(ctx, nil); err != nil {
		log.Fatalf("启动桌面流失败: %v", err)
	}
	authKey, err := desktop.Stream().AuthKey()
	if err != nil {
		log.Fatalf("获取桌面流密码失败: %v", err)
	}
	streamURL, err := desktop.Stream().URL(&sandbox.DesktopStreamURLOptions{AuthKey: authKey})
	if err != nil {
		log.Fatalf("生成桌面流地址失败: %v", err)
	}

	fmt.Printf("Desktop Sandbox: %s\n", desktop.ID())
	fmt.Printf("noVNC URL: %s\n", streamURL)
	fmt.Println("注意：URL 包含 VNC 密码，可能被浏览器历史、代理或日志记录。")
	fmt.Println("按 Enter 终止 Desktop Sandbox。")
	_, _ = fmt.Scanln()
}
