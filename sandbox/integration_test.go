//go:build integration

package sandbox

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/qiniu/go-sdk/v7/sandbox/apis"
)

// testClient 从环境变量创建集成测试用的客户端。
func testClient(t *testing.T) *Client {
	t.Helper()

	apiKey := os.Getenv("Qiniu_API_KEY")
	apiURL := os.Getenv("Qiniu_SANDBOX_API_URL")
	if apiKey == "" {
		t.Fatal("需要设置 Qiniu_API_KEY 环境变量")
	}

	c, err := NewClient(&Config{
		APIKey:   apiKey,
		Endpoint: apiURL,
	})
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	return c
}

func TestIntegrationHealthCheck(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck 失败: %v", err)
	}
	t.Log("HealthCheck 通过")
}

func TestIntegrationListTemplates(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	templates, err := c.ListTemplates(ctx, nil)
	if err != nil {
		t.Fatalf("ListTemplates 失败: %v", err)
	}
	t.Logf("共 %d 个模板", len(templates))
	for _, tmpl := range templates {
		t.Logf("  - %s (buildStatus=%s)", tmpl.TemplateID, tmpl.BuildStatus)
	}
}

func TestIntegrationListSandboxes(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sandboxes, err := c.List(ctx, nil)
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	t.Logf("共 %d 个沙箱", len(sandboxes))
	for _, sb := range sandboxes {
		t.Logf("  - %s (template=%s)", sb.SandboxID, sb.TemplateID)
	}
}

func TestIntegrationSandboxLifecycle(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// 1. 获取可用模板
	templates, err := c.ListTemplates(ctx, nil)
	if err != nil {
		t.Fatalf("ListTemplates 失败: %v", err)
	}

	var templateID string
	for _, tmpl := range templates {
		if tmpl.BuildStatus == apis.TemplateBuildStatusReady || tmpl.BuildStatus == TemplateBuildStatusUploaded {
			templateID = tmpl.TemplateID
			break
		}
	}
	if templateID == "" {
		t.Skip("没有可用模板，跳过生命周期测试")
	}
	t.Logf("使用模板: %s", templateID)

	// 2. 创建沙箱并等待就绪
	timeout := int32(60)
	sb, info, err := c.CreateAndWait(ctx, apis.CreateSandboxJSONRequestBody{
		TemplateID: templateID,
		Timeout:    &timeout,
	}, 2*time.Second)
	if err != nil {
		t.Fatalf("CreateAndWait 失败: %v", err)
	}
	t.Logf("沙箱已创建: %s (state=%s)", sb.SandboxID, info.State)

	// 确保测试结束时清理沙箱
	killed := false
	defer func() {
		if killed {
			return
		}
		killCtx, killCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer killCancel()
		if err := sb.Kill(killCtx); err != nil {
			t.Logf("清理沙箱 %s 失败: %v", sb.SandboxID, err)
		} else {
			t.Logf("沙箱 %s 已清理", sb.SandboxID)
		}
	}()

	// 3. 检查运行状态
	running, err := sb.IsRunning(ctx)
	if err != nil {
		t.Fatalf("IsRunning 失败: %v", err)
	}
	if !running {
		t.Fatal("沙箱应处于运行状态")
	}

	// 4. 获取详细信息
	detail, err := sb.GetInfo(ctx)
	if err != nil {
		t.Fatalf("GetInfo 失败: %v", err)
	}
	t.Logf("沙箱详情: state=%s, templateID=%s, cpuCount=%d, memoryMB=%d",
		detail.State, detail.TemplateID, detail.CPUCount, detail.MemoryMB)

	// 5. 更新超时时间
	if err := sb.SetTimeout(ctx, 120*time.Second); err != nil {
		t.Fatalf("SetTimeout 失败: %v", err)
	}
	t.Log("超时时间已更新为 120s")

	// 6. 在沙箱列表中确认可见
	sandboxes, err := c.List(ctx, nil)
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	found := false
	for _, s := range sandboxes {
		if s.SandboxID == sb.SandboxID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("新创建的沙箱未出现在列表中")
	}

	// 7. 终止沙箱
	if err := sb.Kill(ctx); err != nil {
		t.Fatalf("Kill 失败: %v", err)
	}
	killed = true
	t.Log("沙箱已终止")
}

// createTestSandbox 创建一个用于 envd 集成测试的沙箱并等待就绪。
func createTestSandbox(t *testing.T, c *Client, ctx context.Context) *Sandbox {
	t.Helper()

	templates, err := c.ListTemplates(ctx, nil)
	if err != nil {
		t.Fatalf("ListTemplates 失败: %v", err)
	}

	var templateID string
	for _, tmpl := range templates {
		if tmpl.BuildStatus == apis.TemplateBuildStatusReady || tmpl.BuildStatus == TemplateBuildStatusUploaded {
			templateID = tmpl.TemplateID
			break
		}
	}
	if templateID == "" {
		t.Skip("没有可用模板，跳过测试")
	}

	timeout := int32(120)
	sb, _, err := c.CreateAndWait(ctx, apis.CreateSandboxJSONRequestBody{
		TemplateID: templateID,
		Timeout:    &timeout,
	}, 2*time.Second)
	if err != nil {
		t.Fatalf("CreateAndWait 失败: %v", err)
	}

	t.Cleanup(func() {
		killCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := sb.Kill(killCtx); err != nil {
			t.Logf("清理沙箱 %s 失败: %v", sb.SandboxID, err)
		}
	})

	return sb
}

func TestIntegrationFilesWriteRead(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("沙箱: %s", sb.SandboxID)

	// 写入文件
	content := []byte("hello sandbox\n")
	_, err := sb.Files().Write(ctx, "/tmp/test-file.txt", content)
	if err != nil {
		t.Fatalf("Write 失败: %v", err)
	}
	t.Log("文件写入成功")

	// 读取文件
	got, err := sb.Files().Read(ctx, "/tmp/test-file.txt")
	if err != nil {
		t.Fatalf("Read 失败: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("文件内容不匹配: got %q, want %q", string(got), string(content))
	}
	t.Log("文件读取内容一致")
}

func TestIntegrationCommandsRun(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("沙箱: %s", sb.SandboxID)

	// 执行简单命令
	result, err := sb.Commands().Run(ctx, "echo hello world")
	if err != nil {
		t.Fatalf("Run 失败: %v", err)
	}
	t.Logf("命令结果: exitCode=%d, stdout=%q, stderr=%q", result.ExitCode, result.Stdout, result.Stderr)
	if result.ExitCode != 0 {
		t.Fatalf("命令退出码 %d，期望 0", result.ExitCode)
	}
	if result.Stdout != "hello world\n" {
		t.Fatalf("stdout = %q, want %q", result.Stdout, "hello world\n")
	}
}

func TestIntegrationGetHost(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)

	host := sb.GetHost(8080)
	if host == "" {
		t.Fatal("GetHost 返回空字符串")
	}
	t.Logf("GetHost(8080) = %s", host)

	// 验证格式: {port}-{sandboxID}.{domain}
	expected := "8080-" + sb.SandboxID
	if len(host) < len(expected) || host[:len(expected)] != expected {
		t.Fatalf("GetHost 格式不符: got %q, want prefix %q", host, expected)
	}
}

func TestIntegrationFilesystemOperations(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("沙箱: %s", sb.SandboxID)

	fs := sb.Files()

	// 创建目录
	dirInfo, err := fs.MakeDir(ctx, "/tmp/test-dir")
	if err != nil {
		t.Fatalf("MakeDir 失败: %v", err)
	}
	t.Logf("创建目录: %s (type=%s)", dirInfo.Path, dirInfo.Type)

	// 写入文件
	_, err = fs.Write(ctx, "/tmp/test-dir/hello.txt", []byte("hello"))
	if err != nil {
		t.Fatalf("Write 失败: %v", err)
	}

	// 列目录
	entries, err := fs.List(ctx, "/tmp/test-dir")
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	t.Logf("目录内容: %d 项", len(entries))

	// 文件存在
	exists, err := fs.Exists(ctx, "/tmp/test-dir/hello.txt")
	if err != nil {
		t.Fatalf("Exists 失败: %v", err)
	}
	if !exists {
		t.Fatal("文件应该存在")
	}

	// 获取文件信息
	info, err := fs.GetInfo(ctx, "/tmp/test-dir/hello.txt")
	if err != nil {
		t.Fatalf("GetInfo 失败: %v", err)
	}
	t.Logf("文件信息: name=%s, size=%d, type=%s", info.Name, info.Size, info.Type)

	// 重命名
	newInfo, err := fs.Rename(ctx, "/tmp/test-dir/hello.txt", "/tmp/test-dir/renamed.txt")
	if err != nil {
		t.Fatalf("Rename 失败: %v", err)
	}
	t.Logf("重命名: %s -> %s", "/tmp/test-dir/hello.txt", newInfo.Path)

	// 删除
	if err := fs.Remove(ctx, "/tmp/test-dir/renamed.txt"); err != nil {
		t.Fatalf("Remove 失败: %v", err)
	}
	t.Log("文件已删除")

	// 验证文件不存在
	exists, err = fs.Exists(ctx, "/tmp/test-dir/renamed.txt")
	if err != nil {
		t.Fatalf("Exists 失败: %v", err)
	}
	if exists {
		t.Fatal("文件应已不存在")
	}
}

func TestIntegrationUploadDownload(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("沙箱: %s", sb.SandboxID)

	// Upload
	content := []byte("upload test content\n")
	if err := sb.Upload(ctx, "/tmp/uploaded.txt", content); err != nil {
		t.Fatalf("Upload 失败: %v", err)
	}
	t.Log("文件上传成功")

	// Download
	got, err := sb.Download(ctx, "/tmp/uploaded.txt")
	if err != nil {
		t.Fatalf("Download 失败: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("文件内容不匹配: got %q, want %q", string(got), string(content))
	}
	t.Log("文件下载内容一致")
}

func TestIntegrationWriteFiles(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("沙箱: %s", sb.SandboxID)

	files := []WriteEntry{
		{Path: "/tmp/batch-1.txt", Data: []byte("content one")},
		{Path: "/tmp/batch-2.txt", Data: []byte("content two")},
		{Path: "/tmp/batch-3.txt", Data: []byte("content three")},
	}

	infos, err := sb.Files().WriteFiles(ctx, files)
	if err != nil {
		t.Fatalf("WriteFiles 失败: %v", err)
	}
	if len(infos) != 3 {
		t.Fatalf("WriteFiles 返回 %d 个结果，期望 3", len(infos))
	}

	// 逐个读回验证内容
	for i, f := range files {
		got, err := sb.Files().Read(ctx, f.Path)
		if err != nil {
			t.Fatalf("Read %s 失败: %v", f.Path, err)
		}
		if string(got) != string(f.Data) {
			t.Fatalf("文件 %s 内容不匹配: got %q, want %q", f.Path, string(got), string(f.Data))
		}
		t.Logf("文件 %d (%s) 验证通过: name=%s, size=%d", i, f.Path, infos[i].Name, infos[i].Size)
	}
}

func TestIntegrationReadText(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("沙箱: %s", sb.SandboxID)

	content := "hello read text\n"
	_, err := sb.Files().Write(ctx, "/tmp/readtext.txt", []byte(content))
	if err != nil {
		t.Fatalf("Write 失败: %v", err)
	}

	got, err := sb.Files().ReadText(ctx, "/tmp/readtext.txt")
	if err != nil {
		t.Fatalf("ReadText 失败: %v", err)
	}
	if got != content {
		t.Fatalf("ReadText 内容不匹配: got %q, want %q", got, content)
	}
	t.Log("ReadText 验证通过")
}

func TestIntegrationReadStream(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("沙箱: %s", sb.SandboxID)

	content := []byte("hello read stream\n")
	_, err := sb.Files().Write(ctx, "/tmp/readstream.txt", content)
	if err != nil {
		t.Fatalf("Write 失败: %v", err)
	}

	rc, err := sb.Files().ReadStream(ctx, "/tmp/readstream.txt")
	if err != nil {
		t.Fatalf("ReadStream 失败: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("读取流失败: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("ReadStream 内容不匹配: got %q, want %q", string(got), string(content))
	}
	t.Log("ReadStream 验证通过")
}

// --- Commands 异步执行与进程管理 ---

func TestIntegrationCommandsStartWait(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("沙箱: %s", sb.SandboxID)

	handle, err := sb.Commands().Start(ctx, "sleep 1 && echo done")
	if err != nil {
		t.Fatalf("Start 失败: %v", err)
	}

	result, err := handle.Wait()
	if err != nil {
		t.Fatalf("Wait 失败: %v", err)
	}

	if handle.PID == 0 {
		t.Fatal("PID 应大于 0")
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", result.ExitCode)
	}
	if !strings.Contains(result.Stdout, "done") {
		t.Fatalf("Stdout = %q, 应包含 'done'", result.Stdout)
	}
	t.Logf("Start/Wait 验证通过: PID=%d, ExitCode=%d, Stdout=%q", handle.PID, result.ExitCode, result.Stdout)
}

func TestIntegrationCommandsKill(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("沙箱: %s", sb.SandboxID)

	handle, err := sb.Commands().Start(ctx, "sleep 300")
	if err != nil {
		t.Fatalf("Start 失败: %v", err)
	}

	// 等待 PID 被设置
	if _, err := handle.WaitPID(ctx); err != nil {
		t.Fatalf("WaitPID 失败: %v", err)
	}
	t.Logf("进程已启动: PID=%d", handle.PID)

	// Kill 进程
	if err := sb.Commands().Kill(ctx, handle.PID); err != nil {
		t.Fatalf("Kill 失败: %v", err)
	}

	result, err := handle.Wait()
	if err != nil {
		t.Fatalf("Wait 失败: %v", err)
	}

	if result.ExitCode == 0 {
		t.Fatal("被 Kill 的进程 ExitCode 不应为 0")
	}
	t.Logf("Kill 验证通过: ExitCode=%d", result.ExitCode)
}

func TestIntegrationCommandsList(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("沙箱: %s", sb.SandboxID)

	handle, err := sb.Commands().Start(ctx, "sleep 300")
	if err != nil {
		t.Fatalf("Start 失败: %v", err)
	}

	// 等待 PID 被设置
	if _, err := handle.WaitPID(ctx); err != nil {
		t.Fatalf("WaitPID 失败: %v", err)
	}
	t.Logf("进程已启动: PID=%d", handle.PID)

	// 列出进程
	infos, err := sb.Commands().List(ctx)
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}

	found := false
	for _, info := range infos {
		if info.PID == handle.PID {
			found = true
			t.Logf("找到进程: PID=%d, Cmd=%s, Args=%v", info.PID, info.Cmd, info.Args)
			break
		}
	}
	if !found {
		t.Fatalf("进程列表中未找到 PID=%d，共 %d 个进程", handle.PID, len(infos))
	}

	// 清理
	_ = handle.Kill(ctx)
	_, _ = handle.Wait()
	t.Log("List 验证通过")
}

func TestIntegrationCommandsSendStdin(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("沙箱: %s", sb.SandboxID)

	// 启动 sleep 长命令（stdin 默认禁用，SendStdin 验证 RPC 调用不报错即可）
	handle, err := sb.Commands().Start(ctx, "sleep 300")
	if err != nil {
		t.Fatalf("Start 失败: %v", err)
	}

	// 等待 PID 被设置
	if _, err := handle.WaitPID(ctx); err != nil {
		t.Fatalf("WaitPID 失败: %v", err)
	}

	// 发送 stdin（stdin 未启用，服务端会返回错误，验证错误信息符合预期）
	err = sb.Commands().SendStdin(ctx, handle.PID, []byte("hello\n"))
	if err != nil {
		// stdin 未启用时，服务端返回 "stdin not enabled" 错误是预期行为
		if strings.Contains(err.Error(), "stdin not enabled") {
			t.Logf("SendStdin 返回预期错误: %v", err)
		} else {
			t.Fatalf("SendStdin 失败（非预期错误）: %v", err)
		}
	} else {
		t.Log("SendStdin RPC 调用成功（数据可能被丢弃）")
	}

	// 清理
	_ = handle.Kill(ctx)
	_, _ = handle.Wait()
	t.Log("SendStdin 验证通过")
}

func TestIntegrationCommandsWithCallbacks(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("沙箱: %s", sb.SandboxID)

	var (
		mu        sync.Mutex
		gotStdout []byte
		gotStderr []byte
	)

	result, err := sb.Commands().Run(ctx, "echo out && echo err >&2",
		WithOnStdout(func(data []byte) {
			mu.Lock()
			defer mu.Unlock()
			gotStdout = append(gotStdout, data...)
		}),
		WithOnStderr(func(data []byte) {
			mu.Lock()
			defer mu.Unlock()
			gotStderr = append(gotStderr, data...)
		}),
	)
	if err != nil {
		t.Fatalf("Run 失败: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", result.ExitCode)
	}

	mu.Lock()
	stdoutStr := string(gotStdout)
	stderrStr := string(gotStderr)
	mu.Unlock()

	if !strings.Contains(stdoutStr, "out") {
		t.Fatalf("Stdout 回调未收到预期数据: %q", stdoutStr)
	}
	if !strings.Contains(stderrStr, "err") {
		t.Fatalf("Stderr 回调未收到预期数据: %q", stderrStr)
	}
	t.Logf("回调验证通过: stdout=%q, stderr=%q", stdoutStr, stderrStr)
}

func TestIntegrationCommandsWithOptions(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("沙箱: %s", sb.SandboxID)

	// 测试 WithCwd
	result, err := sb.Commands().Run(ctx, "pwd", WithCwd("/tmp"))
	if err != nil {
		t.Fatalf("Run with WithCwd 失败: %v", err)
	}
	if !strings.Contains(result.Stdout, "/tmp") {
		t.Fatalf("WithCwd: Stdout = %q, 应包含 '/tmp'", result.Stdout)
	}
	t.Logf("WithCwd 验证通过: Stdout=%q", result.Stdout)

	// 测试 WithEnvs
	result, err = sb.Commands().Run(ctx, "echo $FOO", WithEnvs(map[string]string{"FOO": "BAR"}))
	if err != nil {
		t.Fatalf("Run with WithEnvs 失败: %v", err)
	}
	if !strings.Contains(result.Stdout, "BAR") {
		t.Fatalf("WithEnvs: Stdout = %q, 应包含 'BAR'", result.Stdout)
	}
	t.Logf("WithEnvs 验证通过: Stdout=%q", result.Stdout)
}

// --- PTY 交互 ---

func TestIntegrationPtyCreateAndKill(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("沙箱: %s", sb.SandboxID)

	var (
		mu     sync.Mutex
		output []byte
	)

	handle, err := sb.Pty().Create(ctx, PtySize{Cols: 80, Rows: 24},
		WithOnStdout(func(data []byte) {
			mu.Lock()
			defer mu.Unlock()
			output = append(output, data...)
		}),
	)
	if err != nil {
		t.Fatalf("Pty.Create 失败: %v", err)
	}

	// 等待 PID 被设置
	if _, err := handle.WaitPID(ctx); err != nil {
		t.Fatalf("WaitPID 失败: %v", err)
	}
	t.Logf("PTY 已创建: PID=%d", handle.PID)

	// 等待一些 PTY 输出
	time.Sleep(2 * time.Second)

	// Kill PTY
	if err := sb.Pty().Kill(ctx, handle.PID); err != nil {
		t.Fatalf("Pty.Kill 失败: %v", err)
	}

	result, err := handle.Wait()
	if err != nil {
		t.Fatalf("Wait 失败: %v", err)
	}

	mu.Lock()
	outputStr := string(output)
	mu.Unlock()

	t.Logf("PTY 输出长度: %d bytes, ExitCode=%d", len(outputStr), result.ExitCode)
	if len(outputStr) == 0 {
		t.Log("警告: PTY 未收到输出（某些环境下 bash prompt 可能为空）")
	}
	t.Log("Pty.Create/Kill 验证通过")
}

func TestIntegrationPtySendInput(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("沙箱: %s", sb.SandboxID)

	var (
		mu     sync.Mutex
		output []byte
	)

	handle, err := sb.Pty().Create(ctx, PtySize{Cols: 80, Rows: 24},
		WithOnStdout(func(data []byte) {
			mu.Lock()
			defer mu.Unlock()
			output = append(output, data...)
		}),
	)
	if err != nil {
		t.Fatalf("Pty.Create 失败: %v", err)
	}

	// 等待 PID 被设置并让 shell 初始化
	if _, err := handle.WaitPID(ctx); err != nil {
		t.Fatalf("WaitPID 失败: %v", err)
	}
	time.Sleep(2 * time.Second)

	// 发送输入
	if err := sb.Pty().SendInput(ctx, handle.PID, []byte("echo pty-test\n")); err != nil {
		t.Fatalf("Pty.SendInput 失败: %v", err)
	}

	// 等待输出
	deadline := time.After(10 * time.Second)
	for {
		mu.Lock()
		has := strings.Contains(string(output), "pty-test")
		mu.Unlock()
		if has {
			break
		}
		select {
		case <-deadline:
			mu.Lock()
			t.Fatalf("等待 PTY 输出超时，已收到: %q", string(output))
			mu.Unlock()
		default:
			time.Sleep(200 * time.Millisecond)
		}
	}

	// 清理
	_ = sb.Pty().Kill(ctx, handle.PID)
	_, _ = handle.Wait()

	mu.Lock()
	t.Logf("PTY 输出: %q", string(output))
	mu.Unlock()
	t.Log("Pty.SendInput 验证通过")
}

func TestIntegrationPtyResize(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("沙箱: %s", sb.SandboxID)

	handle, err := sb.Pty().Create(ctx, PtySize{Cols: 80, Rows: 24})
	if err != nil {
		t.Fatalf("Pty.Create 失败: %v", err)
	}

	// 等待 PID 被设置
	if _, err := handle.WaitPID(ctx); err != nil {
		t.Fatalf("WaitPID 失败: %v", err)
	}

	// Resize
	if err := sb.Pty().Resize(ctx, handle.PID, PtySize{Cols: 200, Rows: 50}); err != nil {
		t.Fatalf("Pty.Resize 失败: %v", err)
	}
	t.Log("Resize 调用成功")

	// 清理
	_ = sb.Pty().Kill(ctx, handle.PID)
	_, _ = handle.Wait()
	t.Log("Pty.Resize 验证通过")
}

// --- Filesystem.WatchDir ---

func TestIntegrationWatchDir(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	sb := createTestSandbox(t, c, ctx)
	t.Logf("沙箱: %s", sb.SandboxID)

	// 创建监听目标目录
	watchPath := "/tmp/watch-test-" + fmt.Sprintf("%d", time.Now().UnixNano())
	_, err := sb.Files().MakeDir(ctx, watchPath)
	if err != nil {
		t.Fatalf("MakeDir 失败: %v", err)
	}

	// 启动 WatchDir
	watcher, err := sb.Files().WatchDir(ctx, watchPath, WithRecursive(true))
	if err != nil {
		t.Fatalf("WatchDir 失败: %v", err)
	}
	defer watcher.Stop()

	// 等待 watcher 就绪
	time.Sleep(1 * time.Second)

	// 在另一个 goroutine 写入文件触发事件
	go func() {
		_, _ = sb.Files().Write(ctx, watchPath+"/event-file.txt", []byte("watch me"))
	}()

	// 收集事件
	var events []FilesystemEvent
	deadline := time.After(15 * time.Second)
	for {
		select {
		case ev, ok := <-watcher.Events():
			if !ok {
				goto done
			}
			events = append(events, ev)
			t.Logf("收到事件: Name=%s, Type=%s", ev.Name, ev.Type)
			// 收到至少一个事件就可以了
			if len(events) >= 1 {
				goto done
			}
		case <-deadline:
			goto done
		}
	}
done:

	if len(events) == 0 {
		t.Fatal("未收到任何文件系统事件")
	}
	t.Logf("WatchDir 验证通过: 收到 %d 个事件", len(events))
}

func TestIntegrationMetadata(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	templates, err := c.ListTemplates(ctx, nil)
	if err != nil {
		t.Fatalf("ListTemplates 失败: %v", err)
	}

	var templateID string
	for _, tmpl := range templates {
		if tmpl.BuildStatus == apis.TemplateBuildStatusReady || tmpl.BuildStatus == TemplateBuildStatusUploaded {
			templateID = tmpl.TemplateID
			break
		}
	}
	if templateID == "" {
		t.Skip("没有可用模板，跳过测试")
	}

	timeout := int32(60)
	meta := Metadata{"env": "test", "team": "backend"}
	sb, _, err := c.CreateAndWait(ctx, apis.CreateSandboxJSONRequestBody{
		TemplateID: templateID,
		Timeout:    &timeout,
		Metadata:   &meta,
	}, 2*time.Second)
	if err != nil {
		t.Fatalf("CreateAndWait 失败: %v", err)
	}
	t.Cleanup(func() {
		killCtx, killCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer killCancel()
		if err := sb.Kill(killCtx); err != nil {
			t.Logf("清理沙箱 %s 失败: %v", sb.SandboxID, err)
		}
	})

	info, err := sb.GetInfo(ctx)
	if err != nil {
		t.Fatalf("GetInfo 失败: %v", err)
	}

	if info.Metadata == nil {
		t.Fatal("Metadata 应不为 nil")
	}
	got := *info.Metadata
	if got["env"] != "test" {
		t.Errorf("Metadata[env] = %q, want %q", got["env"], "test")
	}
	if got["team"] != "backend" {
		t.Errorf("Metadata[team] = %q, want %q", got["team"], "backend")
	}
	t.Logf("Metadata 验证通过: %v", got)
}
