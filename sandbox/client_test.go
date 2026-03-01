package sandbox

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/qiniu/go-sdk/v7/sandbox/apis"
)

// mockAPI 实现 apis.ClientWithResponsesInterface 用于测试。
// 每个方法字段可按测试设置；未设置的方法会 panic。
type mockAPI struct {
	healthCheckFn            func(ctx context.Context, editors ...apis.RequestEditorFn) (*apis.HealthCheckResponse, error)
	createSandboxFn          func(ctx context.Context, body apis.CreateSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateSandboxResponse, error)
	getSandboxFn             func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error)
	deleteSandboxFn          func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.DeleteSandboxResponse, error)
	listSandboxesFn          func(ctx context.Context, params *apis.ListSandboxesParams, editors ...apis.RequestEditorFn) (*apis.ListSandboxesResponse, error)
	connectSandboxFn         func(ctx context.Context, sandboxID apis.SandboxID, body apis.ConnectSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.ConnectSandboxResponse, error)
	updateSandboxTimeoutFn   func(ctx context.Context, sandboxID apis.SandboxID, body apis.UpdateSandboxTimeoutJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.UpdateSandboxTimeoutResponse, error)
	pauseSandboxFn           func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.PauseSandboxResponse, error)
	getSandboxMetricsFn      func(ctx context.Context, sandboxID apis.SandboxID, params *apis.GetSandboxMetricsParams, editors ...apis.RequestEditorFn) (*apis.GetSandboxMetricsResponse, error)
	getSandboxLogsFn         func(ctx context.Context, sandboxID apis.SandboxID, params *apis.GetSandboxLogsParams, editors ...apis.RequestEditorFn) (*apis.GetSandboxLogsResponse, error)
	refreshSandboxFn         func(ctx context.Context, sandboxID apis.SandboxID, body apis.RefreshSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.RefreshSandboxResponse, error)
	listTemplatesFn          func(ctx context.Context, params *apis.ListTemplatesParams, editors ...apis.RequestEditorFn) (*apis.ListTemplatesResponse, error)
	createTemplateV3Fn       func(ctx context.Context, body apis.CreateTemplateV3JSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateTemplateV3Response, error)
	getTemplateFn            func(ctx context.Context, templateID apis.TemplateID, params *apis.GetTemplateParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateResponse, error)
	deleteTemplateFn         func(ctx context.Context, templateID apis.TemplateID, editors ...apis.RequestEditorFn) (*apis.DeleteTemplateResponse, error)
	getTemplateBuildStatusFn func(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, params *apis.GetTemplateBuildStatusParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateBuildStatusResponse, error)
	getTemplateByAliasFn     func(ctx context.Context, alias string, editors ...apis.RequestEditorFn) (*apis.GetTemplateByAliasResponse, error)
}

func httpResponse(statusCode int) *http.Response {
	return &http.Response{StatusCode: statusCode}
}

// --- 沙箱操作 ---

func (m *mockAPI) HealthCheckWithResponse(ctx context.Context, editors ...apis.RequestEditorFn) (*apis.HealthCheckResponse, error) {
	return m.healthCheckFn(ctx, editors...)
}

func (m *mockAPI) CreateSandboxWithResponse(ctx context.Context, body apis.CreateSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateSandboxResponse, error) {
	return m.createSandboxFn(ctx, body, editors...)
}

func (m *mockAPI) CreateSandboxWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.CreateSandboxResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) GetSandboxWithResponse(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error) {
	return m.getSandboxFn(ctx, sandboxID, editors...)
}

func (m *mockAPI) DeleteSandboxWithResponse(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.DeleteSandboxResponse, error) {
	return m.deleteSandboxFn(ctx, sandboxID, editors...)
}

func (m *mockAPI) ListSandboxesWithResponse(ctx context.Context, params *apis.ListSandboxesParams, editors ...apis.RequestEditorFn) (*apis.ListSandboxesResponse, error) {
	return m.listSandboxesFn(ctx, params, editors...)
}

func (m *mockAPI) ListSandboxesV2WithResponse(ctx context.Context, params *apis.ListSandboxesV2Params, editors ...apis.RequestEditorFn) (*apis.ListSandboxesV2Response, error) {
	panic("not implemented")
}

func (m *mockAPI) ConnectSandboxWithResponse(ctx context.Context, sandboxID apis.SandboxID, body apis.ConnectSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.ConnectSandboxResponse, error) {
	return m.connectSandboxFn(ctx, sandboxID, body, editors...)
}

func (m *mockAPI) ConnectSandboxWithBodyWithResponse(ctx context.Context, sandboxID apis.SandboxID, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.ConnectSandboxResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) UpdateSandboxTimeoutWithResponse(ctx context.Context, sandboxID apis.SandboxID, body apis.UpdateSandboxTimeoutJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.UpdateSandboxTimeoutResponse, error) {
	return m.updateSandboxTimeoutFn(ctx, sandboxID, body, editors...)
}

func (m *mockAPI) UpdateSandboxTimeoutWithBodyWithResponse(ctx context.Context, sandboxID apis.SandboxID, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.UpdateSandboxTimeoutResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) PauseSandboxWithResponse(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.PauseSandboxResponse, error) {
	return m.pauseSandboxFn(ctx, sandboxID, editors...)
}

func (m *mockAPI) ResumeSandboxWithResponse(ctx context.Context, sandboxID apis.SandboxID, body apis.ResumeSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.ResumeSandboxResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) ResumeSandboxWithBodyWithResponse(ctx context.Context, sandboxID apis.SandboxID, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.ResumeSandboxResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) GetSandboxMetricsWithResponse(ctx context.Context, sandboxID apis.SandboxID, params *apis.GetSandboxMetricsParams, editors ...apis.RequestEditorFn) (*apis.GetSandboxMetricsResponse, error) {
	return m.getSandboxMetricsFn(ctx, sandboxID, params, editors...)
}

func (m *mockAPI) GetSandboxLogsWithResponse(ctx context.Context, sandboxID apis.SandboxID, params *apis.GetSandboxLogsParams, editors ...apis.RequestEditorFn) (*apis.GetSandboxLogsResponse, error) {
	return m.getSandboxLogsFn(ctx, sandboxID, params, editors...)
}

func (m *mockAPI) RefreshSandboxWithResponse(ctx context.Context, sandboxID apis.SandboxID, body apis.RefreshSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.RefreshSandboxResponse, error) {
	return m.refreshSandboxFn(ctx, sandboxID, body, editors...)
}

func (m *mockAPI) RefreshSandboxWithBodyWithResponse(ctx context.Context, sandboxID apis.SandboxID, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.RefreshSandboxResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) GetSandboxesMetricsWithResponse(ctx context.Context, params *apis.GetSandboxesMetricsParams, editors ...apis.RequestEditorFn) (*apis.GetSandboxesMetricsResponse, error) {
	panic("not implemented")
}

// --- 模板操作 ---

func (m *mockAPI) ListTemplatesWithResponse(ctx context.Context, params *apis.ListTemplatesParams, editors ...apis.RequestEditorFn) (*apis.ListTemplatesResponse, error) {
	return m.listTemplatesFn(ctx, params, editors...)
}

func (m *mockAPI) CreateTemplateV3WithResponse(ctx context.Context, body apis.CreateTemplateV3JSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateTemplateV3Response, error) {
	return m.createTemplateV3Fn(ctx, body, editors...)
}

func (m *mockAPI) CreateTemplateV3WithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.CreateTemplateV3Response, error) {
	panic("not implemented")
}

func (m *mockAPI) GetTemplateWithResponse(ctx context.Context, templateID apis.TemplateID, params *apis.GetTemplateParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateResponse, error) {
	return m.getTemplateFn(ctx, templateID, params, editors...)
}

func (m *mockAPI) DeleteTemplateWithResponse(ctx context.Context, templateID apis.TemplateID, editors ...apis.RequestEditorFn) (*apis.DeleteTemplateResponse, error) {
	return m.deleteTemplateFn(ctx, templateID, editors...)
}

func (m *mockAPI) UpdateTemplateWithResponse(ctx context.Context, templateID apis.TemplateID, body apis.UpdateTemplateJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.UpdateTemplateResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) UpdateTemplateWithBodyWithResponse(ctx context.Context, templateID apis.TemplateID, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.UpdateTemplateResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) RebuildTemplateWithResponse(ctx context.Context, templateID apis.TemplateID, body apis.RebuildTemplateJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.RebuildTemplateResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) RebuildTemplateWithBodyWithResponse(ctx context.Context, templateID apis.TemplateID, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.RebuildTemplateResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) CreateTemplateWithResponse(ctx context.Context, body apis.CreateTemplateJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateTemplateResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) CreateTemplateWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.CreateTemplateResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) CreateTemplateV2WithResponse(ctx context.Context, body apis.CreateTemplateV2JSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateTemplateV2Response, error) {
	panic("not implemented")
}

func (m *mockAPI) CreateTemplateV2WithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.CreateTemplateV2Response, error) {
	panic("not implemented")
}

func (m *mockAPI) GetTemplateBuildStatusWithResponse(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, params *apis.GetTemplateBuildStatusParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateBuildStatusResponse, error) {
	return m.getTemplateBuildStatusFn(ctx, templateID, buildID, params, editors...)
}

func (m *mockAPI) GetTemplateBuildLogsWithResponse(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, params *apis.GetTemplateBuildLogsParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateBuildLogsResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) StartTemplateBuildWithResponse(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, editors ...apis.RequestEditorFn) (*apis.StartTemplateBuildResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) StartTemplateBuildV2WithResponse(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, body apis.StartTemplateBuildV2JSONRequestBody, editors ...apis.RequestEditorFn) (*apis.StartTemplateBuildV2Response, error) {
	panic("not implemented")
}

func (m *mockAPI) StartTemplateBuildV2WithBodyWithResponse(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.StartTemplateBuildV2Response, error) {
	panic("not implemented")
}

func (m *mockAPI) GetTemplateFilesWithResponse(ctx context.Context, templateID apis.TemplateID, hash string, editors ...apis.RequestEditorFn) (*apis.GetTemplateFilesResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) GetTemplateByAliasWithResponse(ctx context.Context, alias string, editors ...apis.RequestEditorFn) (*apis.GetTemplateByAliasResponse, error) {
	return m.getTemplateByAliasFn(ctx, alias, editors...)
}

func (m *mockAPI) ManageTemplateTagsWithResponse(ctx context.Context, body apis.ManageTemplateTagsJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.ManageTemplateTagsResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) ManageTemplateTagsWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.ManageTemplateTagsResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) DeleteTemplateTagsWithResponse(ctx context.Context, body apis.DeleteTemplateTagsJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.DeleteTemplateTagsResponse, error) {
	panic("not implemented")
}

func (m *mockAPI) DeleteTemplateTagsWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, editors ...apis.RequestEditorFn) (*apis.DeleteTemplateTagsResponse, error) {
	panic("not implemented")
}

// ============================================================
// 测试用例
// ============================================================

func newTestClient(api apis.ClientWithResponsesInterface) *Client {
	return &Client{config: &Config{APIKey: "test-key"}, api: api}
}

func newTestSandbox(c *Client, id string) *Sandbox {
	return &Sandbox{sandboxID: id, client: c}
}

// --- 客户端测试 ---

func TestNewClient(t *testing.T) {
	c, err := NewClient(&Config{APIKey: "test-key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.API() == nil {
		t.Error("expected non-nil API client")
	}
}

func TestNewClientDefaultEndpoint(t *testing.T) {
	c, err := NewClient(&Config{APIKey: "test-key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.config.Endpoint != "" {
		t.Error("expected empty endpoint in config (defaults applied internally)")
	}
}

func TestNewClientCustomEndpoint(t *testing.T) {
	c, err := NewClient(&Config{APIKey: "test-key", Endpoint: "https://custom.example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.config.Endpoint != "https://custom.example.com" {
		t.Errorf("expected custom endpoint, got %q", c.config.Endpoint)
	}
}

func TestAPIKeyEditor(t *testing.T) {
	editor := apiKeyEditor("test-key")
	req, _ := http.NewRequest("GET", "https://example.com", nil)
	if err := editor(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.Header.Get("X-API-Key"); got != "test-key" {
		t.Errorf("expected X-API-Key 'test-key', got %q", got)
	}
}

// --- Sandbox.Create ---

func TestCreate(t *testing.T) {
	mock := &mockAPI{
		createSandboxFn: func(ctx context.Context, body apis.CreateSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateSandboxResponse, error) {
			return &apis.CreateSandboxResponse{
				JSON201:      &apis.Sandbox{SandboxID: "sb-123", TemplateID: "tmpl-1"},
				HTTPResponse: httpResponse(201),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb, err := c.Create(context.Background(), CreateParams{TemplateID: "tmpl-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sb.ID() != "sb-123" {
		t.Errorf("expected sandbox ID 'sb-123', got %q", sb.ID())
	}
	if sb.TemplateID() != "tmpl-1" {
		t.Errorf("expected template ID 'tmpl-1', got %q", sb.TemplateID())
	}
}

func TestCreateError(t *testing.T) {
	mock := &mockAPI{
		createSandboxFn: func(ctx context.Context, body apis.CreateSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateSandboxResponse, error) {
			return &apis.CreateSandboxResponse{
				HTTPResponse: httpResponse(400),
				Body:         []byte(`{"message":"bad request"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	_, err := c.Create(context.Background(), CreateParams{TemplateID: "tmpl-1"})
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}
}

// --- Sandbox.Connect ---

func TestConnect(t *testing.T) {
	mock := &mockAPI{
		connectSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, body apis.ConnectSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.ConnectSandboxResponse, error) {
			return &apis.ConnectSandboxResponse{
				JSON200:      &apis.Sandbox{SandboxID: sandboxID, TemplateID: "tmpl-1"},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb, err := c.Connect(context.Background(), "sb-123", ConnectParams{Timeout: 60})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sb.ID() != "sb-123" {
		t.Errorf("expected sandbox ID 'sb-123', got %q", sb.ID())
	}
}

// --- Sandbox.List ---

func TestList(t *testing.T) {
	mock := &mockAPI{
		listSandboxesFn: func(ctx context.Context, params *apis.ListSandboxesParams, editors ...apis.RequestEditorFn) (*apis.ListSandboxesResponse, error) {
			list := []apis.ListedSandbox{
				{SandboxID: "sb-1"},
				{SandboxID: "sb-2"},
			}
			return &apis.ListSandboxesResponse{
				JSON200:      &list,
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	sandboxes, err := c.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sandboxes) != 2 {
		t.Errorf("expected 2 sandboxes, got %d", len(sandboxes))
	}
}

// --- Sandbox.Kill ---

func TestKill(t *testing.T) {
	mock := &mockAPI{
		deleteSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.DeleteSandboxResponse, error) {
			return &apis.DeleteSandboxResponse{HTTPResponse: httpResponse(204)}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	if err := sb.Kill(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Sandbox.SetTimeout ---

func TestSetTimeout(t *testing.T) {
	var gotTimeout int32
	mock := &mockAPI{
		updateSandboxTimeoutFn: func(ctx context.Context, sandboxID apis.SandboxID, body apis.UpdateSandboxTimeoutJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.UpdateSandboxTimeoutResponse, error) {
			gotTimeout = body.Timeout
			return &apis.UpdateSandboxTimeoutResponse{HTTPResponse: httpResponse(204)}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	if err := sb.SetTimeout(context.Background(), 2*time.Minute); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotTimeout != 120 {
		t.Errorf("expected timeout 120 seconds, got %d", gotTimeout)
	}
}

// --- Sandbox.GetInfo ---

func TestGetInfo(t *testing.T) {
	mock := &mockAPI{
		getSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error) {
			return &apis.GetSandboxResponse{
				JSON200:      &apis.SandboxDetail{SandboxID: sandboxID, State: apis.Running},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	info, err := sb.GetInfo(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.State != StateRunning {
		t.Errorf("expected state 'running', got %q", info.State)
	}
}

// --- Sandbox.IsRunning ---

func TestIsRunning(t *testing.T) {
	mock := &mockAPI{
		getSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error) {
			return &apis.GetSandboxResponse{
				JSON200:      &apis.SandboxDetail{SandboxID: sandboxID, State: apis.Running},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	running, err := sb.IsRunning(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !running {
		t.Error("expected sandbox to be running")
	}
}

func TestIsRunningPaused(t *testing.T) {
	mock := &mockAPI{
		getSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error) {
			return &apis.GetSandboxResponse{
				JSON200:      &apis.SandboxDetail{SandboxID: sandboxID, State: apis.Paused},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	running, err := sb.IsRunning(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if running {
		t.Error("expected sandbox to not be running")
	}
}

// --- Sandbox.Pause ---

func TestPause(t *testing.T) {
	mock := &mockAPI{
		pauseSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.PauseSandboxResponse, error) {
			return &apis.PauseSandboxResponse{HTTPResponse: httpResponse(204)}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	if err := sb.Pause(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Sandbox.WaitForReady ---

func TestWaitForReadyImmediate(t *testing.T) {
	mock := &mockAPI{
		getSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error) {
			return &apis.GetSandboxResponse{
				JSON200:      &apis.SandboxDetail{SandboxID: sandboxID, State: apis.Running},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	info, err := sb.WaitForReady(context.Background(), WithPollInterval(100*time.Millisecond))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.SandboxID != "sb-123" {
		t.Errorf("expected sandbox ID 'sb-123', got %q", info.SandboxID)
	}
}

func TestWaitForReadyTimeout(t *testing.T) {
	mock := &mockAPI{
		getSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error) {
			return &apis.GetSandboxResponse{
				JSON200:      &apis.SandboxDetail{SandboxID: sandboxID, State: apis.Paused},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := sb.WaitForReady(ctx, WithPollInterval(50*time.Millisecond))
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

// --- Client.HealthCheck ---

func TestHealthCheck(t *testing.T) {
	mock := &mockAPI{
		healthCheckFn: func(ctx context.Context, editors ...apis.RequestEditorFn) (*apis.HealthCheckResponse, error) {
			return &apis.HealthCheckResponse{HTTPResponse: httpResponse(200)}, nil
		},
	}
	c := newTestClient(mock)
	if err := c.HealthCheck(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- 模板测试 ---

func TestListTemplates(t *testing.T) {
	mock := &mockAPI{
		listTemplatesFn: func(ctx context.Context, params *apis.ListTemplatesParams, editors ...apis.RequestEditorFn) (*apis.ListTemplatesResponse, error) {
			list := []apis.Template{
				{TemplateID: "tmpl-1"},
				{TemplateID: "tmpl-2"},
			}
			return &apis.ListTemplatesResponse{
				JSON200:      &list,
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	templates, err := c.ListTemplates(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(templates) != 2 {
		t.Errorf("expected 2 templates, got %d", len(templates))
	}
}

func TestDeleteTemplate(t *testing.T) {
	mock := &mockAPI{
		deleteTemplateFn: func(ctx context.Context, templateID apis.TemplateID, editors ...apis.RequestEditorFn) (*apis.DeleteTemplateResponse, error) {
			return &apis.DeleteTemplateResponse{HTTPResponse: httpResponse(204)}, nil
		},
	}
	c := newTestClient(mock)
	if err := c.DeleteTemplate(context.Background(), "tmpl-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetTemplateByAlias(t *testing.T) {
	mock := &mockAPI{
		getTemplateByAliasFn: func(ctx context.Context, alias string, editors ...apis.RequestEditorFn) (*apis.GetTemplateByAliasResponse, error) {
			return &apis.GetTemplateByAliasResponse{
				JSON200:      &apis.TemplateAliasResponse{TemplateID: "tmpl-1", Public: true},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	tmpl, err := c.GetTemplateByAlias(context.Background(), "my-alias")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tmpl.TemplateID != "tmpl-1" {
		t.Errorf("expected template ID 'tmpl-1', got %q", tmpl.TemplateID)
	}
}

// --- WaitForBuild ---

func TestWaitForBuildReady(t *testing.T) {
	mock := &mockAPI{
		getTemplateBuildStatusFn: func(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, params *apis.GetTemplateBuildStatusParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateBuildStatusResponse, error) {
			return &apis.GetTemplateBuildStatusResponse{
				JSON200:      &apis.TemplateBuildInfo{TemplateID: templateID, BuildID: buildID, Status: apis.TemplateBuildStatusReady},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	info, err := c.WaitForBuild(context.Background(), "tmpl-1", "build-1", WithPollInterval(100*time.Millisecond))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Status != BuildStatusReady {
		t.Errorf("expected status 'ready', got %q", info.Status)
	}
}

func TestWaitForBuildError(t *testing.T) {
	mock := &mockAPI{
		getTemplateBuildStatusFn: func(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, params *apis.GetTemplateBuildStatusParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateBuildStatusResponse, error) {
			return &apis.GetTemplateBuildStatusResponse{
				JSON200:      &apis.TemplateBuildInfo{TemplateID: templateID, BuildID: buildID, Status: apis.TemplateBuildStatusError},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	info, err := c.WaitForBuild(context.Background(), "tmpl-1", "build-1", WithPollInterval(100*time.Millisecond))
	if err == nil {
		t.Fatal("expected error for failed build")
	}
	if info == nil {
		t.Fatal("expected non-nil build info even on error")
	}
	if info.Status != BuildStatusError {
		t.Errorf("expected status 'error', got %q", info.Status)
	}
}

// --- APIError ---

func TestAPIErrorMessage(t *testing.T) {
	// 直接构造（不使用工厂），Message 为空，回退到 body 格式
	err := &APIError{StatusCode: 404, Body: []byte(`{"message":"not found"}`)}
	msg := err.Error()
	if msg != `api error: status 404, body: {"message":"not found"}` {
		t.Errorf("unexpected error message: %s", msg)
	}

	// 使用 newAPIError 工厂，自动解析 JSON body 中的 message
	err2 := newAPIError(404, []byte(`{"code":"not_found","message":"resource not found"}`))
	if err2.Code != "not_found" {
		t.Errorf("expected code 'not_found', got %q", err2.Code)
	}
	if err2.Message != "resource not found" {
		t.Errorf("expected message 'resource not found', got %q", err2.Message)
	}
	msg2 := err2.Error()
	if msg2 != "api error: status 404: resource not found" {
		t.Errorf("unexpected error message: %s", msg2)
	}

	// 非 JSON body，回退到 body 格式
	err3 := newAPIError(500, []byte("internal error"))
	if err3.Code != "" || err3.Message != "" {
		t.Errorf("expected empty code/message for non-JSON body, got code=%q message=%q", err3.Code, err3.Message)
	}
	msg3 := err3.Error()
	if msg3 != "api error: status 500, body: internal error" {
		t.Errorf("unexpected error message: %s", msg3)
	}

	// 空 body
	err4 := newAPIError(502, nil)
	if err4.Code != "" || err4.Message != "" {
		t.Errorf("expected empty code/message for nil body, got code=%q message=%q", err4.Code, err4.Message)
	}
}

// --- Sandbox.ListV2 ---

func TestListV2(t *testing.T) {
	mock := &mockAPI{
		listSandboxesFn: func(ctx context.Context, params *apis.ListSandboxesParams, editors ...apis.RequestEditorFn) (*apis.ListSandboxesResponse, error) {
			panic("should not be called")
		},
	}
	// Override the V2 method via a wrapper
	called := false
	v2Mock := &mockAPIWithListV2{
		mockAPI: mock,
		listSandboxesV2Fn: func(ctx context.Context, params *apis.ListSandboxesV2Params, editors ...apis.RequestEditorFn) (*apis.ListSandboxesV2Response, error) {
			called = true
			list := []apis.ListedSandbox{{SandboxID: "sb-v2"}}
			return &apis.ListSandboxesV2Response{
				JSON200:      &list,
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(v2Mock)
	sandboxes, err := c.ListV2(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected V2 endpoint to be called")
	}
	if len(sandboxes) != 1 || sandboxes[0].SandboxID != "sb-v2" {
		t.Errorf("unexpected sandboxes: %v", sandboxes)
	}
}

// mockAPIWithListV2 包装 mockAPI 并覆盖 ListSandboxesV2WithResponse。
type mockAPIWithListV2 struct {
	*mockAPI
	listSandboxesV2Fn func(ctx context.Context, params *apis.ListSandboxesV2Params, editors ...apis.RequestEditorFn) (*apis.ListSandboxesV2Response, error)
}

func (m *mockAPIWithListV2) ListSandboxesV2WithResponse(ctx context.Context, params *apis.ListSandboxesV2Params, editors ...apis.RequestEditorFn) (*apis.ListSandboxesV2Response, error) {
	return m.listSandboxesV2Fn(ctx, params, editors...)
}

// --- Sandbox.GetMetrics ---

func TestGetMetrics(t *testing.T) {
	mock := &mockAPI{
		getSandboxMetricsFn: func(ctx context.Context, sandboxID apis.SandboxID, params *apis.GetSandboxMetricsParams, editors ...apis.RequestEditorFn) (*apis.GetSandboxMetricsResponse, error) {
			metrics := []apis.SandboxMetric{{CPUUsedPct: 50.0, CPUCount: 2}}
			return &apis.GetSandboxMetricsResponse{
				JSON200:      &metrics,
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	metrics, err := sb.GetMetrics(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}
	if metrics[0].CPUUsedPct != 50.0 {
		t.Errorf("expected CPU 50%%, got %f", metrics[0].CPUUsedPct)
	}
}

// --- Sandbox.GetLogs ---

func TestGetLogs(t *testing.T) {
	mock := &mockAPI{
		getSandboxLogsFn: func(ctx context.Context, sandboxID apis.SandboxID, params *apis.GetSandboxLogsParams, editors ...apis.RequestEditorFn) (*apis.GetSandboxLogsResponse, error) {
			return &apis.GetSandboxLogsResponse{
				JSON200:      &apis.SandboxLogs{Logs: []apis.SandboxLog{{Line: "hello"}}},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	logs, err := sb.GetLogs(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs.Logs) != 1 || logs.Logs[0].Line != "hello" {
		t.Errorf("unexpected logs: %v", logs)
	}
}

// --- Sandbox.Refresh ---

func TestRefresh(t *testing.T) {
	mock := &mockAPI{
		refreshSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, body apis.RefreshSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.RefreshSandboxResponse, error) {
			return &apis.RefreshSandboxResponse{HTTPResponse: httpResponse(204)}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	if err := sb.Refresh(context.Background(), RefreshParams{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Client.CreateAndWait ---

func TestCreateAndWait(t *testing.T) {
	mock := &mockAPI{
		createSandboxFn: func(ctx context.Context, body apis.CreateSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateSandboxResponse, error) {
			return &apis.CreateSandboxResponse{
				JSON201:      &apis.Sandbox{SandboxID: "sb-new", TemplateID: "tmpl-1"},
				HTTPResponse: httpResponse(201),
			}, nil
		},
		getSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error) {
			return &apis.GetSandboxResponse{
				JSON200:      &apis.SandboxDetail{SandboxID: sandboxID, State: apis.Running},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb, info, err := c.CreateAndWait(context.Background(), CreateParams{TemplateID: "tmpl-1"}, WithPollInterval(100*time.Millisecond))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sb.ID() != "sb-new" {
		t.Errorf("expected sandbox ID 'sb-new', got %q", sb.ID())
	}
	if info.State != StateRunning {
		t.Errorf("expected state 'running', got %q", info.State)
	}
}

func TestCreateAndWaitCreateFails(t *testing.T) {
	mock := &mockAPI{
		createSandboxFn: func(ctx context.Context, body apis.CreateSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateSandboxResponse, error) {
			return &apis.CreateSandboxResponse{
				HTTPResponse: httpResponse(500),
				Body:         []byte("internal error"),
			}, nil
		},
	}
	c := newTestClient(mock)
	_, _, err := c.CreateAndWait(context.Background(), CreateParams{TemplateID: "tmpl-1"}, WithPollInterval(100*time.Millisecond))
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Sandbox.WaitForReady 轮询 ---

func TestWaitForReadyPolling(t *testing.T) {
	callCount := 0
	mock := &mockAPI{
		getSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error) {
			callCount++
			state := apis.Paused
			if callCount >= 3 {
				state = apis.Running
			}
			return &apis.GetSandboxResponse{
				JSON200:      &apis.SandboxDetail{SandboxID: sandboxID, State: state},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	info, err := sb.WaitForReady(context.Background(), WithPollInterval(50*time.Millisecond))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount < 3 {
		t.Errorf("expected at least 3 calls, got %d", callCount)
	}
	if info.State != StateRunning {
		t.Errorf("expected state 'running', got %q", info.State)
	}
}

// --- 实例方法的错误用例 ---

func TestKillError(t *testing.T) {
	mock := &mockAPI{
		deleteSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.DeleteSandboxResponse, error) {
			return &apis.DeleteSandboxResponse{
				HTTPResponse: httpResponse(404),
				Body:         []byte(`{"message":"not found"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-999")
	err := sb.Kill(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
}

func TestSetTimeoutError(t *testing.T) {
	mock := &mockAPI{
		updateSandboxTimeoutFn: func(ctx context.Context, sandboxID apis.SandboxID, body apis.UpdateSandboxTimeoutJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.UpdateSandboxTimeoutResponse, error) {
			return &apis.UpdateSandboxTimeoutResponse{
				HTTPResponse: httpResponse(404),
				Body:         []byte(`{"message":"not found"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-999")
	err := sb.SetTimeout(context.Background(), time.Minute)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPauseError(t *testing.T) {
	mock := &mockAPI{
		pauseSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.PauseSandboxResponse, error) {
			return &apis.PauseSandboxResponse{
				HTTPResponse: httpResponse(409),
				Body:         []byte(`{"message":"conflict"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-123")
	err := sb.Pause(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConnectError(t *testing.T) {
	mock := &mockAPI{
		connectSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, body apis.ConnectSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.ConnectSandboxResponse, error) {
			return &apis.ConnectSandboxResponse{
				HTTPResponse: httpResponse(404),
				Body:         []byte(`{"message":"not found"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	_, err := c.Connect(context.Background(), "sb-999", ConnectParams{Timeout: 60})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetInfoError(t *testing.T) {
	mock := &mockAPI{
		getSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error) {
			return &apis.GetSandboxResponse{
				HTTPResponse: httpResponse(404),
				Body:         []byte(`{"message":"not found"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb := newTestSandbox(c, "sb-999")
	_, err := sb.GetInfo(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHealthCheckError(t *testing.T) {
	mock := &mockAPI{
		healthCheckFn: func(ctx context.Context, editors ...apis.RequestEditorFn) (*apis.HealthCheckResponse, error) {
			return &apis.HealthCheckResponse{HTTPResponse: httpResponse(503)}, nil
		},
	}
	c := newTestClient(mock)
	err := c.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Template: CreateTemplate ---

func TestCreateTemplate(t *testing.T) {
	mock := &mockAPI{
		createTemplateV3Fn: func(ctx context.Context, body apis.CreateTemplateV3JSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateTemplateV3Response, error) {
			return &apis.CreateTemplateV3Response{
				JSON202:      &apis.TemplateRequestResponseV3{TemplateID: "tmpl-new", BuildID: "build-1"},
				HTTPResponse: httpResponse(202),
			}, nil
		},
	}
	c := newTestClient(mock)
	resp, err := c.CreateTemplate(context.Background(), apis.TemplateBuildRequestV3{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TemplateID != "tmpl-new" {
		t.Errorf("expected template ID 'tmpl-new', got %q", resp.TemplateID)
	}
	if resp.BuildID != "build-1" {
		t.Errorf("expected build ID 'build-1', got %q", resp.BuildID)
	}
}

func TestCreateTemplateError(t *testing.T) {
	mock := &mockAPI{
		createTemplateV3Fn: func(ctx context.Context, body apis.CreateTemplateV3JSONRequestBody, editors ...apis.RequestEditorFn) (*apis.CreateTemplateV3Response, error) {
			return &apis.CreateTemplateV3Response{
				HTTPResponse: httpResponse(400),
				Body:         []byte(`{"message":"bad request"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	_, err := c.CreateTemplate(context.Background(), apis.TemplateBuildRequestV3{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Template: GetTemplate ---

func TestGetTemplate(t *testing.T) {
	mock := &mockAPI{
		getTemplateFn: func(ctx context.Context, templateID apis.TemplateID, params *apis.GetTemplateParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateResponse, error) {
			return &apis.GetTemplateResponse{
				JSON200: &apis.TemplateWithBuilds{
					TemplateID: templateID,
					Builds:     []apis.TemplateBuild{{}},
				},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	tmpl, err := c.GetTemplate(context.Background(), "tmpl-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tmpl.TemplateID != "tmpl-1" {
		t.Errorf("expected template ID 'tmpl-1', got %q", tmpl.TemplateID)
	}
	if len(tmpl.Builds) != 1 {
		t.Errorf("expected 1 build, got %d", len(tmpl.Builds))
	}
}

// --- Template: GetTemplateBuildStatus ---

func TestGetTemplateBuildStatus(t *testing.T) {
	mock := &mockAPI{
		getTemplateBuildStatusFn: func(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, params *apis.GetTemplateBuildStatusParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateBuildStatusResponse, error) {
			return &apis.GetTemplateBuildStatusResponse{
				JSON200:      &apis.TemplateBuildInfo{TemplateID: templateID, BuildID: buildID, Status: apis.TemplateBuildStatusBuilding},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	info, err := c.GetTemplateBuildStatus(context.Background(), "tmpl-1", "build-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Status != BuildStatusBuilding {
		t.Errorf("expected status 'building', got %q", info.Status)
	}
}

// --- Template: WaitForBuild 轮询 ---

func TestWaitForBuildPolling(t *testing.T) {
	callCount := 0
	mock := &mockAPI{
		getTemplateBuildStatusFn: func(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, params *apis.GetTemplateBuildStatusParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateBuildStatusResponse, error) {
			callCount++
			status := apis.TemplateBuildStatusBuilding
			if callCount >= 3 {
				status = apis.TemplateBuildStatusReady
			}
			return &apis.GetTemplateBuildStatusResponse{
				JSON200:      &apis.TemplateBuildInfo{TemplateID: templateID, BuildID: buildID, Status: status},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	info, err := c.WaitForBuild(context.Background(), "tmpl-1", "build-1", WithPollInterval(50*time.Millisecond))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount < 3 {
		t.Errorf("expected at least 3 calls, got %d", callCount)
	}
	if info.Status != BuildStatusReady {
		t.Errorf("expected status 'ready', got %q", info.Status)
	}
}

func TestWaitForBuildTimeout(t *testing.T) {
	mock := &mockAPI{
		getTemplateBuildStatusFn: func(ctx context.Context, templateID apis.TemplateID, buildID apis.BuildID, params *apis.GetTemplateBuildStatusParams, editors ...apis.RequestEditorFn) (*apis.GetTemplateBuildStatusResponse, error) {
			return &apis.GetTemplateBuildStatusResponse{
				JSON200:      &apis.TemplateBuildInfo{TemplateID: templateID, BuildID: buildID, Status: apis.TemplateBuildStatusBuilding},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mock)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := c.WaitForBuild(ctx, "tmpl-1", "build-1", WithPollInterval(50*time.Millisecond))
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

// --- Template: DeleteTemplate 错误 ---

func TestDeleteTemplateError(t *testing.T) {
	mock := &mockAPI{
		deleteTemplateFn: func(ctx context.Context, templateID apis.TemplateID, editors ...apis.RequestEditorFn) (*apis.DeleteTemplateResponse, error) {
			return &apis.DeleteTemplateResponse{
				HTTPResponse: httpResponse(500),
				Body:         []byte("internal error"),
			}, nil
		},
	}
	c := newTestClient(mock)
	err := c.DeleteTemplate(context.Background(), "tmpl-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Template: GetTemplateByAlias 未找到 ---

func TestGetTemplateByAliasNotFound(t *testing.T) {
	mock := &mockAPI{
		getTemplateByAliasFn: func(ctx context.Context, alias string, editors ...apis.RequestEditorFn) (*apis.GetTemplateByAliasResponse, error) {
			return &apis.GetTemplateByAliasResponse{
				HTTPResponse: httpResponse(404),
				Body:         []byte(`{"message":"not found"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	_, err := c.GetTemplateByAlias(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
}

// --- Template: ListTemplates 错误 ---

func TestListTemplatesError(t *testing.T) {
	mock := &mockAPI{
		listTemplatesFn: func(ctx context.Context, params *apis.ListTemplatesParams, editors ...apis.RequestEditorFn) (*apis.ListTemplatesResponse, error) {
			return &apis.ListTemplatesResponse{
				HTTPResponse: httpResponse(401),
				Body:         []byte(`{"message":"unauthorized"}`),
			}, nil
		},
	}
	c := newTestClient(mock)
	_, err := c.ListTemplates(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Sandbox.GetSandboxesMetrics ---

func TestGetSandboxesMetrics(t *testing.T) {
	mockWithMetrics := &mockAPIWithSandboxesMetrics{
		mockAPI: &mockAPI{},
		getSandboxesMetricsFn: func(ctx context.Context, params *apis.GetSandboxesMetricsParams, editors ...apis.RequestEditorFn) (*apis.GetSandboxesMetricsResponse, error) {
			return &apis.GetSandboxesMetricsResponse{
				JSON200:      &apis.SandboxesWithMetrics{Sandboxes: map[string]apis.SandboxMetric{"sb-1": {CPUCount: 4}}},
				HTTPResponse: httpResponse(200),
			}, nil
		},
	}
	c := newTestClient(mockWithMetrics)
	metrics, err := c.GetSandboxesMetrics(context.Background(), &apis.GetSandboxesMetricsParams{SandboxIds: []string{"sb-1"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if metrics.Sandboxes["sb-1"].CPUCount != 4 {
		t.Errorf("unexpected cpu count: %v", metrics.Sandboxes["sb-1"].CPUCount)
	}
}

// mockAPIWithSandboxesMetrics 包装 mockAPI 并覆盖 GetSandboxesMetricsWithResponse。
type mockAPIWithSandboxesMetrics struct {
	*mockAPI
	getSandboxesMetricsFn func(ctx context.Context, params *apis.GetSandboxesMetricsParams, editors ...apis.RequestEditorFn) (*apis.GetSandboxesMetricsResponse, error)
}

func (m *mockAPIWithSandboxesMetrics) GetSandboxesMetricsWithResponse(ctx context.Context, params *apis.GetSandboxesMetricsParams, editors ...apis.RequestEditorFn) (*apis.GetSandboxesMetricsResponse, error) {
	return m.getSandboxesMetricsFn(ctx, params, editors...)
}

// --- Connect 返回 JSON201（新建沙箱） ---

func TestConnectCreated(t *testing.T) {
	mock := &mockAPI{
		connectSandboxFn: func(ctx context.Context, sandboxID apis.SandboxID, body apis.ConnectSandboxJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.ConnectSandboxResponse, error) {
			return &apis.ConnectSandboxResponse{
				JSON201:      &apis.Sandbox{SandboxID: sandboxID, TemplateID: "tmpl-1"},
				HTTPResponse: httpResponse(201),
			}, nil
		},
	}
	c := newTestClient(mock)
	sb, err := c.Connect(context.Background(), "sb-new", ConnectParams{Timeout: 60})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sb.ID() != "sb-new" {
		t.Errorf("expected sandbox ID 'sb-new', got %q", sb.ID())
	}
}
