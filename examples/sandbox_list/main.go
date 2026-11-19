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

	apiURL := os.Getenv("QINIU_SANDBOX_API_URL")

	c, err := sandbox.NewClient(&sandbox.Config{
		APIKey:   apiKey,
		Endpoint: apiURL,
	})
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 使用同一个 List 方法逐页列出沙箱，保留查询条件并更新游标。
	limit := int32(100)
	params := &sandbox.ListParams{Limit: &limit}
	var items []sandbox.ListedSandbox
	for {
		page, err := c.List(ctx, params)
		if err != nil {
			log.Fatalf("列出沙箱失败: %v", err)
		}
		items = append(items, page.Items...)
		if page.NextToken == "" {
			break
		}
		params.NextToken = &page.NextToken
	}

	fmt.Printf("共 %d 个沙箱:\n", len(items))
	for _, sb := range items {
		fmt.Printf("  - %s (模板: %s)\n", sb.SandboxID, sb.TemplateID)
	}

	// 按状态过滤列出沙箱
	fmt.Println("\n=== 按状态过滤 ===")
	states := []sandbox.SandboxState{sandbox.StateRunning}
	filteredPage, err := c.List(ctx, &sandbox.ListParams{
		State: &states,
	})
	if err != nil {
		log.Fatalf("列出沙箱失败: %v", err)
	}
	fmt.Printf("本页有 %d 个 running 状态的沙箱\n", len(filteredPage.Items))

	// 按一个或多个模板 ID / 名称过滤，最多可传 20 个值。
	fmt.Println("\n=== 按模板过滤 ===")
	templates := []string{"base"}
	templatePage, err := c.List(ctx, &sandbox.ListParams{
		Template: &templates,
	})
	if err != nil {
		log.Fatalf("按模板列出沙箱失败: %v", err)
	}
	fmt.Printf("模板 %v 下本页有 %d 个沙箱\n", templates, len(templatePage.Items))

	// 批量获取沙箱指标
	if len(items) > 0 {
		fmt.Println("\n=== 批量获取沙箱指标 ===")
		ids := make([]string, 0, len(items))
		for _, sb := range items {
			ids = append(ids, sb.SandboxID)
		}
		const maxMetricsIDs = 100
		var count int
		for len(ids) > 0 {
			size := min(len(ids), maxMetricsIDs)
			metrics, err := c.GetSandboxesMetrics(ctx, &sandbox.GetSandboxesMetricsParams{
				SandboxIds: ids[:size],
			})
			if err != nil {
				log.Fatalf("获取批量指标失败: %v", err)
			}
			count += len(metrics.Sandboxes)
			ids = ids[size:]
		}
		fmt.Printf("获取到 %d 个沙箱的指标数据\n", count)
	}
}
