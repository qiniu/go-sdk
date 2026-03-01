package sandbox

import (
	"time"

	"github.com/qiniu/go-sdk/v7/sandbox/apis"
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

// CreateTemplateParams 创建模板的请求参数。
type CreateTemplateParams = apis.CreateTemplateV3JSONRequestBody

// UpdateTemplateParams 更新模板的请求参数。
type UpdateTemplateParams = apis.UpdateTemplateJSONRequestBody

// StartTemplateBuildParams 启动模板构建的请求参数。
type StartTemplateBuildParams = apis.StartTemplateBuildV2JSONRequestBody

// ListTemplatesParams 列出模板的查询参数。
type ListTemplatesParams = apis.ListTemplatesParams

// GetTemplateParams 获取模板详情的查询参数。
type GetTemplateParams = apis.GetTemplateParams

// GetBuildStatusParams 获取构建状态的查询参数。
type GetBuildStatusParams = apis.GetTemplateBuildStatusParams

// GetBuildLogsParams 获取构建日志的查询参数。
type GetBuildLogsParams = apis.GetTemplateBuildLogsParams

// ManageTagsParams 管理模板标签的请求参数。
type ManageTagsParams = apis.ManageTemplateTagsJSONRequestBody

// DeleteTagsParams 删除模板标签的请求参数。
type DeleteTagsParams = apis.DeleteTemplateTagsJSONRequestBody

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
	Level     string
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
	result := &TemplateBuildLogs{}
	for _, e := range a.Logs {
		result.Logs = append(result.Logs, BuildLogEntry{
			Level:     string(e.Level),
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
