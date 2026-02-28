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

	// 1. 创建沙箱
	templateID := "base"
	timeout := int32(120)
	sb, _, err := c.CreateAndWait(ctx, apis.CreateSandboxJSONRequestBody{
		TemplateID: templateID,
		Timeout:    &timeout,
	}, 2*time.Second)
	if err != nil {
		log.Fatalf("创建沙箱失败: %v", err)
	}
	fmt.Printf("沙箱已就绪: %s\n", sb.SandboxID)
	defer func() {
		_ = sb.Kill(context.Background())
		fmt.Println("沙箱已终止")
	}()

	// 等待一段时间让沙箱产生指标数据
	fmt.Println("等待 5 秒收集指标...")
	time.Sleep(5 * time.Second)

	// 2. 获取指标
	metrics, err := sb.GetMetrics(ctx, nil)
	if err != nil {
		log.Fatalf("获取指标失败: %v", err)
	}
	fmt.Printf("指标数据 (%d 条):\n", len(metrics))
	for _, m := range metrics {
		fmt.Printf("  - 时间: %s, CPU: %.1f%%, 内存: %d/%d bytes\n",
			m.Timestamp.Format(time.RFC3339), m.CPUUsedPct, m.MemUsed, m.MemTotal)
	}

	// 3. 获取日志
	logs, err := sb.GetLogs(ctx, nil)
	if err != nil {
		fmt.Printf("\n获取日志失败（服务端可能暂不支持）: %v\n", err)
	} else {
		fmt.Printf("\n日志 (%d 条):\n", len(logs.Logs))
		for _, entry := range logs.Logs {
			fmt.Printf("  [%s] %s\n", entry.Timestamp.Format(time.RFC3339), entry.Line)
		}
	}
}
