package sandbox

import (
	"time"

	"github.com/qiniu/go-sdk/v7/sandbox/internal/apis"
)

// ---------------------------------------------------------------------------
// SDK 自有类型 — 沙箱相关
// ---------------------------------------------------------------------------

// Metadata 沙箱自定义元数据（key-value）。
type Metadata map[string]string

// NetworkConfig 沙箱网络配置。
type NetworkConfig struct {
	// AllowOut 允许的出站流量 CIDR 列表。
	AllowOut *[]string

	// AllowPublicTraffic 是否允许公共流量访问沙箱 URL。
	AllowPublicTraffic *bool

	// DenyOut 拒绝的出站流量 CIDR 列表。
	DenyOut *[]string

	// MaskRequestHost 用于沙箱请求的 host 掩码。
	MaskRequestHost *string
}

// RequestInjectionConditions 请求注入的匹配条件。
type RequestInjectionConditions struct {
	// Hosts 需要精确匹配的目标 HTTPS 域名列表（不支持通配符）。
	// 注入仅作用于这些域名的 443 端口 HTTPS 请求。
	// 当为空且设置了 API 时，自动使用协议默认域名。
	Hosts *[]string
}

// APIKeyInjectionType 预定义的 API 协议标识。
type APIKeyInjectionType = apis.APIKeyInjectionType

// APIKeyInjectionType 常量。
const (
	APIKeyInjectionTypeOpenAI    APIKeyInjectionType = "openai"
	APIKeyInjectionTypeAnthropic APIKeyInjectionType = "anthropic"
	APIKeyInjectionTypeGemini    APIKeyInjectionType = "gemini"
)

// APIKeyInjection 简化的 API 密钥注入配置，用于已知 API 协议。
type APIKeyInjection struct {
	// Type 预定义的 API 协议标识（openai、anthropic、gemini）。
	Type APIKeyInjectionType

	// Value API 密钥或令牌，可选。
	Value *string
}

// RequestInjections 请求注入的动作配置。
// 支持两种互斥模式：API 密钥模式（api）和手动模式（headers/queries）。
type RequestInjections struct {
	// API 简化的 API 密钥注入，用于已知 API 协议。与 Headers/Queries 互斥。
	API *APIKeyInjection

	// Headers 需要注入或覆盖的 HTTP Headers。与 API 互斥。
	Headers *map[string]string

	// Queries 需要替换的 URL Query 参数（仅在原请求中存在时有效）。与 API 互斥。
	Queries *map[string]string
}

// RequestInjection 定义对运行在沙箱内部的出站 HTTPS 请求进行拦截与自动注入的规则。
type RequestInjection struct {
	// Conditions 匹配条件。
	Conditions *RequestInjectionConditions

	// Injections 注入动作。
	Injections *RequestInjections
}

// SandboxState 沙箱状态。
type SandboxState string

// 沙箱状态常量。
const (
	StateRunning SandboxState = "running"
	StatePaused  SandboxState = "paused"
)

// CreateParams 创建沙箱的请求参数。
type CreateParams struct {
	// TemplateID 模板 ID（必填）。
	TemplateID string

	// Timeout 沙箱超时时间（秒），可选。
	Timeout *int32

	// AutoPause 超时后自动暂停，可选。
	AutoPause *bool

	// AllowInternetAccess 允许沙箱访问互联网，可选。
	AllowInternetAccess *bool

	// Secure 安全通信模式，可选。
	Secure *bool

	// EnvVars 环境变量，可选。
	EnvVars *map[string]string

	// Metadata 自定义元数据，可选。
	Metadata *Metadata

	// Network 网络配置，可选。
	Network *NetworkConfig

	// RequestInjections 针对出站 HTTPS 请求的注入规则列表，可选。
	// 与 RequestInjectionIds 不能同时设置。
	RequestInjections *[]RequestInjection

	// RequestInjectionIds 指定预定义 InjectionRule id 列表，可选。
	// 与 RequestInjections 不能同时设置。
	RequestInjectionIds *[]string
}

// ConnectParams 连接沙箱的请求参数。
type ConnectParams struct {
	// Timeout 超时时间（秒）。
	Timeout int32
}

// RefreshParams 延长沙箱存活时间的请求参数。
type RefreshParams struct {
	// Duration 延长的秒数，可选。
	Duration *int
}

// ListParams 列出沙箱的查询参数，支持分页和状态过滤。
type ListParams struct {
	// Metadata 用于过滤沙箱的元数据查询（如 "user=abc&app=prod"）。
	Metadata *string

	// State 按一个或多个状态过滤沙箱。
	State *[]SandboxState

	// NextToken 分页游标。
	NextToken *string

	// Limit 每页最大返回数。
	Limit *int32
}

// GetMetricsParams 获取沙箱指标的查询参数。
type GetMetricsParams struct {
	// Start 起始时间的 Unix 时间戳（秒）。
	Start *int64

	// End 结束时间的 Unix 时间戳（秒）。
	End *int64
}

func (p *GetMetricsParams) toAPI() *apis.GetSandboxMetricsParams {
	if p == nil {
		return nil
	}
	return &apis.GetSandboxMetricsParams{
		Start: p.Start,
		End:   p.End,
	}
}

// GetLogsParams 获取沙箱日志的查询参数。
type GetLogsParams struct {
	// Start 日志起始时间的毫秒级时间戳。
	Start *int64

	// Limit 返回的最大日志条数。
	Limit *int32
}

func (p *GetLogsParams) toAPI() *apis.GetSandboxLogsParams {
	if p == nil {
		return nil
	}
	return &apis.GetSandboxLogsParams{
		Start: p.Start,
		Limit: p.Limit,
	}
}

// GetSandboxesMetricsParams 批量获取沙箱指标的查询参数。
type GetSandboxesMetricsParams struct {
	// SandboxIds 要获取指标的沙箱 ID 列表。
	SandboxIds []string
}

func (p *GetSandboxesMetricsParams) toAPI() *apis.GetSandboxesMetricsParams {
	if p == nil {
		return nil
	}
	return &apis.GetSandboxesMetricsParams{
		SandboxIds: p.SandboxIds,
	}
}

// SandboxInfo 沙箱详细信息。
type SandboxInfo struct {
	SandboxID   string
	TemplateID  string
	ClientID    string
	Alias       *string
	Domain      *string
	State       SandboxState
	CPUCount    int32
	MemoryMB    int32
	DiskSizeMB  int32
	EnvdVersion string
	StartedAt   time.Time
	EndAt       time.Time
	Metadata    *Metadata
}

// ListedSandbox 沙箱列表中的条目。
type ListedSandbox struct {
	SandboxID   string
	TemplateID  string
	ClientID    string
	Alias       *string
	State       SandboxState
	CPUCount    int32
	MemoryMB    int32
	DiskSizeMB  int32
	EnvdVersion string
	StartedAt   time.Time
	EndAt       time.Time
	Metadata    *Metadata
}

// SandboxMetric 沙箱资源指标。
type SandboxMetric struct {
	CPUCount      int32
	CPUUsedPct    float32
	MemTotal      int64
	MemUsed       int64
	DiskTotal     int64
	DiskUsed      int64
	Timestamp     time.Time
	TimestampUnix int64
}

// SandboxLogs 沙箱日志。
type SandboxLogs struct {
	Logs       []SandboxLog
	LogEntries []SandboxLogEntry
}

// SandboxLog 沙箱日志条目。
type SandboxLog struct {
	Line      string
	Timestamp time.Time
}

// SandboxLogEntry 结构化沙箱日志条目。
type SandboxLogEntry struct {
	Level     LogLevel
	Message   string
	Fields    map[string]string
	Timestamp time.Time
}

// SandboxesWithMetrics 批量沙箱指标数据。
type SandboxesWithMetrics struct {
	Sandboxes map[string]SandboxMetric
}

// ---------------------------------------------------------------------------
// 转换函数 — apis → SDK
// ---------------------------------------------------------------------------

func sandboxInfoFromAPI(d *apis.SandboxDetail) *SandboxInfo {
	if d == nil {
		return nil
	}
	info := &SandboxInfo{
		SandboxID:   d.SandboxID,
		TemplateID:  d.TemplateID,
		ClientID:    d.ClientID,
		Alias:       d.Alias,
		Domain:      d.Domain,
		State:       SandboxState(d.State),
		CPUCount:    d.CPUCount,
		MemoryMB:    d.MemoryMB,
		DiskSizeMB:  d.DiskSizeMB,
		EnvdVersion: d.EnvdVersion,
		StartedAt:   d.StartedAt,
		EndAt:       d.EndAt,
	}
	if d.Metadata != nil {
		m := Metadata(*d.Metadata)
		info.Metadata = &m
	}
	return info
}

func listedSandboxFromAPI(a apis.ListedSandbox) ListedSandbox {
	ls := ListedSandbox{
		SandboxID:   a.SandboxID,
		TemplateID:  a.TemplateID,
		ClientID:    a.ClientID,
		Alias:       a.Alias,
		State:       SandboxState(a.State),
		CPUCount:    a.CPUCount,
		MemoryMB:    a.MemoryMB,
		DiskSizeMB:  a.DiskSizeMB,
		EnvdVersion: a.EnvdVersion,
		StartedAt:   a.StartedAt,
		EndAt:       a.EndAt,
	}
	if a.Metadata != nil {
		m := Metadata(*a.Metadata)
		ls.Metadata = &m
	}
	return ls
}

func listedSandboxesFromAPI(a []apis.ListedSandbox) []ListedSandbox {
	if a == nil {
		return nil
	}
	result := make([]ListedSandbox, len(a))
	for i, s := range a {
		result[i] = listedSandboxFromAPI(s)
	}
	return result
}

func sandboxMetricFromAPI(a apis.SandboxMetric) SandboxMetric {
	return SandboxMetric{
		CPUCount:      a.CPUCount,
		CPUUsedPct:    a.CPUUsedPct,
		MemTotal:      a.MemTotal,
		MemUsed:       a.MemUsed,
		DiskTotal:     a.DiskTotal,
		DiskUsed:      a.DiskUsed,
		Timestamp:     a.Timestamp,
		TimestampUnix: a.TimestampUnix,
	}
}

func sandboxMetricsFromAPI(a []apis.SandboxMetric) []SandboxMetric {
	if a == nil {
		return nil
	}
	result := make([]SandboxMetric, len(a))
	for i, m := range a {
		result[i] = sandboxMetricFromAPI(m)
	}
	return result
}

func sandboxLogsFromAPI(a *apis.SandboxLogs) *SandboxLogs {
	if a == nil {
		return nil
	}
	result := &SandboxLogs{
		Logs:       make([]SandboxLog, 0, len(a.Logs)),
		LogEntries: make([]SandboxLogEntry, 0, len(a.LogEntries)),
	}
	for _, l := range a.Logs {
		result.Logs = append(result.Logs, SandboxLog{Line: l.Line, Timestamp: l.Timestamp})
	}
	for _, e := range a.LogEntries {
		result.LogEntries = append(result.LogEntries, SandboxLogEntry{
			Level:     LogLevel(e.Level),
			Message:   e.Message,
			Fields:    e.Fields,
			Timestamp: e.Timestamp,
		})
	}
	return result
}

func sandboxesWithMetricsFromAPI(a *apis.SandboxesWithMetrics) *SandboxesWithMetrics {
	if a == nil {
		return nil
	}
	result := &SandboxesWithMetrics{Sandboxes: make(map[string]SandboxMetric, len(a.Sandboxes))}
	for k, v := range a.Sandboxes {
		result.Sandboxes[k] = sandboxMetricFromAPI(v)
	}
	return result
}

// ---------------------------------------------------------------------------
// 转换函数 — SDK → apis
// ---------------------------------------------------------------------------

func (p *CreateParams) toAPI() apis.CreateSandboxJSONRequestBody {
	body := apis.CreateSandboxJSONRequestBody{
		TemplateID:          p.TemplateID,
		Timeout:             p.Timeout,
		AutoPause:           p.AutoPause,
		AllowInternetAccess: p.AllowInternetAccess,
		Secure:              p.Secure,
		RequestInjectionIds: p.RequestInjectionIds,
	}
	if p.EnvVars != nil {
		ev := apis.EnvVars(*p.EnvVars)
		body.EnvVars = &ev
	}
	if p.Metadata != nil {
		m := apis.SandboxMetadata(*p.Metadata)
		body.Metadata = &m
	}
	if p.Network != nil {
		body.Network = &apis.SandboxNetworkConfig{
			AllowOut:           p.Network.AllowOut,
			AllowPublicTraffic: p.Network.AllowPublicTraffic,
			DenyOut:            p.Network.DenyOut,
			MaskRequestHost:    p.Network.MaskRequestHost,
		}
	}
	if p.RequestInjections != nil {
		riList := make([]apis.RequestInjection, len(*p.RequestInjections))
		for i, ri := range *p.RequestInjections {
			var conditions *apis.RequestInjectionConditions
			var injections *apis.RequestInjections

			if ri.Conditions != nil {
				conditions = &apis.RequestInjectionConditions{
					Hosts: ri.Conditions.Hosts,
				}
			}
			if ri.Injections != nil {
				injections = &apis.RequestInjections{
					Headers: ri.Injections.Headers,
					Queries: ri.Injections.Queries,
				}
				if ri.Injections.API != nil {
					injections.API = &apis.APIKeyInjection{
						Type:  ri.Injections.API.Type,
						Value: ri.Injections.API.Value,
					}
				}
			}

			riList[i] = apis.RequestInjection{
				Conditions: conditions,
				Injections: injections,
			}
		}
		body.RequestInjections = &riList
	}
	return body
}

func (p *ConnectParams) toAPI() apis.ConnectSandboxJSONRequestBody {
	return apis.ConnectSandboxJSONRequestBody{
		Timeout: p.Timeout,
	}
}

func (p *RefreshParams) toAPI() apis.RefreshSandboxJSONRequestBody {
	return apis.RefreshSandboxJSONRequestBody{
		Duration: p.Duration,
	}
}

func (p *ListParams) toAPI() *apis.ListSandboxesV2Params {
	if p == nil {
		return nil
	}
	params := &apis.ListSandboxesV2Params{
		Metadata:  p.Metadata,
		NextToken: p.NextToken,
		Limit:     p.Limit,
	}
	if p.State != nil {
		states := make([]apis.SandboxState, len(*p.State))
		for i, s := range *p.State {
			states[i] = apis.SandboxState(s)
		}
		params.State = &states
	}
	return params
}
