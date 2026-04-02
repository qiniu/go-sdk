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

	"github.com/qiniu/go-sdk/v7/reqid"
	"github.com/qiniu/go-sdk/v7/sandbox/internal/apis"
	"github.com/qiniu/go-sdk/v7/sandbox/internal/envdapi/process/processconnect"

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
	trafficAccessToken *string

	// envdAccessToken 用于 envd 认证，需通过 envdTokenMu 保护并发读写。
	envdTokenMu     sync.RWMutex
	envdAccessToken *string
	envdTokenLoaded bool // 标记是否已尝试获取过 token（避免重复请求）

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
	sb := &Sandbox{
		sandboxID:          s.SandboxID,
		templateID:         s.TemplateID,
		clientID:           s.ClientID,
		alias:              s.Alias,
		domain:             s.Domain,
		trafficAccessToken: s.TrafficAccessToken,
		client:             c,
	}
	if s.EnvdAccessToken != nil {
		sb.envdAccessToken = s.EnvdAccessToken
		sb.envdTokenLoaded = true
	}
	return sb
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
		s.processRPC = processconnect.NewProcessClient(
			s.client.config.HTTPClient,
			s.envdURL(),
			connect.WithInterceptors(keepaliveInterceptor{}),
		)
	})
	return s.processRPC
}

// Create 根据指定模板创建一个新的沙箱。
func (c *Client) Create(ctx context.Context, params CreateParams) (*Sandbox, error) {
	apiParams, err := params.toAPI()
	if err != nil {
		return nil, err
	}
	resp, err := c.api.CreateSandboxWithResponse(ctx, apiParams)
	if err != nil {
		return nil, err
	}
	if resp.JSON201 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	sb := newSandbox(c, resp.JSON201)
	// Create API 可能不返回 envdAccessToken（与 Connect 同理），通过 GetSandbox 补充。
	if !sb.envdTokenLoaded {
		if err := sb.refreshEnvdToken(ctx); err != nil {
			return nil, fmt.Errorf("create sandbox %s: %w", sb.sandboxID, err)
		}
	}
	return sb, nil
}

// Connect 连接到一个已有的沙箱，可选择恢复已暂停的沙箱。
func (c *Client) Connect(ctx context.Context, sandboxID string, params ConnectParams) (*Sandbox, error) {
	resp, err := c.api.ConnectSandboxWithResponse(ctx, sandboxID, params.toAPI())
	if err != nil {
		return nil, err
	}
	var sb *Sandbox
	if resp.JSON200 != nil {
		sb = newSandbox(c, resp.JSON200)
	} else if resp.JSON201 != nil {
		sb = newSandbox(c, resp.JSON201)
	} else {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	// Connect API 可能不返回 envdAccessToken，需要通过 GetSandbox 补充。
	// envdAccessToken 用于 envd gRPC 认证，缺少时 PTY/命令执行等操作会静默失败。
	if !sb.envdTokenLoaded {
		if err := sb.refreshEnvdToken(ctx); err != nil {
			return nil, fmt.Errorf("connect sandbox %s: %w", sandboxID, err)
		}
	}
	return sb, nil
}

// List 列出沙箱，支持分页和状态过滤。
func (c *Client) List(ctx context.Context, params *ListParams) ([]ListedSandbox, error) {
	resp, err := c.api.ListSandboxesV2WithResponse(ctx, params.toAPI())
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
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
		return newAPIError(resp.HTTPResponse, resp.Body)
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
		return newAPIError(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// refreshEnvdToken 通过 GetSandbox API 获取 envdAccessToken 并更新到当前实例。
// 调用者必须确保已持有 envdTokenMu 的写锁或在初始化阶段调用。
func (s *Sandbox) refreshEnvdToken(ctx context.Context) error {
	resp, err := s.client.api.GetSandboxWithResponse(ctx, s.sandboxID)
	if err != nil {
		return fmt.Errorf("get sandbox %s for envd token: %w", s.sandboxID, err)
	}
	if resp.JSON200 == nil {
		return fmt.Errorf("get sandbox %s for envd token: %w", s.sandboxID, newAPIError(resp.HTTPResponse, resp.Body))
	}
	s.envdTokenMu.Lock()
	s.envdAccessToken = resp.JSON200.EnvdAccessToken
	s.envdTokenLoaded = true
	s.envdTokenMu.Unlock()
	return nil
}

// GetInfo 返回沙箱的详细信息。
func (s *Sandbox) GetInfo(ctx context.Context) (*SandboxInfo, error) {
	resp, err := s.client.api.GetSandboxWithResponse(ctx, s.sandboxID)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	return sandboxInfoFromAPI(resp.JSON200), nil
}

// IsRunning 通过探测 envd /health 端点检查沙箱是否正在运行且可用。
// 与 GetInfo（查询控制面状态）不同，此方法直接验证沙箱内部 agent 是否可达。
// 返回 true 表示沙箱运行中且 envd agent 已就绪；返回 false 表示沙箱不可达（已暂停、已终止等）。
func (s *Sandbox) IsRunning(ctx context.Context) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.envdURL()+"/health", nil)
	if err != nil {
		return false, err
	}
	setReqidHeader(ctx, req)
	resp, err := s.client.config.HTTPClient.Do(req)
	if err != nil {
		return false, err
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return true, nil
	}
	if resp.StatusCode == http.StatusBadGateway {
		return false, nil
	}
	return false, newAPIError(resp, nil)
}

// GetMetrics 返回沙箱的资源指标。
func (s *Sandbox) GetMetrics(ctx context.Context, params *GetMetricsParams) ([]SandboxMetric, error) {
	resp, err := s.client.api.GetSandboxMetricsWithResponse(ctx, s.sandboxID, params.toAPI())
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	return sandboxMetricsFromAPI(*resp.JSON200), nil
}

// GetLogs 返回沙箱日志。
func (s *Sandbox) GetLogs(ctx context.Context, params *GetLogsParams) (*SandboxLogs, error) {
	resp, err := s.client.api.GetSandboxLogsWithResponse(ctx, s.sandboxID, params.toAPI())
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
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
		return newAPIError(resp.HTTPResponse, resp.Body)
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
		return newAPIError(resp.HTTPResponse, resp.Body)
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

// GetSandboxesMetrics 返回指定沙箱 ID 列表的指标数据。
func (c *Client) GetSandboxesMetrics(ctx context.Context, params *GetSandboxesMetricsParams) (*SandboxesWithMetrics, error) {
	resp, err := c.api.GetSandboxesMetricsWithResponse(ctx, params.toAPI())
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
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
	if s.domain == nil || *s.domain == "" {
		return ""
	}
	return fmt.Sprintf("%d-%s.%s", port, s.sandboxID, *s.domain)
}

// envdURL 返回 envd agent 的基础 URL。
func (s *Sandbox) envdURL() string {
	return fmt.Sprintf("https://%s", s.GetHost(envdPort))
}

// envdBasicAuth 返回 envd 用户身份认证的 Authorization 头值。
// 格式为 Basic base64(username:)，仅用于 OS 用户身份标识，不含密码。
// envd 的访问控制通过独立的 X-Access-Token 头实现。
func envdBasicAuth(user string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"))
}

// setEnvdAuth 将 envd 认证头设置到 ConnectRPC 请求。
// 设置两个独立的 header：
//   - Authorization: Basic base64(user:) — OS 用户身份标识
//   - X-Access-Token: <token> — envd 访问控制（仅当 token 存在时）
func (s *Sandbox) setEnvdAuth(req interface{ Header() http.Header }, user string) {
	req.Header().Set("Authorization", envdBasicAuth(user))
	s.envdTokenMu.RLock()
	tok := s.envdAccessToken
	s.envdTokenMu.RUnlock()
	if tok != nil && *tok != "" {
		req.Header().Set("X-Access-Token", *tok)
	}
}

// keepalivePingIntervalSec 是 keepalive ping 间隔（秒），与 JS SDK 保持一致。
// envd 服务端会按此间隔在 gRPC 流中发送 keepalive 消息，防止代理/LB 断开空闲连接。
const keepalivePingIntervalSec = "50"

// keepalivePingHeader 是 keepalive ping 间隔的 HTTP header 名。
const keepalivePingHeader = "Keepalive-Ping-Interval"

// keepaliveInterceptor 是一个 ConnectRPC 拦截器，为所有流式请求注入 Keepalive-Ping-Interval header。
type keepaliveInterceptor struct{}

func (keepaliveInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return next
}

func (keepaliveInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		conn.RequestHeader().Set(keepalivePingHeader, keepalivePingIntervalSec)
		return conn
	}
}

func (keepaliveInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

// setReqidHeader 从 context 中提取 reqid 并注入到 HTTP 请求头。
// 用于绕过 oapi-codegen 客户端的直接 HTTP 调用（如 envd API）。
func setReqidHeader(ctx context.Context, req *http.Request) {
	if id, ok := reqid.ReqidFromContext(ctx); ok {
		req.Header.Set("X-Reqid", id)
	}
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

	s.envdTokenMu.RLock()
	tok := s.envdAccessToken
	s.envdTokenMu.RUnlock()
	if tok != nil && *tok != "" {
		exp := o.signatureExpiration
		if exp == 0 {
			exp = 300
		}
		sig := fileSignature(path, operation, o.user, *tok, exp)
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
