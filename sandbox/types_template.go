package sandbox

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/qiniu/go-sdk/v7/sandbox/internal/apis"
)

// ---------------------------------------------------------------------------
// SDK 自有类型 — 模板相关
// ---------------------------------------------------------------------------

// TemplateBuildStatus 模板构建状态。
type TemplateBuildStatus string

// 模板构建状态常量。
const (
	BuildStatusReady    TemplateBuildStatus = "ready"
	BuildStatusError    TemplateBuildStatus = "error"
	BuildStatusBuilding TemplateBuildStatus = "building"
	BuildStatusWaiting  TemplateBuildStatus = "waiting"
	BuildStatusUploaded TemplateBuildStatus = "uploaded"
)

// LogLevel 日志级别。
type LogLevel string

// 日志级别常量。
const (
	LogLevelDebug LogLevel = "debug"
	LogLevelError LogLevel = "error"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
)

// LogsDirection 日志方向。
type LogsDirection string

// 日志方向常量。
const (
	LogsDirectionBackward LogsDirection = "backward"
	LogsDirectionForward  LogsDirection = "forward"
)

// LogsSource 日志来源。
type LogsSource string

// 日志来源常量。
const (
	LogsSourcePersistent LogsSource = "persistent"
	LogsSourceTemporary  LogsSource = "temporary"
)

// TemplateStep 模板构建步骤。
type TemplateStep struct {
	// Args 步骤参数。
	Args *[]string

	// FilesHash 步骤中使用文件的哈希值。
	FilesHash *string

	// Force 是否强制执行（忽略缓存）。
	Force *bool

	// Type 步骤类型。
	Type string
}

func templateStepsToAPI(steps *[]TemplateStep) *[]apis.TemplateStep {
	if steps == nil {
		return nil
	}
	result := make([]apis.TemplateStep, len(*steps))
	for i, s := range *steps {
		result[i] = apis.TemplateStep{
			Args:      s.Args,
			FilesHash: s.FilesHash,
			Force:     s.Force,
			Type:      s.Type,
		}
	}
	return &result
}

// FromImageRegistry 镜像仓库认证配置（union 类型，支持 AWS/GCP/General 三种 registry）。
// 使用 json.RawMessage 保留原始 JSON 格式，由服务端解析具体类型。
type FromImageRegistry = json.RawMessage

// CreateTemplateParams 创建模板的请求参数。
type CreateTemplateParams struct {
	// Alias 模板别名（已废弃，请使用 Name）。
	Alias *string

	// CPUCount 沙箱 CPU 核数。
	CPUCount *int32

	// MemoryMB 沙箱内存大小（MiB）。
	MemoryMB *int32

	// Name 模板名称，可包含标签（如 "my-template" 或 "my-template:v1"）。
	Name *string

	// Tags 分配给模板构建的标签列表。
	Tags *[]string

	// TeamID 团队 ID（已废弃）。
	TeamID *string
}

func (p *CreateTemplateParams) toAPI() apis.CreateTemplateV3JSONRequestBody {
	return apis.CreateTemplateV3JSONRequestBody{
		Alias:    p.Alias,
		CPUCount: p.CPUCount,
		MemoryMB: p.MemoryMB,
		Name:     p.Name,
		Tags:     p.Tags,
		TeamID:   p.TeamID,
	}
}

// UpdateTemplateParams 更新模板的请求参数。
type UpdateTemplateParams struct {
	// Public 模板是否公开。
	Public *bool
}

func (p *UpdateTemplateParams) toAPI() apis.UpdateTemplateJSONRequestBody {
	return apis.UpdateTemplateJSONRequestBody{
		Public: p.Public,
	}
}

// RebuildTemplateParams 重新构建已有模板的请求参数。
// 对应 POST /templates/{templateID}：在已存在的模板上创建一个新的 waiting build，
// 返回新的 buildID，后续可通过 StartTemplateBuild 驱动该 build。
type RebuildTemplateParams struct {
	// Dockerfile 必填，模板使用的 Dockerfile 内容。
	Dockerfile string

	// Alias 模板别名。
	Alias *string

	// CPUCount 沙箱 CPU 核数。
	CPUCount *int32

	// MemoryMB 沙箱内存大小（MiB）。
	MemoryMB *int32

	// StartCmd 构建完成后执行的启动命令。
	StartCmd *string

	// ReadyCmd 就绪检查命令。
	ReadyCmd *string

	// TeamID 团队 ID（已废弃）。
	TeamID *string
}

func (p *RebuildTemplateParams) toAPI() apis.RebuildTemplateJSONRequestBody {
	return apis.RebuildTemplateJSONRequestBody{
		Alias:      p.Alias,
		CPUCount:   p.CPUCount,
		Dockerfile: p.Dockerfile,
		MemoryMB:   p.MemoryMB,
		ReadyCmd:   p.ReadyCmd,
		StartCmd:   p.StartCmd,
		TeamID:     p.TeamID,
	}
}

// StartTemplateBuildParams 启动模板构建的请求参数。
type StartTemplateBuildParams struct {
	// Force 是否强制完整构建（忽略缓存）。
	Force *bool

	// FromImage 用作模板构建基础的镜像。
	FromImage *string

	// FromImageRegistry 镜像仓库认证配置。
	FromImageRegistry *FromImageRegistry

	// FromTemplate 用作模板构建基础的模板。
	FromTemplate *string

	// ReadyCmd 构建完成后执行的就绪检查命令。
	ReadyCmd *string

	// StartCmd 构建完成后执行的启动命令。
	StartCmd *string

	// Steps 模板构建步骤列表。
	Steps *[]TemplateStep
}

func (p *StartTemplateBuildParams) toAPI() (apis.StartTemplateBuildV2JSONRequestBody, error) {
	body := apis.StartTemplateBuildV2JSONRequestBody{
		Force:        p.Force,
		FromImage:    p.FromImage,
		FromTemplate: p.FromTemplate,
		ReadyCmd:     p.ReadyCmd,
		StartCmd:     p.StartCmd,
		Steps:        templateStepsToAPI(p.Steps),
	}
	if p.FromImageRegistry != nil {
		reg := apis.FromImageRegistry{}
		if err := reg.UnmarshalJSON(*p.FromImageRegistry); err != nil {
			return body, fmt.Errorf("unmarshal from_image_registry: %w", err)
		}
		body.FromImageRegistry = &reg
	}
	return body, nil
}

// ListTemplatesParams 列出模板的查询参数。
type ListTemplatesParams struct {
	// TeamID 团队 ID。
	TeamID *string
}

func (p *ListTemplatesParams) toAPI() *apis.ListTemplatesParams {
	if p == nil {
		return nil
	}
	return &apis.ListTemplatesParams{
		TeamID: p.TeamID,
	}
}

// GetTemplateParams 获取模板详情的查询参数。
type GetTemplateParams struct {
	// NextToken 分页游标。
	NextToken *string

	// Limit 每页最大返回数。
	Limit *int32
}

func (p *GetTemplateParams) toAPI() *apis.GetTemplateParams {
	if p == nil {
		return nil
	}
	return &apis.GetTemplateParams{
		NextToken: p.NextToken,
		Limit:     p.Limit,
	}
}

// GetBuildStatusParams 获取构建状态的查询参数。
type GetBuildStatusParams struct {
	// LogsOffset 起始构建日志的索引。
	LogsOffset *int32

	// Limit 返回的最大日志条数。
	Limit *int32

	// Level 日志级别过滤。
	Level *LogLevel
}

func (p *GetBuildStatusParams) toAPI() *apis.GetTemplateBuildStatusParams {
	if p == nil {
		return nil
	}
	params := &apis.GetTemplateBuildStatusParams{
		LogsOffset: p.LogsOffset,
		Limit:      p.Limit,
	}
	if p.Level != nil {
		level := apis.LogLevel(*p.Level)
		params.Level = &level
	}
	return params
}

// GetBuildLogsParams 获取构建日志的查询参数。
type GetBuildLogsParams struct {
	// Cursor 起始时间戳（毫秒）。
	Cursor *int64

	// Limit 返回的最大日志条数。
	Limit *int32

	// Direction 日志方向。
	Direction *LogsDirection

	// Level 日志级别过滤。
	Level *LogLevel

	// Source 日志来源过滤。
	Source *LogsSource
}

func (p *GetBuildLogsParams) toAPI() *apis.GetTemplateBuildLogsParams {
	if p == nil {
		return nil
	}
	params := &apis.GetTemplateBuildLogsParams{
		Cursor: p.Cursor,
		Limit:  p.Limit,
	}
	if p.Direction != nil {
		dir := apis.LogsDirection(*p.Direction)
		params.Direction = &dir
	}
	if p.Level != nil {
		level := apis.LogLevel(*p.Level)
		params.Level = &level
	}
	if p.Source != nil {
		src := apis.LogsSource(*p.Source)
		params.Source = &src
	}
	return params
}

// ManageTagsParams 管理模板标签的请求参数。
type ManageTagsParams struct {
	// Tags 要分配给模板的标签列表。
	Tags []string

	// Target 目标模板（格式："name:tag"）。
	Target string
}

func (p *ManageTagsParams) toAPI() apis.AssignTemplateTagsJSONRequestBody {
	return apis.AssignTemplateTagsJSONRequestBody{
		Tags:   p.Tags,
		Target: p.Target,
	}
}

// DeleteTagsParams 删除模板标签的请求参数。
type DeleteTagsParams struct {
	// Name 模板名称。
	Name string

	// Tags 要删除的标签列表。
	Tags []string
}

func (p *DeleteTagsParams) toAPI() apis.DeleteTemplateTagsJSONRequestBody {
	return apis.DeleteTemplateTagsJSONRequestBody{
		Name: p.Name,
		Tags: p.Tags,
	}
}

// Template 模板信息。
type Template struct {
	TemplateID    string
	Aliases       []string
	BuildID       string
	BuildStatus   TemplateBuildStatus
	BuildCount    int32
	CPUCount      int32
	MemoryMB      int32
	DiskSizeMB    int32
	EnvdVersion   string
	Public        bool
	SpawnCount    int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	LastSpawnedAt *time.Time
}

// TemplateBuild 模板构建记录。
type TemplateBuild struct {
	BuildID     string
	Status      TemplateBuildStatus
	CPUCount    int32
	MemoryMB    int32
	DiskSizeMB  *int32
	EnvdVersion *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	FinishedAt  *time.Time
}

// TemplateWithBuilds 模板及其构建记录。
type TemplateWithBuilds struct {
	TemplateID    string
	Aliases       []string
	Public        bool
	SpawnCount    int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	LastSpawnedAt *time.Time
	Builds        []TemplateBuild
}

// TemplateBuildInfo 模板构建状态信息。
type TemplateBuildInfo struct {
	TemplateID string
	BuildID    string
	Status     TemplateBuildStatus
	Logs       []string
}

// TemplateBuildLogs 模板构建日志。
type TemplateBuildLogs struct {
	Logs []BuildLogEntry
}

// BuildLogEntry 构建日志条目。
type BuildLogEntry struct {
	Level     LogLevel
	Message   string
	Step      *string
	Timestamp time.Time
}

// TemplateCreateResponse 创建模板的响应。
type TemplateCreateResponse struct {
	TemplateID string
	BuildID    string
	Aliases    []string
	Names      []string
	Tags       []string
	Public     bool
}

// TemplateBuildFileUpload 模板构建文件上传信息。
type TemplateBuildFileUpload struct {
	// Present 文件是否已存在。
	Present bool
	// URL 上传地址。
	URL *string
}

// TemplateAliasResponse 模板别名查询响应。
type TemplateAliasResponse struct {
	TemplateID string
	Public     bool
}

// AssignedTemplateTags 分配的模板标签。
type AssignedTemplateTags struct {
	BuildID string
	Tags    []string
}

// ---------------------------------------------------------------------------
// 转换函数 — apis → SDK
// ---------------------------------------------------------------------------

func templateFromAPI(a apis.Template) Template {
	return Template{
		TemplateID:    a.TemplateID,
		Aliases:       a.Aliases,
		BuildID:       a.BuildID,
		BuildStatus:   TemplateBuildStatus(a.BuildStatus),
		BuildCount:    a.BuildCount,
		CPUCount:      a.CPUCount,
		MemoryMB:      a.MemoryMB,
		DiskSizeMB:    a.DiskSizeMB,
		EnvdVersion:   a.EnvdVersion,
		Public:        a.Public,
		SpawnCount:    a.SpawnCount,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
		LastSpawnedAt: a.LastSpawnedAt,
	}
}

func templatesFromAPI(a []apis.Template) []Template {
	if a == nil {
		return nil
	}
	result := make([]Template, len(a))
	for i, t := range a {
		result[i] = templateFromAPI(t)
	}
	return result
}

func templateBuildFromAPI(a apis.TemplateBuild) TemplateBuild {
	return TemplateBuild{
		BuildID:     a.BuildID.String(),
		Status:      TemplateBuildStatus(a.Status),
		CPUCount:    a.CPUCount,
		MemoryMB:    a.MemoryMB,
		DiskSizeMB:  a.DiskSizeMB,
		EnvdVersion: a.EnvdVersion,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
		FinishedAt:  a.FinishedAt,
	}
}

func templateWithBuildsFromAPI(a *apis.TemplateWithBuilds) *TemplateWithBuilds {
	if a == nil {
		return nil
	}
	result := &TemplateWithBuilds{
		TemplateID:    a.TemplateID,
		Aliases:       a.Aliases,
		Public:        a.Public,
		SpawnCount:    a.SpawnCount,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
		LastSpawnedAt: a.LastSpawnedAt,
		Builds:        make([]TemplateBuild, 0, len(a.Builds)),
	}
	for _, b := range a.Builds {
		result.Builds = append(result.Builds, templateBuildFromAPI(b))
	}
	return result
}

func templateBuildInfoFromAPI(a *apis.TemplateBuildInfo) *TemplateBuildInfo {
	if a == nil {
		return nil
	}
	return &TemplateBuildInfo{
		TemplateID: a.TemplateID,
		BuildID:    a.BuildID,
		Status:     TemplateBuildStatus(a.Status),
		Logs:       a.Logs,
	}
}

func templateBuildLogsFromAPI(a *apis.TemplateBuildLogsResponse) *TemplateBuildLogs {
	if a == nil {
		return nil
	}
	result := &TemplateBuildLogs{Logs: make([]BuildLogEntry, 0, len(a.Logs))}
	for _, e := range a.Logs {
		result.Logs = append(result.Logs, BuildLogEntry{
			Level:     LogLevel(e.Level),
			Message:   e.Message,
			Step:      e.Step,
			Timestamp: e.Timestamp,
		})
	}
	return result
}

func templateCreateResponseFromAPI(a *apis.TemplateRequestResponseV3) *TemplateCreateResponse {
	if a == nil {
		return nil
	}
	return &TemplateCreateResponse{
		TemplateID: a.TemplateID,
		BuildID:    a.BuildID,
		Aliases:    a.Aliases,
		Names:      a.Names,
		Tags:       a.Tags,
		Public:     a.Public,
	}
}

func templateBuildFileUploadFromAPI(a *apis.TemplateBuildFileUpload) *TemplateBuildFileUpload {
	if a == nil {
		return nil
	}
	return &TemplateBuildFileUpload{
		Present: a.Present,
		URL:     a.URL,
	}
}

func templateAliasResponseFromAPI(a *apis.TemplateAliasResponse) *TemplateAliasResponse {
	if a == nil {
		return nil
	}
	return &TemplateAliasResponse{
		TemplateID: a.TemplateID,
		Public:     a.Public,
	}
}

func assignedTemplateTagsFromAPI(a *apis.AssignedTemplateTags) *AssignedTemplateTags {
	if a == nil {
		return nil
	}
	return &AssignedTemplateTags{
		BuildID: a.BuildID.String(),
		Tags:    a.Tags,
	}
}
