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

// KodoResource Kodo 存储桶资源，沙箱启动前由平台通过 NFS 代理挂载到指定路径。
// 使用 Kodo 资源创建沙箱时，客户端必须可用 Qiniu AK/SK 凭证。
type KodoResource struct {
	// Bucket Kodo 存储桶名称（必填）。
	Bucket string

	// MountPath 存储桶内容在沙箱内的绝对挂载路径（必填）。
	MountPath string

	// Prefix 存储桶内可选的对象名前缀；不设置时挂载整个存储桶根目录。
	Prefix *string

	// ReadOnly 是否以只读方式挂载；当 AK/SK 缺少写权限时服务端也会自动只读。
	ReadOnly *bool
}

// SandboxResourceSpec 沙箱资源规约（discriminated union），各字段互斥，只能设置一个。
type SandboxResourceSpec struct {
	// GitRepository GitHub 仓库资源。
	GitRepository *GitRepositoryResource

	// Kodo Kodo 存储桶资源。
	Kodo *KodoResource
}

// ---------------------------------------------------------------------------
// 转换函数 — SDK → apis
// ---------------------------------------------------------------------------

func sandboxResourceSpecToAPI(spec SandboxResourceSpec) (apis.SandboxResource, error) {
	var r apis.SandboxResource
	count := 0
	if spec.GitRepository != nil {
		count++
	}
	if spec.Kodo != nil {
		count++
	}
	if count == 0 {
		return r, fmt.Errorf("SandboxResourceSpec: exactly one resource type must be set (GitRepository or Kodo), got none")
	}
	if count > 1 {
		return r, fmt.Errorf("SandboxResourceSpec: exactly one resource type must be set, but got %d", count)
	}

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
	case spec.Kodo != nil:
		if spec.Kodo.Bucket == "" {
			return r, fmt.Errorf("KodoResource.Bucket must be set")
		}
		if spec.Kodo.MountPath == "" {
			return r, fmt.Errorf("KodoResource.MountPath must be set")
		}
		if err := r.FromKodoResource(apis.KodoResource{
			Bucket:    spec.Kodo.Bucket,
			MountPath: spec.Kodo.MountPath,
			Prefix:    spec.Kodo.Prefix,
			ReadOnly:  spec.Kodo.ReadOnly,
		}); err != nil {
			return r, err
		}
	default:
		panic("unreachable")
	}
	return r, nil
}
