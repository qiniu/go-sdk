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

	// 1. 列出模板，选取第一个可用模板
	templates, err := c.ListTemplates(ctx, nil)
	if err != nil {
		log.Fatalf("列出模板失败: %v", err)
	}
	if len(templates) == 0 {
		log.Fatal("没有可用模板")
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
	fmt.Printf("使用模板: %s\n", templateID)

	// 2. 创建沙箱并等待就绪
	timeout := int32(120)
	sb, info, err := c.CreateAndWait(ctx, apis.CreateSandboxJSONRequestBody{
		TemplateID: templateID,
		Timeout:    &timeout,
	}, 2*time.Second)
	if err != nil {
		log.Fatalf("创建沙箱失败: %v", err)
	}
	fmt.Printf("沙箱已就绪: %s (状态: %s)\n", sb.SandboxID, info.State)

	// 3. 获取沙箱详情
	detail, err := sb.GetInfo(ctx)
	if err != nil {
		log.Fatalf("获取详情失败: %v", err)
	}
	fmt.Printf("CPU: %d 核, 内存: %d MB\n", detail.CPUCount, detail.MemoryMB)

	// 4. 检查运行状态
	running, err := sb.IsRunning(ctx)
	if err != nil {
		log.Fatalf("检查状态失败: %v", err)
	}
	fmt.Printf("是否运行中: %v\n", running)

	// 5. 更新超时时间
	if err := sb.SetTimeout(ctx, 5*time.Minute); err != nil {
		log.Fatalf("更新超时失败: %v", err)
	}
	fmt.Println("超时时间已更新为 5 分钟")

	// 6. 延长存活时间（Refresh）
	duration := 300
	if err := sb.Refresh(ctx, apis.RefreshSandboxJSONRequestBody{
		Duration: &duration,
	}); err != nil {
		log.Fatalf("Refresh 失败: %v", err)
	}
	fmt.Println("沙箱存活时间已延长 300 秒")

	// 7. 终止沙箱
	if err := sb.Kill(ctx); err != nil {
		log.Fatalf("终止沙箱失败: %v", err)
	}
	fmt.Printf("沙箱 %s 已终止\n", sb.SandboxID)
}
