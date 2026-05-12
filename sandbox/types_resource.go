package sandbox

import (
	"fmt"

	"github.com/qiniu/go-sdk/v7/sandbox/internal/apis"
)

// ---------------------------------------------------------------------------
// SDK 自有类型 — 沙箱资源
// ---------------------------------------------------------------------------

// GitRepositoryType Git 仓库托管平台类型。
type GitRepositoryType string

// Git 仓库托管平台类型常量。后续若服务端扩展更多类型，将在此新增对应常量。
const (
	// GitRepositoryTypeGithub GitHub 仓库。
	GitRepositoryTypeGithub GitRepositoryType = "github_repository"
)

// GitRepositoryResource Git 仓库资源，沙箱启动前由平台拉取并挂载快照到指定路径。
// 同一沙箱内多个仓库资源当前必须共用同一 token。
type GitRepositoryResource struct {
	// Type 仓库托管平台类型（必填）。当前仅支持 GitRepositoryTypeGithub。
	Type GitRepositoryType

	// URL 仓库 URL（HTTPS 或 SSH 形式），如
	// https://github.com/owner/repo.git 或 git@github.com:owner/repo.git。
	URL string

	// MountPath 仓库内容在沙箱内的绝对挂载路径。
	MountPath string

	// AuthorizationToken 用于克隆该仓库的访问 token。
	// 同一沙箱内多个仓库资源当前必须共用同一 token。
	AuthorizationToken *string
}

// SandboxResourceSpec 沙箱资源规约（discriminated union），各字段互斥，只能设置一个。
type SandboxResourceSpec struct {
	// GitRepository GitHub 仓库资源。
	GitRepository *GitRepositoryResource
}

// ---------------------------------------------------------------------------
// 转换函数 — SDK → apis
// ---------------------------------------------------------------------------

func sandboxResourceSpecToAPI(spec SandboxResourceSpec) (apis.SandboxResource, error) {
	var r apis.SandboxResource
	switch {
	case spec.GitRepository != nil:
		if spec.GitRepository.Type == "" {
			return r, fmt.Errorf("GitRepositoryResource.Type must be set (e.g. GitRepositoryTypeGithub)")
		}
		if spec.GitRepository.URL == "" {
			return r, fmt.Errorf("GitRepositoryResource.URL must be set")
		}
		if spec.GitRepository.MountPath == "" {
			return r, fmt.Errorf("GitRepositoryResource.MountPath must be set")
		}
		if spec.GitRepository.AuthorizationToken == nil || *spec.GitRepository.AuthorizationToken == "" {
			return r, fmt.Errorf("GitRepositoryResource.AuthorizationToken must be set")
		}
		if err := r.FromGitRepositoryResource(apis.GitRepositoryResource{
			URL:                spec.GitRepository.URL,
			MountPath:          spec.GitRepository.MountPath,
			AuthorizationToken: spec.GitRepository.AuthorizationToken,
			Type:               apis.GitRepositoryResourceType(spec.GitRepository.Type),
		}); err != nil {
			return r, err
		}
	default:
		return r, fmt.Errorf("SandboxResourceSpec: exactly one resource type must be set (GitRepository), got none")
	}
	return r, nil
}
