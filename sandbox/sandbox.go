package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/qiniu/go-sdk/v7/sandbox/apis"
)

// envdPort 是 envd agent 的默认端口。
const envdPort = 49983

// Sandbox 表示一个运行中的沙箱实例。
// 持有客户端引用，用于执行生命周期操作和 envd agent 通信。
type Sandbox struct {
	SandboxID          string
	TemplateID         string
	ClientID           string
	Alias              *string
	Domain             *string
	EnvdAccessToken    *string
	TrafficAccessToken *string

	client *Client

	// envd 子模块（懒初始化）
	filesOnce sync.Once
	files     *Filesystem

	commandsOnce sync.Once
	commands     *Commands

	ptyOnce sync.Once
	pty     *Pty
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

// Files 返回文件系统操作接口。
func (s *Sandbox) Files() *Filesystem {
	s.filesOnce.Do(func() {
		s.files = newFilesystem(s)
	})
	return s.files
}

// Commands 返回命令执行操作接口。
func (s *Sandbox) Commands() *Commands {
	s.commandsOnce.Do(func() {
		s.commands = newCommands(s)
	})
	return s.commands
}

// Pty 返回 PTY 终端操作接口。
func (s *Sandbox) Pty() *Pty {
	s.ptyOnce.Do(func() {
		s.pty = newPty(s)
	})
	return s.pty
}

// GetHost 返回访问沙箱指定端口的外部域名。
// 格式: {port}-{sandboxID}.{domain}
func (s *Sandbox) GetHost(port int) string {
	domain := s.client.config.Domain
	if domain == "" {
		domain = DefaultDomain
	}
	if s.Domain != nil && *s.Domain != "" {
		domain = *s.Domain
	}
	return fmt.Sprintf("%d-%s.%s", port, s.SandboxID, domain)
}

// envdURL 返回 envd agent 的基础 URL。
func (s *Sandbox) envdURL() string {
	return fmt.Sprintf("https://%s", s.GetHost(envdPort))
}

// envdAuthHeader 返回 envd 认证头。
// 认证格式为 Basic base64(username:)。
func envdAuthHeader(user string) http.Header {
	h := http.Header{}
	cred := base64.StdEncoding.EncodeToString([]byte(user + ":"))
	h.Set("Authorization", "Basic "+cred)
	return h
}

// FileURLOption 文件 URL 选项。
type FileURLOption func(*fileURLOpts)

type fileURLOpts struct {
	user                string
	signatureExpiration int
}

// WithFileUser 设置文件操作的用户。
func WithFileUser(user string) FileURLOption {
	return func(o *fileURLOpts) { o.user = user }
}

// WithSignatureExpiration 设置签名过期时间（秒）。
func WithSignatureExpiration(seconds int) FileURLOption {
	return func(o *fileURLOpts) { o.signatureExpiration = seconds }
}

// fileSignature 计算文件操作签名。
// 算法: "v1_" + SHA256(path + ":" + operation + ":" + username + ":" + accessToken + ":" + expiration)
func fileSignature(path, operation, username, accessToken string, expiration int) string {
	raw := fmt.Sprintf("%s:%s:%s:%s:%d", path, operation, username, accessToken, expiration)
	hash := sha256.Sum256([]byte(raw))
	return "v1_" + fmt.Sprintf("%x", hash)
}

// DownloadURL 返回从沙箱下载文件的 URL。
func (s *Sandbox) DownloadURL(path string, opts ...FileURLOption) string {
	o := &fileURLOpts{user: "user"}
	for _, fn := range opts {
		fn(o)
	}

	q := url.Values{}
	q.Set("path", path)
	q.Set("username", o.user)

	if s.EnvdAccessToken != nil && *s.EnvdAccessToken != "" {
		exp := o.signatureExpiration
		if exp == 0 {
			exp = 300
		}
		sig := fileSignature(path, "read", o.user, *s.EnvdAccessToken, exp)
		q.Set("signature", sig)
		q.Set("signature_expiration", strconv.Itoa(exp))
	}

	return s.envdURL() + "/files?" + q.Encode()
}

// UploadURL 返回向沙箱上传文件的 URL（POST multipart/form-data）。
func (s *Sandbox) UploadURL(path string, opts ...FileURLOption) string {
	o := &fileURLOpts{user: "user"}
	for _, fn := range opts {
		fn(o)
	}

	q := url.Values{}
	q.Set("path", path)
	q.Set("username", o.user)

	if s.EnvdAccessToken != nil && *s.EnvdAccessToken != "" {
		exp := o.signatureExpiration
		if exp == 0 {
			exp = 300
		}
		sig := fileSignature(path, "write", o.user, *s.EnvdAccessToken, exp)
		q.Set("signature", sig)
		q.Set("signature_expiration", strconv.Itoa(exp))
	}

	return s.envdURL() + "/files?" + q.Encode()
}

// Upload 向沙箱上传文件。data 为文件内容，path 为沙箱内目标路径。
// 如果文件已存在则覆盖，自动创建父目录。
func (s *Sandbox) Upload(ctx context.Context, path string, data []byte, opts ...FileURLOption) error {
	uploadURL := s.UploadURL(path, opts...)

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	go func() {
		part, err := mw.CreateFormFile("file", filepath.Base(path))
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		if _, err := part.Write(data); err != nil {
			pw.CloseWithError(err)
			return
		}
		if err := mw.Close(); err != nil {
			pw.CloseWithError(err)
			return
		}
		pw.Close()
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, pr)
	if err != nil {
		return fmt.Errorf("create upload request: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	httpClient := s.client.config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{StatusCode: resp.StatusCode, Body: body}
	}
	return nil
}

// Download 从沙箱下载文件，返回文件内容。
func (s *Sandbox) Download(ctx context.Context, path string, opts ...FileURLOption) ([]byte, error) {
	downloadURL := s.DownloadURL(path, opts...)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}

	httpClient := s.client.config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{StatusCode: resp.StatusCode, Body: body}
	}

	return io.ReadAll(resp.Body)
}
