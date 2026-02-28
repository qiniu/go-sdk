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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 列出所有运行中的沙箱
	sandboxes, err := c.List(ctx, nil)
	if err != nil {
		log.Fatalf("列出沙箱失败: %v", err)
	}

	fmt.Printf("共 %d 个运行中的沙箱:\n", len(sandboxes))
	for _, sb := range sandboxes {
		fmt.Printf("  - %s (模板: %s)\n", sb.SandboxID, sb.TemplateID)
	}

	// 使用 ListV2 列出沙箱（支持分页和状态过滤）
	fmt.Println("\n=== ListV2（按状态过滤）===")
	states := []apis.SandboxState{apis.Running}
	sandboxesV2, err := c.ListV2(ctx, &apis.ListSandboxesV2Params{
		State: &states,
	})
	if err != nil {
		log.Fatalf("ListV2 失败: %v", err)
	}
	fmt.Printf("共 %d 个 running 状态的沙箱\n", len(sandboxesV2))

	// 批量获取沙箱指标
	if len(sandboxes) > 0 {
		fmt.Println("\n=== 批量获取沙箱指标 ===")
		ids := make([]string, 0, len(sandboxes))
		for _, sb := range sandboxes {
			ids = append(ids, sb.SandboxID)
		}
		metrics, err := c.GetSandboxesMetrics(ctx, &apis.GetSandboxesMetricsParams{
			SandboxIds: ids,
		})
		if err != nil {
			fmt.Printf("获取批量指标失败: %v\n", err)
		} else {
			fmt.Printf("获取到 %d 个沙箱的指标数据\n", len(metrics.Sandboxes))
		}
	}
}
