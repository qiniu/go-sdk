package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/qiniu/go-sdk/v7/sandbox/apis"
	"github.com/qiniu/go-sdk/v7/sandbox/envdapi/process/processconnect"

	connect "connectrpc.com/connect"
)

// envdPort 是 envd agent 的默认端口。
const envdPort = 49983

// DefaultUser 是沙箱命令执行和文件操作的默认用户名。
const DefaultUser = "user"

// Sandbox 表示一个运行中的沙箱实例。
// 持有客户端引用，用于执行生命周期操作和 envd agent 通信。
type Sandbox struct {
	sandboxID          string
	templateID         string
	clientID           string
	alias              *string
	domain             *string
	envdAccessToken    *string
	trafficAccessToken *string

	client *Client

	// 共享的 ProcessClient（Commands 和 Pty 共用）
	processRPCOnce sync.Once
	processRPC     processconnect.ProcessClient

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
		sandboxID:          s.SandboxID,
		templateID:         s.TemplateID,
		clientID:           s.ClientID,
		alias:              s.Alias,
		domain:             s.Domain,
		envdAccessToken:    s.EnvdAccessToken,
		trafficAccessToken: s.TrafficAccessToken,
		client:             c,
	}
}

// ID 返回沙箱 ID。
func (s *Sandbox) ID() string { return s.sandboxID }

// TemplateID 返回沙箱所属的模板 ID。
func (s *Sandbox) TemplateID() string { return s.templateID }

// Alias 返回沙箱的别名。
func (s *Sandbox) Alias() *string { return s.alias }

// Domain 返回沙箱的域名。
func (s *Sandbox) Domain() *string { return s.domain }

// processClient 返回共享的 ProcessClient 实例，Commands 和 Pty 共用。
func (s *Sandbox) processClient() processconnect.ProcessClient {
	s.processRPCOnce.Do(func() {
		httpClient := s.client.config.HTTPClient
		if httpClient == nil {
			httpClient = http.DefaultClient
		}
		s.processRPC = processconnect.NewProcessClient(
			httpClient,
			s.envdURL(),
		)
	})
	return s.processRPC
}

// Create 根据指定模板创建一个新的沙箱。
func (c *Client) Create(ctx context.Context, params CreateParams) (*Sandbox, error) {
	resp, err := c.api.CreateSandboxWithResponse(ctx, params.toAPI())
	if err != nil {
		return nil, err
	}
	if resp.JSON201 == nil {
		return nil, newAPIError(resp.StatusCode(), resp.Body)
	}
	return newSandbox(c, resp.JSON201), nil
}

// Connect 连接到一个已有的沙箱，可选择恢复已暂停的沙箱。
func (c *Client) Connect(ctx context.Context, sandboxID string, params ConnectParams) (*Sandbox, error) {
	resp, err := c.api.ConnectSandboxWithResponse(ctx, sandboxID, params.toAPI())
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return newSandbox(c, resp.JSON200), nil
	}
	if resp.JSON201 != nil {
		return newSandbox(c, resp.JSON201), nil
	}
	return nil, newAPIError(resp.StatusCode(), resp.Body)
}

// List 列出所有运行中的沙箱。
func (c *Client) List(ctx context.Context, params *ListParams) ([]ListedSandbox, error) {
	resp, err := c.api.ListSandboxesWithResponse(ctx, params)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.StatusCode(), resp.Body)
	}
	return listedSandboxesFromAPI(*resp.JSON200), nil
}

// ListV2 列出沙箱，支持分页和状态过滤。
func (c *Client) ListV2(ctx context.Context, params *ListV2Params) ([]ListedSandbox, error) {
	resp, err := c.api.ListSandboxesV2WithResponse(ctx, params.toAPI())
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.StatusCode(), resp.Body)
	}
	return listedSandboxesFromAPI(*resp.JSON200), nil
}

// Kill 终止沙箱。
func (s *Sandbox) Kill(ctx context.Context) error {
	resp, err := s.client.api.DeleteSandboxWithResponse(ctx, s.sandboxID)
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != http.StatusNoContent {
		return newAPIError(resp.StatusCode(), resp.Body)
	}
	return nil
}

// SetTimeout 更新沙箱超时时间。
// 沙箱将在从现在起经过指定时长后过期。
// timeout 必须 >= 1 秒。
func (s *Sandbox) SetTimeout(ctx context.Context, timeout time.Duration) error {
	if timeout < time.Second {
		return fmt.Errorf("timeout must be at least 1 second, got %v", timeout)
	}
	secs := timeout.Seconds()
	if secs > float64(math.MaxInt32) {
		return fmt.Errorf("timeout %v exceeds maximum allowed value", timeout)
	}
	timeoutSec := int32(secs)
	resp, err := s.client.api.UpdateSandboxTimeoutWithResponse(ctx, s.sandboxID, apis.UpdateSandboxTimeoutJSONRequestBody{
		Timeout: timeoutSec,
	})
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != http.StatusNoContent {
		return newAPIError(resp.StatusCode(), resp.Body)
	}
	return nil
}

// GetInfo 返回沙箱的详细信息。
func (s *Sandbox) GetInfo(ctx context.Context) (*SandboxInfo, error) {
	resp, err := s.client.api.GetSandboxWithResponse(ctx, s.sandboxID)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.StatusCode(), resp.Body)
	}
	return sandboxInfoFromAPI(resp.JSON200), nil
}

// IsRunning 检查沙箱是否处于运行状态。
func (s *Sandbox) IsRunning(ctx context.Context) (bool, error) {
	info, err := s.GetInfo(ctx)
	if err != nil {
		return false, err
	}
	return info.State == StateRunning, nil
}

// GetMetrics 返回沙箱的资源指标。
func (s *Sandbox) GetMetrics(ctx context.Context, params *GetMetricsParams) ([]SandboxMetric, error) {
	resp, err := s.client.api.GetSandboxMetricsWithResponse(ctx, s.sandboxID, params)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.StatusCode(), resp.Body)
	}
	return sandboxMetricsFromAPI(*resp.JSON200), nil
}

// GetLogs 返回沙箱日志。
func (s *Sandbox) GetLogs(ctx context.Context, params *GetLogsParams) (*SandboxLogs, error) {
	resp, err := s.client.api.GetSandboxLogsWithResponse(ctx, s.sandboxID, params)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.StatusCode(), resp.Body)
	}
	return sandboxLogsFromAPI(resp.JSON200), nil
}

// Pause 暂停沙箱，以便后续恢复。
func (s *Sandbox) Pause(ctx context.Context) error {
	resp, err := s.client.api.PauseSandboxWithResponse(ctx, s.sandboxID)
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != http.StatusNoContent {
		return newAPIError(resp.StatusCode(), resp.Body)
	}
	return nil
}

// Refresh 延长沙箱的存活时间。
func (s *Sandbox) Refresh(ctx context.Context, params RefreshParams) error {
	resp, err := s.client.api.RefreshSandboxWithResponse(ctx, s.sandboxID, params.toAPI())
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != http.StatusNoContent {
		return newAPIError(resp.StatusCode(), resp.Body)
	}
	return nil
}

// WaitForReady 轮询 GetInfo 直到沙箱状态变为 "running" 或上下文被取消。
// 默认轮询间隔为 1 秒，可通过 WithPollInterval 等选项自定义。
func (s *Sandbox) WaitForReady(ctx context.Context, opts ...PollOption) (*SandboxInfo, error) {
	o := defaultPollOpts(time.Second)
	for _, fn := range opts {
		fn(o)
	}

	return pollLoop(ctx, o, func() (bool, *SandboxInfo, error) {
		info, err := s.GetInfo(ctx)
		if err != nil {
			return false, nil, fmt.Errorf("get sandbox %s: %w", s.sandboxID, err)
		}
		if info.State == StateRunning {
			return true, info, nil
		}
		return false, nil, nil
	})
}

// CreateAndWait 创建沙箱并等待其就绪。
func (c *Client) CreateAndWait(ctx context.Context, params CreateParams, opts ...PollOption) (*Sandbox, *SandboxInfo, error) {
	sb, err := c.Create(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("create sandbox: %w", err)
	}
	info, err := sb.WaitForReady(ctx, opts...)
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
		return newAPIError(resp.StatusCode(), resp.Body)
	}
	return nil
}

// GetSandboxesMetrics 返回指定沙箱 ID 列表的指标数据。
func (c *Client) GetSandboxesMetrics(ctx context.Context, params *GetSandboxesMetricsParams) (*SandboxesWithMetrics, error) {
	resp, err := c.api.GetSandboxesMetricsWithResponse(ctx, params)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.StatusCode(), resp.Body)
	}
	return sandboxesWithMetricsFromAPI(resp.JSON200), nil
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
		s.commands = newCommands(s, s.processClient())
	})
	return s.commands
}

// Pty 返回 PTY 终端操作接口。
func (s *Sandbox) Pty() *Pty {
	s.ptyOnce.Do(func() {
		s.pty = newPty(s, s.processClient())
	})
	return s.pty
}

// GetHost 返回访问沙箱指定端口的外部域名。
// 格式: {port}-{sandboxID}.{domain}
func (s *Sandbox) GetHost(port int) string {
	domain := s.client.config.Domain
	if s.domain != nil && *s.domain != "" {
		domain = *s.domain
	}
	return fmt.Sprintf("%d-%s.%s", port, s.sandboxID, domain)
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

// setEnvdAuth 将 envd 认证头设置到 ConnectRPC 请求。
func setEnvdAuth[T any](req *connect.Request[T], user string) {
	cred := base64.StdEncoding.EncodeToString([]byte(user + ":"))
	req.Header().Set("Authorization", "Basic "+cred)
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
//
// 注意: 此签名算法由后端服务定义，SDK 端需与服务端保持一致，不可单独修改。
// 当前算法未使用 HMAC，存在已知 accessToken 情况下的签名伪造风险，
// 后续安全加固需由服务端统一推进。
//
// 当前使用 ":" 作为字段分隔符，若 path 或 username 包含 ":"，可能导致签名碰撞。
// 此问题需由后端统一修复（如切换到不可见分隔符或对字段做转义）。
func fileSignature(path, operation, username, accessToken string, expiration int) string {
	raw := fmt.Sprintf("%s:%s:%s:%s:%d", path, operation, username, accessToken, expiration)
	hash := sha256.Sum256([]byte(raw))
	return "v1_" + fmt.Sprintf("%x", hash)
}

// DownloadURL 返回从沙箱下载文件的 URL。
func (s *Sandbox) DownloadURL(path string, opts ...FileURLOption) string {
	return s.fileURL(path, "read", opts...)
}

// UploadURL 返回向沙箱上传文件的 URL（POST multipart/form-data）。
func (s *Sandbox) UploadURL(path string, opts ...FileURLOption) string {
	return s.fileURL(path, "write", opts...)
}

// fileURL 构造带签名的 envd 文件操作 URL。
func (s *Sandbox) fileURL(path, operation string, opts ...FileURLOption) string {
	o := &fileURLOpts{user: DefaultUser}
	for _, fn := range opts {
		fn(o)
	}

	q := url.Values{}
	q.Set("path", path)
	q.Set("username", o.user)

	if s.envdAccessToken != nil && *s.envdAccessToken != "" {
		exp := o.signatureExpiration
		if exp == 0 {
			exp = 300
		}
		sig := fileSignature(path, operation, o.user, *s.envdAccessToken, exp)
		q.Set("signature", sig)
		q.Set("signature_expiration", strconv.Itoa(exp))
	}

	return s.envdURL() + "/files?" + q.Encode()
}

// batchUploadURL 返回批量上传文件的 URL。
// 与 UploadURL 不同，不设置 path 查询参数，文件路径由 multipart part filename 提供。
func (s *Sandbox) batchUploadURL(user string) string {
	q := url.Values{}
	q.Set("username", user)
	return s.envdURL() + "/files?" + q.Encode()
}
