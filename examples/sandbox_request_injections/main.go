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

	log.Println("Creating sandbox with request injections...")

	// 设置针对特定域名的请求注入规则
	// 这个规则告诉底层网络架构，当请求向 "httpbin.org" 发起时，
	// 如果它是 HTTPS，拦截掉该请求并将 header 中的 "Authorization"
	// 重写为 "Bearer real_xxx"
	headers := map[string]string{
		"Authorization": "Bearer real_xxx",
	}

	// 准备创建参数（HTTP 自定义注入）
	params := sandbox.CreateParams{
		TemplateID: "base",
		Injections: &[]sandbox.SandboxInjectionSpec{
			{
				HTTP: &sandbox.HTTPInjection{
					BaseURL: "https://httpbin.org",
					Headers: &headers,
				},
			},
		},
	}

	// 也可以使用已知 API 协议的快捷注入（OpenAI、Anthropic、Gemini）
	// openaiKey := "sk-real-openai-key"
	// params := sandbox.CreateParams{
	// 	TemplateID: "base",
	// 	Injections: &[]sandbox.SandboxInjectionSpec{
	// 		{
	// 			OpenAI: &sandbox.OpenAIInjection{
	// 				APIKey: &openaiKey,
	// 				// BaseURL 可选，默认 api.openai.com
	// 			},
	// 		},
	// 	},
	// }

	// 也可以通过注入规则 ID 引用已保存的注入规则
	// injectionRuleID := "<injection-rule-id>"
	// params := sandbox.CreateParams{
	// 	TemplateID: "base",
	// 	Injections: &[]sandbox.SandboxInjectionSpec{
	// 		{ByID: &injectionRuleID},
	// 	},
	// }

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
}
