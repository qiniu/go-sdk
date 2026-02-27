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
