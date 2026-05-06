package sandbox

import "time"

// GitOptions 是 git 命令执行的通用选项，可以被各 Options Struct 嵌入复用。
type GitOptions struct {
	// Envs 是额外的环境变量。SDK 会自动注入 GIT_TERMINAL_PROMPT=0 以禁止交互式提示。
	Envs map[string]string
	// User 指定执行命令的用户，缺省使用 DefaultUser。
	User string
	// Cwd 指定工作目录。
	Cwd string
	// Timeout 指定命令超时时间。
	Timeout time.Duration
}

// CloneOptions 是 git clone 操作的选项。
type CloneOptions struct {
	GitOptions
	// Path 指定克隆目标路径。
	Path string
	// Branch 指定要检出的分支，设置后会附加 --single-branch。
	Branch string
	// Depth 指定浅克隆深度。
	Depth int
	// Username 是 HTTPS 认证的用户名。
	Username string
	// Password 是 HTTPS 认证的密码或 token。
	Password string
	// DangerouslyStoreCredentials 为 true 时，凭证会持久化在 .git/config 中；
	// 默认会在 clone 完成后通过 remote set-url 清除。
	DangerouslyStoreCredentials bool
}

// InitOptions 是 git init 操作的选项。
type InitOptions struct {
	GitOptions
	// Bare 为 true 时创建 bare 仓库。
	Bare bool
	// InitialBranch 指定初始分支名（例如 "main"）。
	InitialBranch string
}

// RemoteAddOptions 是 git remote add 操作的选项。
type RemoteAddOptions struct {
	GitOptions
	// Fetch 为 true 时在添加 remote 后立即执行 fetch。
	Fetch bool
	// Overwrite 为 true 时，若 remote 已存在则覆盖其 URL。
	Overwrite bool
}

// CommitOptions 是 git commit 操作的选项。
type CommitOptions struct {
	GitOptions
	// AuthorName 覆盖提交作者名。
	AuthorName string
	// AuthorEmail 覆盖提交作者邮箱。
	AuthorEmail string
	// AllowEmpty 允许空提交。
	AllowEmpty bool
}

// AddOptions 是 git add 操作的选项。
type AddOptions struct {
	GitOptions
	// Files 指定要暂存的文件列表，为空时根据 All 决定使用 -A 或 "."。
	Files []string
	// All 控制 Files 为空时是否使用 -A 暂存所有变更。
	// 为 nil 时默认 true（与 E2B 对齐）；显式为 false 则使用 "." 仅暂存当前目录。
	All *bool
}

// DeleteBranchOptions 是 git branch -d/-D 操作的选项。
type DeleteBranchOptions struct {
	GitOptions
	// Force 为 true 时使用 -D 强制删除分支。
	Force bool
}

// GitResetMode 是 git reset 的模式。
type GitResetMode string

const (
	// GitResetModeSoft 对应 git reset --soft。
	GitResetModeSoft GitResetMode = "soft"
	// GitResetModeMixed 对应 git reset --mixed。
	GitResetModeMixed GitResetMode = "mixed"
	// GitResetModeHard 对应 git reset --hard。
	GitResetModeHard GitResetMode = "hard"
	// GitResetModeMerge 对应 git reset --merge。
	GitResetModeMerge GitResetMode = "merge"
	// GitResetModeKeep 对应 git reset --keep。
	GitResetModeKeep GitResetMode = "keep"
)

// ResetOptions 是 git reset 操作的选项。
type ResetOptions struct {
	GitOptions
	// Mode 是重置模式，缺省时不传 --<mode> 参数。
	Mode GitResetMode
	// Target 是要重置到的提交、分支或引用，缺省为 HEAD。
	Target string
	// Paths 是要重置的路径列表。
	Paths []string
}

// RestoreOptions 是 git restore 操作的选项。
// 当 Staged 与 Worktree 均未显式设置时，默认仅恢复工作区（Worktree=true）。
type RestoreOptions struct {
	GitOptions
	// Paths 是要恢复的路径列表，至少一个。
	Paths []string
	// Staged 为非 nil 时显式控制是否恢复索引。
	Staged *bool
	// Worktree 为非 nil 时显式控制是否恢复工作区。
	Worktree *bool
	// Source 指定从该提交或引用恢复。
	Source string
}

// PushOptions 是 git push 操作的选项。
type PushOptions struct {
	GitOptions
	// Remote 指定远程名（例如 "origin"）。
	Remote string
	// Branch 指定要推送的分支名。
	Branch string
	// SetUpstream 控制是否附加 --set-upstream。
	// 为 nil 时按 SDK 默认（true，与 E2B 对齐）；显式设置为 false 可关闭。
	SetUpstream *bool
	// Username 是 HTTPS 认证的用户名。
	Username string
	// Password 是 HTTPS 认证的密码或 token。
	Password string
}

// PullOptions 是 git pull 操作的选项。
type PullOptions struct {
	GitOptions
	// Remote 指定远程名。
	Remote string
	// Branch 指定要拉取的分支名。
	Branch string
	// Username 是 HTTPS 认证的用户名。
	Username string
	// Password 是 HTTPS 认证的密码或 token。
	Password string
}

// ConfigOptions 是 git config 操作的选项。
// Scope 为 "local" 时必须提供 Path。
type ConfigOptions struct {
	GitOptions
	// Scope 是 git config 作用域，缺省为 GitConfigScopeGlobal。
	Scope GitConfigScope
	// Path 是仓库路径，仅在 Scope 为 GitConfigScopeLocal 时必填。
	Path string
}

// AuthenticateOptions 是 DangerouslyAuthenticate 的选项。
type AuthenticateOptions struct {
	GitOptions
	// Username 是 HTTPS 认证的用户名，必填。
	Username string
	// Password 是 HTTPS 认证的密码或 token，必填。
	Password string
	// Host 是要认证的主机，缺省为 "github.com"。
	Host string
	// Protocol 是要认证的协议，缺省为 "https"。
	Protocol string
}
