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

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 1. 创建注入规则
	fmt.Println("=== 创建注入规则 ===")
	hosts := []string{"httpbin.org"}
	headers := map[string]string{
		"Authorization": "Bearer real_token",
	}
	queries := map[string]string{
		"api-key": "real-api-key-value",
	}
	rule, err := c.CreateInjectionRule(ctx, sandbox.CreateInjectionRuleParams{
		Name: "example-rule",
		Conditions: &sandbox.RequestInjectionConditions{
			Hosts: &hosts,
		},
		Injections: &sandbox.RequestInjections{
			Headers: &headers,
			Queries: &queries,
		},
	})
	if err != nil {
		log.Fatalf("创建注入规则失败: %v", err)
	}
	ruleID := rule.RuleID
	fmt.Printf("规则已创建: ID=%s, 名称=%s\n", ruleID, rule.Name)
	fmt.Printf("  创建时间: %s\n", rule.CreatedAt.Format(time.RFC3339))

	// 确保测试结束时清理
	defer func() {
		fmt.Println("\n=== 删除注入规则 ===")
		if err := c.DeleteInjectionRule(context.Background(), ruleID); err != nil {
			fmt.Printf("删除注入规则失败: %v\n", err)
		} else {
			fmt.Printf("规则 %s 已删除\n", ruleID)
		}
	}()

	// 2. 获取注入规则详情
	fmt.Println("\n=== 获取注入规则详情 ===")
	detail, err := c.GetInjectionRule(ctx, ruleID)
	if err != nil {
		log.Fatalf("获取注入规则失败: %v", err)
	}
	fmt.Printf("规则: ID=%s, 名称=%s\n", detail.RuleID, detail.Name)
	if detail.Conditions != nil && detail.Conditions.Hosts != nil {
		fmt.Printf("  匹配域名: %v\n", *detail.Conditions.Hosts)
	}
	if detail.Injections != nil {
		if detail.Injections.Headers != nil {
			fmt.Printf("  注入 Headers: %v\n", *detail.Injections.Headers)
		}
		if detail.Injections.Queries != nil {
			fmt.Printf("  注入 Queries: %v\n", *detail.Injections.Queries)
		}
	}

	// 3. 更新注入规则
	fmt.Println("\n=== 更新注入规则 ===")
	newName := "example-rule-updated"
	newHeaders := map[string]string{
		"Authorization": "Bearer updated_token",
		"X-Custom":      "custom-value",
	}
	updated, err := c.UpdateInjectionRule(ctx, ruleID, sandbox.UpdateInjectionRuleParams{
		Name: &newName,
		Injections: &sandbox.RequestInjections{
			Headers: &newHeaders,
		},
	})
	if err != nil {
		log.Fatalf("更新注入规则失败: %v", err)
	}
	fmt.Printf("规则已更新: 名称=%s, 更新时间=%s\n", updated.Name, updated.UpdatedAt.Format(time.RFC3339))

	// 4. 列出所有注入规则
	fmt.Println("\n=== 列出所有注入规则 ===")
	rules, err := c.ListInjectionRules(ctx)
	if err != nil {
		log.Fatalf("列出注入规则失败: %v", err)
	}
	fmt.Printf("共 %d 条规则:\n", len(rules))
	for _, r := range rules {
		fmt.Printf("  - ID=%s, 名称=%s, 创建时间=%s\n",
			r.RuleID, r.Name, r.CreatedAt.Format(time.RFC3339))
	}

	// 5. 使用注入规则创建沙箱
	fmt.Println("\n=== 使用注入规则创建沙箱 ===")
	injectionIDs := []string{ruleID}
	sb, info, err := c.CreateAndWait(ctx, sandbox.CreateParams{
		TemplateID:          "base",
		RequestInjectionIds: &injectionIDs,
	})
	if err != nil {
		log.Fatalf("创建沙箱失败: %v", err)
	}
	defer func() {
		fmt.Println("\n=== 销毁沙箱 ===")
		_ = sb.Kill(context.Background())
		fmt.Println("沙箱已销毁")
	}()
	fmt.Printf("沙箱已创建: ID=%s, 状态=%s\n", sb.ID(), info.State)

	// 在沙箱内验证注入规则生效
	cmd := `curl -sSL -X GET "https://httpbin.org/bearer" -H "accept: application/json" -H "Authorization: Bearer fake_token"`
	fmt.Printf("\n执行命令:\n$ %s\n", cmd)
	result, err := sb.Commands().Run(ctx, cmd)
	if err != nil {
		log.Fatalf("执行命令失败: %v", err)
	}
	fmt.Printf("退出码: %d\n", result.ExitCode)
	fmt.Printf("输出:\n%s\n", result.Stdout)

	// 6. 删除注入规则（通过 defer 执行）
}
