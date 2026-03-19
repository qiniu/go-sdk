package main

import (
	"context"
	"log"
	"os"

	"github.com/qiniu/go-sdk/v7/sandbox"
)

func main() {
	// 确保设置了环境变量 QINIU_API_KEY
	apiKey := os.Getenv("QINIU_API_KEY")
	if apiKey == "" {
		log.Fatal("请设置 QINIU_API_KEY 环境变量")
	}

	apiURL := os.Getenv("QINIU_SANDBOX_API_URL")

	ctx := context.Background()

	// 初始化客户端
	config := &sandbox.Config{
		APIKey:   apiKey,
		Endpoint: apiURL,
	}
	client, err := sandbox.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	log.Println("Creating sandbox with request transforms...")

	// 设置针对特定域名的请求替换规则
	// 这个规则告诉底层网络架构，当请求向 "httpbin.org" 发起时，
	// 如果它是 HTTPS，拦截掉该请求并将 header 中的 "Authorization"
	// 重写为 "Bearer real_xxx"
	hosts := []string{"httpbin.org"}
	headers := map[string]string{
		"Authorization": "Bearer real_xxx",
	}
	queries := map[string]string{
		"api-key": "real-api-key-value",
	}

	// 准备创建参数
	params := sandbox.CreateParams{
		TemplateID: "base-apikey",
		RequestTransforms: &[]sandbox.RequestTransform{
			{
				Conditions: &sandbox.RequestTransformConditions{
					Hosts: &hosts,
				},
				Replacements: &sandbox.RequestTransformReplacements{
					Headers: &headers,
					Queries: &queries,
				},
			},
		},
	}

	// 创建并等待沙箱就绪
	sb, info, err := client.CreateAndWait(ctx, params)
	if err != nil {
		log.Fatalf("Failed to create sandbox: %v", err)
	}
	defer func() {
		log.Println("Killing sandbox...")
		_ = sb.Kill(ctx)
	}()

	log.Printf("Sandbox created successfully! ID: %s, State: %s\n", sb.ID(), info.State)

	// 在沙箱内执行使用假 APIKEY 的请求
	// 沙箱内的程序完全感觉不到外部注入，实际上外层将其拦截并替换为了真实的 Key
	fakeTokenCmd := `curl -sSL -X GET "https://httpbin.org/bearer" -H "accept: application/json" -H "Authorization: Bearer fake_xxx"`
	log.Printf("Executing command in sandbox:\n$ %s\n", fakeTokenCmd)

	result, err := sb.Commands().Run(ctx, fakeTokenCmd)
	if err != nil {
		log.Fatalf("Failed to run command: %v", err)
	}

	// 打印输出，应当能看到 real_xxx 出现在 token 字段中
	log.Println("Command Execution Result:", result.ExitCode)
	if result.Error != "" {
		log.Printf("Error: %s\n", result.Error)
	}
	log.Printf("Stdout: \n%s\n", result.Stdout)
	if result.Stderr != "" {
		log.Printf("Stderr: \n%s\n", result.Stderr)
	}

	// 测试 Query 参数替换
	// 我们在规则中配置了 api-key 的替换。
	// 下面的请求包含 api-key=old-key，应当被替换为 real-api-key-value
	queryCmd := `curl -sSL -X GET "https://httpbin.org/get?api-key=old-key&other=foo"`
	log.Printf("\nExecuting query transform test:\n$ %s\n", queryCmd)
	result, err = sb.Commands().Run(ctx, queryCmd)
	if err != nil {
		log.Fatalf("Failed to run query test command: %v", err)
	}
	log.Printf("Query Test Stdout: \n%s\n", result.Stdout)

	// 测试“仅替换，不新增”逻辑
	// 下面的请求不包含 api-key，因此不应当被注入新的参数
	noQueryCmd := `curl -sSL -X GET "https://httpbin.org/get?other=bar"`
	log.Printf("\nExecuting no-query (no-injection) test:\n$ %s\n", noQueryCmd)
	result, err = sb.Commands().Run(ctx, noQueryCmd)
	if err != nil {
		log.Fatalf("Failed to run no-query test command: %v", err)
	}
	log.Printf("No-Query Test Stdout: \n%s\n", result.Stdout)
}
