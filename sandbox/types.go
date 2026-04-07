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

	// Injections 针对出站 HTTPS 请求的注入规则列表，可选。
	// 每项可引用已保存的注入规则 ID，或直接指定注入配置。
	Injections *[]SandboxInjectionSpec
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

func (p *CreateParams) toAPI() (apis.CreateSandboxJSONRequestBody, error) {
	body := apis.CreateSandboxJSONRequestBody{
		TemplateID:          p.TemplateID,
		Timeout:             p.Timeout,
		AutoPause:           p.AutoPause,
		AllowInternetAccess: p.AllowInternetAccess,
		Secure:              p.Secure,
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
	if p.Injections != nil {
		siList := make([]apis.SandboxInjection, len(*p.Injections))
		for i, spec := range *p.Injections {
			si, err := sandboxInjectionSpecToAPI(spec)
			if err != nil {
				return body, err
			}
			siList[i] = si
		}
		body.Injections = &siList
	}
	return body, nil
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
