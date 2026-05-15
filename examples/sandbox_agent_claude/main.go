// Sandbox Claude Code Agent 多轮对话示例。
//
// 演示如何在七牛云沙箱中使用 claude 模板进行多轮上下文延续的 Agent 对话：
// 首轮创建新 session，后续通过 --resume <session_id> 复用上下文。
//
// 模拟场景：与 AI 协作完成一次日志统计需求 —— 本地预生成一份访问日志，通过
// sandbox.Filesystem().Write 上传到沙箱后，与 Agent 多轮讨论并实现 top 10
// 统计。覆盖文件上传、纯对话、工具调用、流式优化、独立验证多个场景。
//
// 环境变量：
//   - QINIU_API_KEY:         七牛沙箱 API Key（必填）
//   - QINIU_SANDBOX_API_URL: 沙箱 API 地址，可选
//   - ANTHROPIC_API_KEY:     Anthropic API Key（必填，注入到沙箱）
//   - ANTHROPIC_BASE_URL:    第三方网关地址，可选
package main

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/qiniu/go-sdk/v7/sandbox"
)

// accessLogJSONL 是预生成的 100 行访问日志，作为示例数据嵌入二进制。
// 数据通过 testdata/access.jsonl 维护，user_id 分布固定（seed=42），便于校对结果。
//
//go:embed testdata/access.jsonl
var accessLogJSONL []byte

// sandboxDataPath 是数据文件在沙箱内的路径，与下面 prompt 中的描述保持一致。
const sandboxDataPath = "/tmp/topn/data.jsonl"

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

	// 2. 上传预生成的访问日志到沙箱，Write 会自动创建父目录。
	info, err := sb.Files().Write(ctx, sandboxDataPath, accessLogJSONL)
	if err != nil {
		log.Printf("上传测试数据失败: %v", err)
		return
	}
	fmt.Printf("已上传 %s (%d bytes)\n", sandboxDataPath, info.Size)

	// 3. 本地预计算 top 10，作为 Agent 输出的对照。
	expected := computeTopN(accessLogJSONL, 10)
	fmt.Println("\n--- 期望 top 10（本地计算）---")
	for _, uc := range expected {
		fmt.Printf("  user_id=%d  count=%d\n", uc.userID, uc.count)
	}

	// 4. 模拟一次"AI 协作开发"会话：问题分析 → 动手实现 → 流式优化。
	//    后续轮次大量依赖前一轮的上下文（设计思路、代码、对应文件路径）。
	turns := []string{
		// Turn 1: 讨论方案（纯对话）
		"我有一批 jsonl 访问日志已经放在沙箱里 /tmp/topn/data.jsonl（约 100 行），" +
			"每行是 JSON 对象，字段包含 user_id (int)、action (string)、timestamp (int64)。" +
			"我需要按 user_id 统计每个用户的 action 总数并输出 top 10。" +
			"计划用 Python 实现（沙箱内 python3 可用）。先说一下你的实现思路，先不要写代码。",

		// Turn 2: 动手实现（工具调用：写文件 + 跑代码）
		"按你刚才的思路，请在 /tmp/topn 下写一个 main.py，读取同目录下的 data.jsonl，" +
			"按 user_id 统计 action 总数并输出 top 10（count 降序，相同时 user_id 升序）。" +
			"完成后运行 main.py 贴出结果。",

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

		// Turn 2/3 结束后列出工作目录并独立运行 main.py，与期望 top 10 对照。
		if i >= 1 {
			listWorkdir(ctx, sb, "/tmp/topn")
			verifyTopN(ctx, sb, "/tmp/topn/main.py")
		}
	}
}

// userCount 是一个 user 的 action 总数。
type userCount struct {
	userID int
	count  int
}

// computeTopN 解析 jsonl 数据并返回按 count 降序、user_id 升序的前 n 名。
func computeTopN(data []byte, n int) []userCount {
	counts := make(map[int]int)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	// 单条日志远小于默认 64KB，无需扩 buffer。
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec struct {
			UserID int `json:"user_id"`
		}
		if err := json.Unmarshal(line, &rec); err != nil {
			continue
		}
		counts[rec.UserID]++
	}
	if err := scanner.Err(); err != nil {
		log.Printf("computeTopN: scanner 提前终止: %v", err)
	}
	out := make([]userCount, 0, len(counts))
	for uid, c := range counts {
		out = append(out, userCount{uid, c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].count != out[j].count {
			return out[i].count > out[j].count
		}
		return out[i].userID < out[j].userID
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// verifyTopN 在沙箱里独立运行 Agent 写的 main.py，便于与期望 top 10 对照。
func verifyTopN(ctx context.Context, sb *sandbox.Sandbox, scriptPath string) {
	fmt.Printf("\n--- 独立运行 %s ---\n", scriptPath)
	res, err := sb.Commands().Run(ctx, "python3 "+shellQuote(scriptPath))
	if err != nil {
		fmt.Printf("运行脚本失败: %v\n", err)
		return
	}
	if res.ExitCode != 0 {
		fmt.Printf("脚本退出码 %d, stderr:\n%s\n", res.ExitCode, res.Stderr)
		return
	}
	if res.Stdout != "" {
		fmt.Print(res.Stdout)
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
		// stderr 数据按 chunk 透传，避免对未完成的行加前缀产生 [stderr] part1[stderr] part2 这类碎片。
		sandbox.WithOnStderr(func(data []byte) {
			fmt.Fprint(os.Stderr, string(data))
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
//
// 并发模型：每次 runTurn 新建独立实例，与其他 turn 完全隔离；本实例的 onStdout
// 仅由对应 Start 调用产生的事件流单 goroutine 顺序回调（参见 SDK 内
// processEventStream）。因此 buf 无需加锁。若未来要在多个 Start 之间共享同一个
// streamProcessor，则需自行同步。
type streamProcessor struct {
	buf       bytes.Buffer
	sessionID string
}

// onStdout 是 Start 的 stdout 回调，由进程事件流单 goroutine 顺序调用，无需加锁。
func (p *streamProcessor) onStdout(data []byte) {
	p.buf.Write(data)
	for {
		buf := p.buf.Bytes()
		idx := bytes.IndexByte(buf, '\n')
		if idx < 0 {
			break
		}
		// onStdout 单 goroutine、handleLine 同步消费 line 后不保留引用，
		// 直接切片 buf 无需额外复制。
		line := buf[:idx]
		p.handleLine(line)
		p.buf.Next(idx + 1)
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
		item, ok := c.(map[string]any)
		if !ok {
			continue
		}
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
		item, ok := c.(map[string]any)
		if !ok {
			continue
		}
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
