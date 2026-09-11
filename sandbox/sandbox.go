package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
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

	gitOnce sync.Once
	git     *Git
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

// GetInfo 返回当前沙箱的详细信息。
// 具体行为参见 [Client.GetInfo]。
func (s *Sandbox) GetInfo(ctx context.Context) (*SandboxInfo, error) {
	return s.client.GetInfo(ctx, s.sandboxID)
}

// GetMetrics 返回当前沙箱的资源指标。
// 具体行为参见 [Client.GetMetrics]。
func (s *Sandbox) GetMetrics(ctx context.Context, params *GetMetricsParams) ([]SandboxMetric, error) {
	return s.client.GetMetrics(ctx, s.sandboxID, params)
}

// GetLogs 返回当前沙箱的日志。
// 具体行为参见 [Client.GetLogs]。
func (s *Sandbox) GetLogs(ctx context.Context, params *GetLogsParams) (*SandboxLogs, error) {
	return s.client.GetLogs(ctx, s.sandboxID, params)
}

// GetResources 返回当前沙箱已挂载的资源配置。
// 具体行为参见 [Client.GetResources]。
func (s *Sandbox) GetResources(ctx context.Context) ([]SandboxResourceInfo, error) {
	return s.client.GetResources(ctx, s.sandboxID)
}

// GetInjections 返回当前沙箱的运行时请求注入规则。
// 具体行为参见 [Client.GetInjections]。
func (s *Sandbox) GetInjections(ctx context.Context) ([]MaskedSandboxInjection, error) {
	return s.client.GetInjections(ctx, s.sandboxID)
}

// Kill 终止当前沙箱。
// 具体行为参见 [Client.Kill]。
func (s *Sandbox) Kill(ctx context.Context) error {
	return s.client.Kill(ctx, s.sandboxID)
}

// Pause 暂停当前沙箱。
// 具体行为参见 [Client.Pause]。
func (s *Sandbox) Pause(ctx context.Context) error {
	return s.client.Pause(ctx, s.sandboxID)
}

// Refresh 延长当前沙箱的存活时间。
// 具体行为参见 [Client.Refresh]。
func (s *Sandbox) Refresh(ctx context.Context, params RefreshParams) error {
	return s.client.Refresh(ctx, s.sandboxID, params)
}

// SetTimeout 更新当前沙箱的超时时间，timeout 必须至少为 1 秒。
// 具体行为参见 [Client.SetTimeout]。
func (s *Sandbox) SetTimeout(ctx context.Context, timeout time.Duration) error {
	return s.client.SetTimeout(ctx, s.sandboxID, timeout)
}

// UpdateInjections 替换当前沙箱的全部运行时请求注入规则。
// 具体行为参见 [Client.UpdateInjections]。
func (s *Sandbox) UpdateInjections(ctx context.Context, injections []SandboxInjectionSpec) error {
	return s.client.UpdateInjections(ctx, s.sandboxID, injections)
}

// UpdateGitHubToken 更新当前沙箱使用的 GitHub 授权令牌。
// 具体行为参见 [Client.UpdateGitHubToken]。
func (s *Sandbox) UpdateGitHubToken(ctx context.Context, authorizationToken string) error {
	return s.client.UpdateGitHubToken(ctx, s.sandboxID, authorizationToken)
}

// UpdateGitRepositoryResourceToken 更新当前沙箱中指定 Git 仓库资源的授权令牌。
// 具体行为参见 [Client.UpdateGitRepositoryResourceToken]。
func (s *Sandbox) UpdateGitRepositoryResourceToken(ctx context.Context, resourceID, authorizationToken string) error {
	return s.client.UpdateGitRepositoryResourceToken(ctx, s.sandboxID, resourceID, authorizationToken)
}

// WaitForReady 轮询当前沙箱的状态，直到进入 running 状态或上下文被取消。
// 具体行为参见 [Client.WaitForReady]。
func (s *Sandbox) WaitForReady(ctx context.Context, opts ...PollOption) (*SandboxInfo, error) {
	return s.client.WaitForReady(ctx, s.sandboxID, opts...)
}

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
// 默认自动生成幂等键，在遇到服务端错误或网络错误时自动重试。
func (c *Client) Create(ctx context.Context, params CreateParams) (*Sandbox, error) {
	apiParams, err := params.toAPI()
	if err != nil {
		return nil, err
	}
	var editors []apis.RequestEditorFn
	if params.hasKodoResource() {
		cred, err := c.GetCredentialsOption()
		if err != nil {
			return nil, err
		}
		if cred == nil {
			return nil, fmt.Errorf("kodo resource requires Qiniu AK/SK credentials, please configure them in sandbox.Config")
		}
		editors = append(editors, cred)
	}
	idempotencyKey := params.IdempotencyKey
	if idempotencyKey == "" {
		idempotencyKey = newIdempotencyKey()
	}
	var resp *apis.CreateSandboxResponse
	err = c.retryCall(ctx, func() error {
		var e error
		resp, e = c.api.CreateSandboxWithResponse(ctx, &apis.CreateSandboxParams{
			IdempotencyKey: &idempotencyKey,
		}, apiParams, editors...)
		if e != nil {
			return e
		}
		if resp.JSON201 == nil {
			return newAPIError(resp.HTTPResponse, resp.Body)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sb := newSandbox(c, resp.JSON201)
	if !sb.envdTokenLoaded {
		if tErr := sb.refreshEnvdToken(ctx); tErr != nil {
			return nil, fmt.Errorf("create sandbox %s: %w", sb.sandboxID, tErr)
		}
	}
	return sb, nil
}

// Connect 连接到一个已有的沙箱，可选择恢复已暂停的沙箱。
// 在遇到服务端错误或网络错误时自动重试。
func (c *Client) Connect(ctx context.Context, sandboxID string, params ConnectParams) (*Sandbox, error) {
	var resp *apis.ConnectSandboxResponse
	err := c.retryCall(ctx, func() error {
		var e error
		resp, e = c.api.ConnectSandboxWithResponse(ctx, sandboxID, params.toAPI())
		if e != nil {
			return e
		}
		if resp.JSON200 == nil && resp.JSON201 == nil {
			return newAPIError(resp.HTTPResponse, resp.Body)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var sandboxResp *apis.Sandbox
	if resp.JSON200 != nil {
		sandboxResp = resp.JSON200
	} else {
		sandboxResp = resp.JSON201
	}
	sb := newSandbox(c, sandboxResp)
	// Connect API 可能不返回 envdAccessToken，需要通过 GetSandbox 补充。
	// envdAccessToken 用于 envd gRPC 认证，缺少时 PTY/命令执行等操作会静默失败。
	if !sb.envdTokenLoaded {
		if tErr := sb.refreshEnvdToken(ctx); tErr != nil {
			return nil, fmt.Errorf("connect sandbox %s: %w", sandboxID, tErr)
		}
	}
	return sb, nil
}

// retryCall 执行 API 调用并在可重试错误时自动重试。
// 重试次数由 Config.RetryMax 控制（nil 默认 5，0 禁用重试）。
func (c *Client) retryCall(ctx context.Context, fn func() error) error {
	maxRetries := 5
	if c.config.RetryMax != nil {
		maxRetries = *c.config.RetryMax
	}
	backoffFn := c.config.RetryBackoff
	if backoffFn == nil {
		backoffFn = exponentialBackoff
	}
	for attempt := 0; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := fn()
		if err == nil {
			return nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if attempt >= maxRetries || !isRetryable(err) {
			return err
		}
		d := backoffFn(attempt)
		if d > 0 {
			timer := time.NewTimer(d)
			select {
			case <-timer.C:
			case <-ctx.Done():
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				return ctx.Err()
			}
		}
	}
}

// exponentialBackoff 指数退避（500ms → 1s → 2s → 4s → 8s，上限 10s）+ 随机抖动。
func exponentialBackoff(attempt int) time.Duration {
	const maxBackoff = 10 * time.Second
	base := 500 * time.Millisecond
	if attempt >= 5 {
		base = maxBackoff
	} else if attempt > 0 {
		base *= time.Duration(1 << attempt)
	}
	if base > maxBackoff {
		base = maxBackoff
	}
	jitter := time.Duration(rand.Int64N(int64(base/2 + 1)))
	return base + jitter
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

// GetResources 返回指定沙箱已挂载的资源配置。
// 响应中的访问密钥和授权令牌等敏感字段由服务端脱敏。
func (c *Client) GetResources(ctx context.Context, sandboxID string) ([]SandboxResourceInfo, error) {
	resp, err := c.api.GetSandboxResourcesWithResponse(ctx, sandboxID)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	return sandboxResourceInfosFromAPI(resp.JSON200.Resources)
}

// UpdateGitRepositoryResourceToken 更新指定沙箱中 Git 仓库资源的授权令牌。
// 对正在运行的沙箱，新令牌会立即应用到对应的 GitHub 注入配置。
func (c *Client) UpdateGitRepositoryResourceToken(ctx context.Context, sandboxID, resourceID, authorizationToken string) error {
	resp, err := c.api.PatchSandboxResourceWithResponse(ctx, sandboxID, resourceID, apis.PatchSandboxResourceJSONRequestBody{
		AuthorizationToken: &authorizationToken,
	})
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != http.StatusNoContent {
		return newAPIError(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// GetInfo 返回指定沙箱的详细信息。
func (c *Client) GetInfo(ctx context.Context, sandboxID string) (*SandboxInfo, error) {
	resp, err := c.api.GetSandboxWithResponse(ctx, sandboxID)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	return sandboxInfoFromAPI(resp.JSON200), nil
}

// GetMetrics 返回指定沙箱的资源指标。
func (c *Client) GetMetrics(ctx context.Context, sandboxID string, params *GetMetricsParams) ([]SandboxMetric, error) {
	resp, err := c.api.GetSandboxMetricsWithResponse(ctx, sandboxID, params.toAPI())
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	return sandboxMetricsFromAPI(*resp.JSON200), nil
}

// GetLogs 返回指定沙箱的日志。
func (c *Client) GetLogs(ctx context.Context, sandboxID string, params *GetLogsParams) (*SandboxLogs, error) {
	resp, err := c.api.GetSandboxLogsWithResponse(ctx, sandboxID, params.toAPI())
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	return sandboxLogsFromAPI(resp.JSON200), nil
}

// Kill 终止指定沙箱。
func (c *Client) Kill(ctx context.Context, sandboxID string) error {
	resp, err := c.api.DeleteSandboxWithResponse(ctx, sandboxID)
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != http.StatusNoContent {
		return newAPIError(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// Pause 暂停指定沙箱。
func (c *Client) Pause(ctx context.Context, sandboxID string) error {
	resp, err := c.api.PauseSandboxWithResponse(ctx, sandboxID)
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != http.StatusNoContent {
		return newAPIError(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// GetInjections 返回指定沙箱当前的运行时请求注入规则。
// 响应中的密钥、令牌和 Header 等敏感字段由服务端脱敏。
// 返回值不能直接用于 UpdateInjections；更新时必须提供包含真实敏感值的新配置。
func (c *Client) GetInjections(ctx context.Context, sandboxID string) ([]MaskedSandboxInjection, error) {
	resp, err := c.api.GetSandboxInjectionsWithResponse(ctx, sandboxID)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	return maskedSandboxInjectionsFromAPI(resp.JSON200.Injections)
}

// UpdateInjections 替换指定沙箱的全部运行时请求注入规则。
// 变更会立即应用于新的出站 HTTPS 连接。
func (c *Client) UpdateInjections(ctx context.Context, sandboxID string, injections []SandboxInjectionSpec) error {
	apiInjections := make([]apis.SandboxInjection, len(injections))
	for i, injection := range injections {
		apiInjection, err := sandboxInjectionSpecToAPI(injection)
		if err != nil {
			return err
		}
		apiInjections[i] = apiInjection
	}
	resp, err := c.api.UpdateSandboxInjectionsWithResponse(ctx, sandboxID, apis.UpdateSandboxInjectionsJSONRequestBody{
		Injections: apiInjections,
	})
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != http.StatusNoContent {
		return newAPIError(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// UpdateGitHubToken 更新指定沙箱使用的 GitHub 授权令牌。
func (c *Client) UpdateGitHubToken(ctx context.Context, sandboxID, authorizationToken string) error {
	resp, err := c.api.UpdateSandboxGithubTokenWithResponse(ctx, sandboxID, apis.UpdateSandboxGithubTokenJSONRequestBody{
		AuthorizationToken: authorizationToken,
	})
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != http.StatusNoContent {
		return newAPIError(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// SetTimeout 更新指定沙箱的超时时间。
// 沙箱将在从现在起经过指定时长后过期。
// timeout 必须 >= 1 秒。
func (c *Client) SetTimeout(ctx context.Context, sandboxID string, timeout time.Duration) error {
	if timeout < time.Second {
		return fmt.Errorf("timeout must be at least 1 second, got %v", timeout)
	}
	secs := timeout.Seconds()
	if secs > float64(math.MaxInt32) {
		return fmt.Errorf("timeout %v exceeds maximum allowed value", timeout)
	}
	resp, err := c.api.UpdateSandboxTimeoutWithResponse(ctx, sandboxID, apis.UpdateSandboxTimeoutJSONRequestBody{
		Timeout: int32(secs),
	})
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != http.StatusNoContent {
		return newAPIError(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// Refresh 延长指定沙箱的存活时间。
func (c *Client) Refresh(ctx context.Context, sandboxID string, params RefreshParams) error {
	resp, err := c.api.RefreshSandboxWithResponse(ctx, sandboxID, params.toAPI())
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != http.StatusNoContent {
		return newAPIError(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// WaitForReady 轮询指定沙箱的状态，直到变为 "running" 或上下文被取消。
// 默认轮询间隔为 1 秒，可通过 WithPollInterval 等选项自定义。
func (c *Client) WaitForReady(ctx context.Context, sandboxID string, opts ...PollOption) (*SandboxInfo, error) {
	o := defaultPollOpts(time.Second)
	for _, fn := range opts {
		fn(o)
	}

	return pollLoop(ctx, o, func() (bool, *SandboxInfo, error) {
		info, err := c.GetInfo(ctx, sandboxID)
		if err != nil {
			return false, nil, fmt.Errorf("get sandbox %s: %w", sandboxID, err)
		}
		if info.State == StateRunning {
			return true, info, nil
		}
		return false, nil, nil
	})
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

// CreateAndWait 创建沙箱并等待其就绪。
func (c *Client) CreateAndWait(ctx context.Context, params CreateParams, opts ...PollOption) (*Sandbox, *SandboxInfo, error) {
	sb, err := c.Create(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("create sandbox: %w", err)
	}
	info, err := c.WaitForReady(ctx, sb.ID(), opts...)
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

// Git 返回 git 操作接口。
// 沙箱内需预装 git 二进制；当前仅支持 HTTPS + username/password (token) 认证。
func (s *Sandbox) Git() *Git {
	s.gitOnce.Do(func() {
		s.git = newGit(s.Commands())
	})
	return s.git
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

// newIdempotencyKey 生成一个 UUID v4 格式的幂等键。
func newIdempotencyKey() string {
	return uuid.New().String()
}

// isRetryableStatusCode 判断 HTTP 状态码是否可重试。
// 可重试：408（超时）及 5xx 服务端错误（排除 501）。
func isRetryableStatusCode(code int) bool {
	if code == 408 {
		return true
	}
	return code >= 500 && code != 501
}

// isRetryableError 判断错误是否为网络层面的可重试错误。
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	s := err.Error()
	return strings.Contains(s, "connection refused") ||
		strings.Contains(s, "connection reset") ||
		strings.Contains(s, "broken pipe") ||
		strings.Contains(s, "no such host") ||
		strings.Contains(s, "unexpected EOF") ||
		strings.Contains(s, "use of closed network connection")
}

// isRetryable 判断错误是否可重试：APIError 按状态码判断，其余按网络错误判断。
func isRetryable(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return isRetryableStatusCode(apiErr.StatusCode)
	}
	return isRetryableError(err)
}
