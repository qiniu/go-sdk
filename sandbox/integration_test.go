//go:build integration

package sandbox

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/qiniu/go-sdk/v7/sandbox/apis"
)

// testClient 从环境变量创建集成测试用的客户端。
// 优先使用 Qiniu_API_KEY / Qiniu_SANDBOX_API_URL，
// 其次使用 E2B_API_KEY / E2B_API_URL。
func testClient(t *testing.T) *Client {
	t.Helper()

	apiKey := os.Getenv("Qiniu_API_KEY")
	apiURL := os.Getenv("Qiniu_SANDBOX_API_URL")
	if apiKey == "" {
		apiKey = os.Getenv("E2B_API_KEY")
		apiURL = os.Getenv("E2B_API_URL")
	}
	if apiKey == "" {
		t.Fatal("需要设置 Qiniu_API_KEY 或 E2B_API_KEY 环境变量")
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
		if tmpl.BuildStatus == apis.TemplateBuildStatusReady || tmpl.BuildStatus == "uploaded" {
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
		if tmpl.BuildStatus == apis.TemplateBuildStatusReady || tmpl.BuildStatus == "uploaded" {
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
