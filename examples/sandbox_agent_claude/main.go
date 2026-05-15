// Sandbox Claude Code Agent 多轮对话示例。
//
// 演示如何在七牛云沙箱中使用 claude 模板进行多轮上下文延续的 Agent 对话：
// 首轮创建新 session，后续通过 --resume <session_id> 复用上下文。
//
// 模拟场景：与 AI 协作完成一次日志统计需求 —— 先讨论方案，再生成测试数据 +
// 实现 Go 程序，最后迭代优化为流式处理。覆盖纯对话与工具调用两类场景。
//
// 环境变量：
//   - QINIU_API_KEY:         七牛沙箱 API Key（必填）
//   - QINIU_SANDBOX_API_URL: 沙箱 API 地址，可选
//   - ANTHROPIC_API_KEY:     Anthropic API Key（必填，注入到沙箱）
//   - ANTHROPIC_BASE_URL:    第三方网关地址，可选
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/qiniu/go-sdk/v7/sandbox"
)

func main() {
	apiKey := os.Getenv("QINIU_API_KEY")
	if apiKey == "" {
		log.Fatal("请设置 QINIU_API_KEY 环境变量")
	}
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	if anthropicKey == "" {
		log.Fatal("请设置 ANTHROPIC_API_KEY 环境变量")
	}

	c, err := sandbox.NewClient(&sandbox.Config{
		APIKey:   apiKey,
		Endpoint: os.Getenv("QINIU_SANDBOX_API_URL"),
	})
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	// 1. 创建沙箱：使用预构建的 claude 模板，通过环境变量注入 API Key。
	//    生产场景建议使用密钥注入规则（injection-rule），避免明文环境变量。
	envs := map[string]string{
		"ANTHROPIC_API_KEY": anthropicKey,
	}
	if baseURL := os.Getenv("ANTHROPIC_BASE_URL"); baseURL != "" {
		envs["ANTHROPIC_BASE_URL"] = baseURL
	}

	timeout := int32(1800)
	sb, _, err := c.CreateAndWait(ctx, sandbox.CreateParams{
		TemplateID: "claude",
		Timeout:    &timeout,
		EnvVars:    &envs,
	}, sandbox.WithPollInterval(2*time.Second))
	if err != nil {
		log.Fatalf("创建沙箱失败: %v", err)
	}
	fmt.Printf("沙箱已创建: %s\n", sb.ID())

	defer func() {
		killCtx, killCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer killCancel()
		if err := sb.Kill(killCtx); err != nil {
			log.Printf("终止沙箱失败: %v", err)
		} else {
			fmt.Printf("\n沙箱 %s 已终止\n", sb.ID())
		}
	}()

	// 2. 模拟一次"AI 协作开发"会话：问题分析 → 动手实现 → 流式优化。
	//    后续轮次大量依赖前一轮的上下文（设计思路、代码、测试数据路径）。
	turns := []string{
		// Turn 1: 讨论方案（纯对话）
		"我有一批 jsonl 日志，每行是 JSON 对象，字段包含 user_id (int)、action (string)、" +
			"timestamp (int64)。我需要按 user_id 统计每个用户的 action 总数并输出 top 10。" +
			"计划用 Python 实现（沙箱内 python3 可用）。先说一下你的实现思路，先不要写代码。",

		// Turn 2: 动手实现（工具调用：写文件、生成数据、跑代码）
		"按你刚才的思路来。请在 /tmp/topn 下：先用 gen.py 生成 1000 行测试 jsonl 数据" +
			"（user_id 取 0-49 随机，action 从 ['login','view','click','purchase'] 随机），" +
			"输出到 /tmp/topn/data.jsonl。然后写 main.py 读取该文件并输出 top 10。" +
			"最后运行 main.py 贴出结果。",

		// Turn 3: 迭代优化（依赖前一轮代码）
		"很好。现在请把 main.py 改成严格流式处理：不要 readlines() 或一次读全文件，" +
			"改成 with open(...) as f: for line in f 逐行扫描，使内存占用与文件大小解耦。" +
			"改完后再跑一次，确认输出的 top 10 与上一版一致。",
	}

	var sessionID string
	for i, prompt := range turns {
		fmt.Printf("\n========== Turn %d/%d ==========\n", i+1, len(turns))
		fmt.Printf("USER: %s\n\n", prompt)

		newSessionID, err := runTurn(ctx, sb, prompt, sessionID)
		if err != nil {
			// 用 return 而非 log.Fatal，确保 defer 中的 sb.Kill 能执行，避免沙箱泄漏。
			log.Printf("Turn %d 执行失败: %v", i+1, err)
			return
		}
		if newSessionID != "" {
			sessionID = newSessionID
		}
		fmt.Printf("\n(session_id: %s)\n", sessionID)

		// Turn 2/3 结束后列出工作目录文件，观察 Agent 实际产物。
		if i >= 1 {
			listWorkdir(ctx, sb, "/tmp/topn")
		}
	}
}

// listWorkdir 打印沙箱内指定目录下的文件列表。
func listWorkdir(ctx context.Context, sb *sandbox.Sandbox, dir string) {
	fmt.Printf("\n--- 沙箱目录 %s ---\n", dir)
	res, err := sb.Commands().Run(ctx, "ls -la "+shellQuote(dir))
	if err != nil {
		fmt.Printf("列出文件失败: %v\n", err)
		return
	}
	if res.Stdout != "" {
		fmt.Print(res.Stdout)
	}
	if res.Stderr != "" {
		fmt.Fprint(os.Stderr, res.Stderr)
	}
}

// runTurn 在沙箱内执行一轮 claude 对话。
// resumeID 为空时创建新会话；非空时通过 --resume 延续上下文。
// 返回本轮捕获到的 session_id，供下一轮复用。
func runTurn(ctx context.Context, sb *sandbox.Sandbox, prompt, resumeID string) (string, error) {
	args := []string{
		"-p", shellQuote(prompt),
		"--output-format", "stream-json",
		"--verbose",
		"--dangerously-skip-permissions",
	}
	if resumeID != "" {
		// resumeID 也经 shell 拼接执行，统一 shellQuote 防御非预期字符。
		args = append(args, "--resume", shellQuote(resumeID))
	}
	cmd := "claude " + strings.Join(args, " ")

	p := &streamProcessor{}
	handle, err := sb.Commands().Start(ctx, cmd,
		sandbox.WithOnStdout(p.onStdout),
		sandbox.WithOnStderr(func(data []byte) {
			fmt.Fprintf(os.Stderr, "[stderr] %s", data)
		}),
	)
	if err != nil {
		return "", fmt.Errorf("启动命令: %w", err)
	}

	result, err := handle.Wait()
	if err != nil {
		return "", fmt.Errorf("等待命令: %w", err)
	}

	// flush 残留缓冲（若没有以 '\n' 收尾）。
	if rest := bytes.TrimSpace(p.buf.Bytes()); len(rest) > 0 {
		p.handleLine(rest)
	}

	if result.ExitCode != 0 {
		return p.sessionID, fmt.Errorf("命令退出码 %d, error: %s", result.ExitCode, result.Error)
	}
	return p.sessionID, nil
}

// streamProcessor 维护单次 claude 调用的 JSONL 解析状态：
// stdout 是字节流，需按 '\n' 拆分为 JSONL 后再解析；同时捕获 session_id 供后续轮次使用。
type streamProcessor struct {
	buf       bytes.Buffer
	sessionID string
}

// onStdout 是 Start 的 stdout 回调，由进程事件流单 goroutine 顺序调用，无需加锁。
func (p *streamProcessor) onStdout(data []byte) {
	p.buf.Write(data)
	for {
		idx := bytes.IndexByte(p.buf.Bytes(), '\n')
		if idx < 0 {
			break
		}
		line := append([]byte(nil), p.buf.Bytes()[:idx]...)
		p.buf.Next(idx + 1)
		p.handleLine(line)
	}
}

func (p *streamProcessor) handleLine(line []byte) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return
	}
	var evt map[string]any
	if err := json.Unmarshal(line, &evt); err != nil {
		// 非 JSON 行（例如 CLI 早期提示），原样输出便于排查。
		fmt.Printf("[raw] %s\n", line)
		return
	}
	// 多数事件都带 session_id，取最后一次见到的值（init/result 一致）。
	if sid, ok := evt["session_id"].(string); ok && sid != "" {
		p.sessionID = sid
	}
	printEvent(evt)
}

// printEvent 打印 Claude Code 流式事件的关键字段。
// 事件类型参见 https://developer.qiniu.com/las/13452/sandbox-agent-claude-code
func printEvent(evt map[string]any) {
	t, _ := evt["type"].(string)
	switch t {
	case "system":
		sub, _ := evt["subtype"].(string)
		session, _ := evt["session_id"].(string)
		model, _ := evt["model"].(string)
		fmt.Printf("[system/%s] session=%s model=%s\n", sub, session, model)

	case "assistant":
		if msg, ok := evt["message"].(map[string]any); ok {
			printAssistantMessage(msg)
		}

	case "user":
		if msg, ok := evt["message"].(map[string]any); ok {
			printUserMessage(msg)
		}

	case "result":
		sub, _ := evt["subtype"].(string)
		isErr, _ := evt["is_error"].(bool)
		dur, _ := evt["duration_ms"].(float64)
		cost, _ := evt["total_cost_usd"].(float64)
		text, _ := evt["result"].(string)
		fmt.Printf("[result/%s] duration=%.0fms error=%v cost=$%.4f\n", sub, dur, isErr, cost)
		if text != "" {
			fmt.Printf("        最终结果: %s\n", text)
		}

	default:
		// 未知类型，原样打印以便后续扩展。
		b, _ := json.Marshal(evt)
		fmt.Printf("[%s] %s\n", t, b)
	}
}

// printAssistantMessage 打印助手消息，区分纯文本与工具调用。
func printAssistantMessage(msg map[string]any) {
	contents, _ := msg["content"].([]any)
	var text strings.Builder
	for _, c := range contents {
		item, _ := c.(map[string]any)
		switch item["type"] {
		case "text":
			if s, ok := item["text"].(string); ok {
				text.WriteString(s)
			}
		case "tool_use":
			name, _ := item["name"].(string)
			input, _ := json.Marshal(item["input"])
			fmt.Printf("[assistant/tool_use] %s input=%s\n", name, input)
		}
	}
	if text.Len() > 0 {
		fmt.Printf("[assistant] %s\n", text.String())
	}
}

// printUserMessage 打印用户消息，主要是工具结果回灌。
func printUserMessage(msg map[string]any) {
	contents, _ := msg["content"].([]any)
	for _, c := range contents {
		item, _ := c.(map[string]any)
		if item["type"] != "tool_result" {
			continue
		}
		id, _ := item["tool_use_id"].(string)
		isErr, _ := item["is_error"].(bool)
		fmt.Printf("[user/tool_result] id=%s error=%v\n", id, isErr)
	}
}

// shellQuote 将字符串包装为单引号形式的 shell 字面量。
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
