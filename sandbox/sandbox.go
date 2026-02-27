package sandbox

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/qiniu/go-sdk/v7/sandbox/apis"
)

// Sandbox 表示一个运行中的沙箱实例。
// 持有客户端引用，用于执行生命周期操作。
type Sandbox struct {
	SandboxID          string
	TemplateID         string
	ClientID           string
	Alias              *string
	Domain             *string
	EnvdAccessToken    *string
	TrafficAccessToken *string

	client *Client
}

// newSandbox 从 API 响应创建 Sandbox 实例。
func newSandbox(c *Client, s *apis.Sandbox) *Sandbox {
	return &Sandbox{
		SandboxID:          s.SandboxID,
		TemplateID:         s.TemplateID,
		ClientID:           s.ClientID,
		Alias:              s.Alias,
		Domain:             s.Domain,
		EnvdAccessToken:    s.EnvdAccessToken,
		TrafficAccessToken: s.TrafficAccessToken,
		client:             c,
	}
}

// Create 根据指定模板创建一个新的沙箱。
func (c *Client) Create(ctx context.Context, body apis.CreateSandboxJSONRequestBody) (*Sandbox, error) {
	resp, err := c.api.CreateSandboxWithResponse(ctx, body)
	if err != nil {
		return nil, err
	}
	if resp.JSON201 == nil {
		return nil, &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return newSandbox(c, resp.JSON201), nil
}

// Connect 连接到一个已有的沙箱，可选择恢复已暂停的沙箱。
func (c *Client) Connect(ctx context.Context, sandboxID string, body apis.ConnectSandboxJSONRequestBody) (*Sandbox, error) {
	resp, err := c.api.ConnectSandboxWithResponse(ctx, sandboxID, body)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return newSandbox(c, resp.JSON200), nil
	}
	if resp.JSON201 != nil {
		return newSandbox(c, resp.JSON201), nil
	}
	return nil, &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
}

// List 列出所有运行中的沙箱。
func (c *Client) List(ctx context.Context, params *apis.ListSandboxesParams) ([]apis.ListedSandbox, error) {
	resp, err := c.api.ListSandboxesWithResponse(ctx, params)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return *resp.JSON200, nil
}

// ListV2 列出沙箱，支持分页和状态过滤。
func (c *Client) ListV2(ctx context.Context, params *apis.ListSandboxesV2Params) ([]apis.ListedSandbox, error) {
	resp, err := c.api.ListSandboxesV2WithResponse(ctx, params)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return *resp.JSON200, nil
}

// Kill 终止沙箱。
func (s *Sandbox) Kill(ctx context.Context) error {
	resp, err := s.client.api.DeleteSandboxWithResponse(ctx, s.SandboxID)
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != http.StatusNoContent {
		return &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return nil
}

// SetTimeout 更新沙箱超时时间。
// 沙箱将在从现在起经过指定时长后过期。
func (s *Sandbox) SetTimeout(ctx context.Context, timeout time.Duration) error {
	timeoutSec := int32(timeout.Seconds())
	resp, err := s.client.api.UpdateSandboxTimeoutWithResponse(ctx, s.SandboxID, apis.UpdateSandboxTimeoutJSONRequestBody{
		Timeout: timeoutSec,
	})
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != http.StatusNoContent {
		return &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return nil
}

// GetInfo 返回沙箱的详细信息。
func (s *Sandbox) GetInfo(ctx context.Context) (*apis.SandboxDetail, error) {
	resp, err := s.client.api.GetSandboxWithResponse(ctx, s.SandboxID)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return resp.JSON200, nil
}

// IsRunning 检查沙箱是否处于运行状态。
func (s *Sandbox) IsRunning(ctx context.Context) (bool, error) {
	info, err := s.GetInfo(ctx)
	if err != nil {
		return false, err
	}
	return info.State == apis.Running, nil
}

// GetMetrics 返回沙箱的资源指标。
func (s *Sandbox) GetMetrics(ctx context.Context, params *apis.GetSandboxMetricsParams) ([]apis.SandboxMetric, error) {
	resp, err := s.client.api.GetSandboxMetricsWithResponse(ctx, s.SandboxID, params)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return *resp.JSON200, nil
}

// GetLogs 返回沙箱日志。
func (s *Sandbox) GetLogs(ctx context.Context, params *apis.GetSandboxLogsParams) (*apis.SandboxLogs, error) {
	resp, err := s.client.api.GetSandboxLogsWithResponse(ctx, s.SandboxID, params)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return resp.JSON200, nil
}

// Pause 暂停沙箱，以便后续恢复。
func (s *Sandbox) Pause(ctx context.Context) error {
	resp, err := s.client.api.PauseSandboxWithResponse(ctx, s.SandboxID)
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != http.StatusNoContent {
		return &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return nil
}

// Refresh 延长沙箱的存活时间。
func (s *Sandbox) Refresh(ctx context.Context, body apis.RefreshSandboxJSONRequestBody) error {
	resp, err := s.client.api.RefreshSandboxWithResponse(ctx, s.SandboxID, body)
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != http.StatusNoContent {
		return &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return nil
}

// WaitForReady 轮询 GetInfo 直到沙箱状态变为 "running" 或上下文被取消。
func (s *Sandbox) WaitForReady(ctx context.Context, pollInterval time.Duration) (*apis.SandboxDetail, error) {
	if pollInterval <= 0 {
		pollInterval = time.Second
	}

	for {
		info, err := s.GetInfo(ctx)
		if err != nil {
			return nil, fmt.Errorf("get sandbox %s: %w", s.SandboxID, err)
		}
		if info.State == apis.Running {
			return info, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

// CreateAndWait 创建沙箱并等待其就绪。
func (c *Client) CreateAndWait(ctx context.Context, body apis.CreateSandboxJSONRequestBody, pollInterval time.Duration) (*Sandbox, *apis.SandboxDetail, error) {
	sb, err := c.Create(ctx, body)
	if err != nil {
		return nil, nil, fmt.Errorf("create sandbox: %w", err)
	}
	info, err := sb.WaitForReady(ctx, pollInterval)
	if err != nil {
		return nil, nil, err
	}
	return sb, info, nil
}

// HealthCheck 对 API 执行健康检查。
func (c *Client) HealthCheck(ctx context.Context) error {
	resp, err := c.api.HealthCheckWithResponse(ctx)
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != http.StatusOK {
		return &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return nil
}

// GetSandboxesMetrics 返回指定沙箱 ID 列表的指标数据。
func (c *Client) GetSandboxesMetrics(ctx context.Context, params *apis.GetSandboxesMetricsParams) (*apis.SandboxesWithMetrics, error) {
	resp, err := c.api.GetSandboxesMetricsWithResponse(ctx, params)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return resp.JSON200, nil
}
