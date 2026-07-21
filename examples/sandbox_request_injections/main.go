package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

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
	if err := run(ctx, client); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, client *sandbox.Client) error {
	log.Println("Creating sandbox with request injections...")

	// 设置针对特定域名的请求注入规则
	// 这个规则告诉底层网络架构，当请求向 "httpbingo.org" 发起时，
	// 如果它是 HTTPS，拦截掉该请求并将 header 中的 "Authorization"
	// 重写为 "Bearer real_xxx"
	headers := map[string]string{
		"Authorization": "Bearer real_xxx",
	}
	ifHeaders := map[string]string{
		"X-Injection-Scope": "demo",
	}
	ifQueries := map[string]string{
		"inject": "true",
	}

	// 准备创建参数（HTTP 自定义注入）。如果设置了 QINIU_GITHUB_TOKEN，
	// 同时演示 GitHub 凭证注入和运行时 token 更新。
	injections := []sandbox.SandboxInjectionSpec{
		{
			HTTP: &sandbox.HTTPInjection{
				BaseURL:   "https://httpbingo.org",
				Headers:   &headers,
				IfHeaders: &ifHeaders,
				IfQueries: &ifQueries,
			},
		},
	}
	githubToken := os.Getenv("QINIU_GITHUB_TOKEN")
	if githubToken != "" {
		injections = append(injections, sandbox.SandboxInjectionSpec{
			Github: &sandbox.GithubInjection{Token: &githubToken},
		})
	}

	params := sandbox.CreateParams{
		TemplateID: "base",
		Injections: &injections,
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
		return fmt.Errorf("failed to create sandbox: %w", err)
	}
	defer func() {
		log.Println("Killing sandbox...")
		if err := sb.Kill(ctx); err != nil {
			log.Printf("Failed to kill sandbox: %v", err)
		}
	}()

	log.Printf("Sandbox created successfully! ID: %s, State: %s\n", sb.ID(), info.State)

	// 查询当前运行时注入。敏感字段由服务端脱敏，返回值不能直接用于更新。
	current, err := sb.GetInjections(ctx)
	if err != nil {
		return fmt.Errorf("failed to get runtime injections: %w", err)
	}
	logMaskedInjections("Current runtime injections", current)

	// 在沙箱内执行使用假 APIKEY 的请求
	// 沙箱内的程序完全感觉不到外部注入；只有请求同时满足 IfHeaders / IfQueries
	// 匹配条件时，外层才会拦截并替换为真实的 Key。
	fakeTokenCmd := `curl -sSL -X GET "https://httpbingo.org/bearer?inject=true" -H "accept: application/json" -H "Authorization: Bearer fake_xxx" -H "X-Injection-Scope: demo"`
	if err := runBearerCheck(ctx, sb, fakeTokenCmd, "real_xxx"); err != nil {
		return err
	}

	// 替换运行中沙箱的全部注入规则。更新接口需要重新提供真实敏感值。
	updatedHeaders := map[string]string{
		"Authorization": "Bearer updated_xxx",
	}
	updatedInjections := []sandbox.SandboxInjectionSpec{
		{
			HTTP: &sandbox.HTTPInjection{
				BaseURL:   "https://httpbingo.org",
				Headers:   &updatedHeaders,
				IfHeaders: &ifHeaders,
				IfQueries: &ifQueries,
			},
		},
	}
	if githubToken != "" {
		updatedInjections = append(updatedInjections, sandbox.SandboxInjectionSpec{
			Github: &sandbox.GithubInjection{Token: &githubToken},
		})
	}
	if err := sb.UpdateInjections(ctx, updatedInjections); err != nil {
		return fmt.Errorf("failed to update runtime injections: %w", err)
	}
	current, err = sb.GetInjections(ctx)
	if err != nil {
		return fmt.Errorf("failed to get updated runtime injections: %w", err)
	}
	logMaskedInjections("Updated runtime injections", current)
	if err := runBearerCheck(ctx, sb, fakeTokenCmd, "updated_xxx"); err != nil {
		return err
	}

	// 已配置 GitHub 注入时，可独立轮换 GitHub token，无需替换其他注入规则。
	if updatedGitHubToken := os.Getenv("QINIU_GITHUB_TOKEN_UPDATED"); updatedGitHubToken != "" {
		if githubToken == "" {
			return fmt.Errorf("设置 QINIU_GITHUB_TOKEN_UPDATED 时也必须设置 QINIU_GITHUB_TOKEN")
		}
		if err := sb.UpdateGitHubToken(ctx, updatedGitHubToken); err != nil {
			return fmt.Errorf("failed to update GitHub token: %w", err)
		}
		log.Println("GitHub token updated successfully")
	}
	return nil
}

func runBearerCheck(ctx context.Context, sb *sandbox.Sandbox, command, expectedToken string) error {
	log.Printf("Executing command in sandbox:\n$ %s\n", command)
	result, err := sb.Commands().Run(ctx, command)
	if err != nil {
		return fmt.Errorf("failed to run command: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("command failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, `"token": "`+expectedToken+`"`) {
		return fmt.Errorf("request injection did not replace the bearer token, stdout: %s", result.Stdout)
	}
	log.Printf("Request injection verified with token %q", expectedToken)
	return nil
}

func logMaskedInjections(label string, injections []sandbox.MaskedSandboxInjection) {
	log.Printf("%s (%d):", label, len(injections))
	for i, injection := range injections {
		switch {
		case injection.ByID != nil:
			log.Printf("  %d. rule ID: %s", i+1, *injection.ByID)
		case injection.OpenAI != nil:
			log.Printf("  %d. OpenAI injection", i+1)
		case injection.Anthropic != nil:
			log.Printf("  %d. Anthropic injection", i+1)
		case injection.Gemini != nil:
			log.Printf("  %d. Gemini injection", i+1)
		case injection.Qiniu != nil:
			log.Printf("  %d. Qiniu injection", i+1)
		case injection.Github != nil:
			log.Printf("  %d. GitHub injection", i+1)
		case injection.HTTP != nil:
			log.Printf("  %d. HTTP injection for %s", i+1, injection.HTTP.BaseURL)
		default:
			log.Printf("  %d. unknown injection", i+1)
		}
	}
}
