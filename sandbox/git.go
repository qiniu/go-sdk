// Package sandbox 中的 git.go 提供沙箱内 git 操作的高层封装。
//
// Git 操作通过 Commands.Run 调用沙箱内已预装的 git 二进制实现，
// 仅支持 HTTPS + username/password (token) 形式的认证，不支持 SSH key。

package sandbox

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"
)

// defaultGitEnv 是所有 git 命令默认注入的环境变量，用于禁止交互式提示。
var defaultGitEnv = map[string]string{
	"GIT_TERMINAL_PROMPT": "0",
}

// Git 提供沙箱内的 git 操作接口。
//
// 仅支持 HTTPS + username/password (token) 认证，不支持 SSH key。
// 所有命令都会强制注入 GIT_TERMINAL_PROMPT=0 以禁止交互式输入。
type Git struct {
	commands *Commands
}

// newGit 创建 Git 实例。
func newGit(c *Commands) *Git {
	return &Git{commands: c}
}

// Clone 克隆远程仓库到沙箱中。
// 当 opts.Username/Password 提供且 opts.DangerouslyStoreCredentials 为 false 时，
// SDK 会在 clone 完成后通过 git remote set-url 移除 origin URL 中的凭证。
func (g *Git) Clone(ctx context.Context, repoURL string, opts *CloneOptions) (*CommandResult, error) {
	if opts == nil {
		opts = &CloneOptions{}
	}
	if opts.Password != "" && opts.Username == "" {
		return nil, &InvalidArgumentError{Msg: "Username is required when using a password or token for git clone."}
	}

	plan, err := buildClonePlan(repoURL, opts.Path, opts.Branch, opts.Depth, opts.Username, opts.Password, opts.DangerouslyStoreCredentials)
	if err != nil {
		return nil, err
	}

	result, err := g.runGit(ctx, "clone", plan.args, "", &opts.GitOptions)
	if err != nil {
		if isAuthFailure(err) {
			return nil, &GitAuthError{Msg: buildAuthErrorMessage("clone", opts.Username != "" && opts.Password == "")}
		}
		return nil, err
	}

	if plan.shouldStrip {
		// 即使 ctx 已取消也要清理凭证，避免带凭证的 URL 残留在 .git/config。
		if _, serr := g.runGit(context.Background(), "remote", buildRemoteSetURLArgs("origin", plan.sanitizedURL), plan.repoPath, &opts.GitOptions); serr != nil {
			return result, fmt.Errorf("clone succeeded but failed to strip credentials: %w", serr)
		}
	}
	return result, nil
}

// Init 在指定路径初始化新的 git 仓库。
func (g *Git) Init(ctx context.Context, path string, opts *InitOptions) (*CommandResult, error) {
	if opts == nil {
		opts = &InitOptions{}
	}
	args := []string{"init"}
	if opts.InitialBranch != "" {
		args = append(args, "--initial-branch", opts.InitialBranch)
	}
	if opts.Bare {
		args = append(args, "--bare")
	}
	args = append(args, path)
	return g.runGit(ctx, "init", args, "", &opts.GitOptions)
}

// Status 返回仓库状态信息。
func (g *Git) Status(ctx context.Context, path string, opts *GitOptions) (*GitStatus, error) {
	result, err := g.runGit(ctx, "status", buildStatusArgs(), path, opts)
	if err != nil {
		return nil, err
	}
	return parseGitStatus(result.Stdout), nil
}

// Branches 返回仓库的分支列表。
//
// unborn 仓库（git init 后尚未提交）下 `git branch` 无任何输出，但 HEAD 已经
// 指向初始分支；此时退化到 `git symbolic-ref --short HEAD` 取出 CurrentBranch，
// 与 GitBranches.CurrentBranch "仅在 detached HEAD 时为空" 的契约保持一致。
func (g *Git) Branches(ctx context.Context, path string, opts *GitOptions) (*GitBranches, error) {
	result, err := g.runGit(ctx, "branch", buildBranchesArgs(), path, opts)
	if err != nil {
		return nil, err
	}
	branches := parseGitBranches(result.Stdout)
	if branches.CurrentBranch == "" && len(branches.Branches) == 0 {
		if name, ok := g.unbornCurrentBranch(ctx, path, opts); ok {
			branches.CurrentBranch = name
		}
	}
	return branches, nil
}

// unbornCurrentBranch 尝试通过 symbolic-ref 读取 unborn 仓库的当前分支名。
// 仓库为 detached HEAD 或非 git 目录时 symbolic-ref 失败，返回 ok=false。
func (g *Git) unbornCurrentBranch(ctx context.Context, path string, opts *GitOptions) (string, bool) {
	result, err := g.runGit(ctx, "symbolic-ref", []string{"symbolic-ref", "--short", "HEAD"}, path, opts)
	if err != nil {
		return "", false
	}
	name := strings.TrimSpace(result.Stdout)
	if name == "" {
		return "", false
	}
	return name, true
}

// CreateBranch 创建并切换到新分支。
func (g *Git) CreateBranch(ctx context.Context, path, branch string, opts *GitOptions) (*CommandResult, error) {
	if branch == "" {
		return nil, &InvalidArgumentError{Msg: "Branch name is required."}
	}
	return g.runGit(ctx, "checkout", buildCreateBranchArgs(branch), path, opts)
}

// CheckoutBranch 切换到已存在的分支。
func (g *Git) CheckoutBranch(ctx context.Context, path, branch string, opts *GitOptions) (*CommandResult, error) {
	if branch == "" {
		return nil, &InvalidArgumentError{Msg: "Branch name is required."}
	}
	return g.runGit(ctx, "checkout", buildCheckoutBranchArgs(branch), path, opts)
}

// DeleteBranch 删除分支。
func (g *Git) DeleteBranch(ctx context.Context, path, branch string, opts *DeleteBranchOptions) (*CommandResult, error) {
	if branch == "" {
		return nil, &InvalidArgumentError{Msg: "Branch name is required."}
	}
	if opts == nil {
		opts = &DeleteBranchOptions{}
	}
	return g.runGit(ctx, "branch", buildDeleteBranchArgs(branch, opts.Force), path, &opts.GitOptions)
}

// Add 暂存指定文件，未指定文件时默认 stage 全部变更。
func (g *Git) Add(ctx context.Context, path string, opts *AddOptions) (*CommandResult, error) {
	if opts == nil {
		opts = &AddOptions{}
	}
	all := true
	if opts.All != nil {
		all = *opts.All
	}
	return g.runGit(ctx, "add", buildAddArgs(opts.Files, all), path, &opts.GitOptions)
}

// Commit 在仓库中创建一次提交。
func (g *Git) Commit(ctx context.Context, path, message string, opts *CommitOptions) (*CommandResult, error) {
	if message == "" {
		return nil, &InvalidArgumentError{Msg: "Commit message is required."}
	}
	if opts == nil {
		opts = &CommitOptions{}
	}
	return g.runGit(ctx, "commit", buildCommitArgs(message, opts.AuthorName, opts.AuthorEmail, opts.AllowEmpty), path, &opts.GitOptions)
}

// Reset 重置 HEAD 到指定状态。
func (g *Git) Reset(ctx context.Context, path string, opts *ResetOptions) (*CommandResult, error) {
	if opts == nil {
		opts = &ResetOptions{}
	}
	args, err := buildResetArgs(opts.Mode, opts.Target, opts.Paths)
	if err != nil {
		return nil, err
	}
	return g.runGit(ctx, "reset", args, path, &opts.GitOptions)
}

// Restore 恢复工作区文件或取消暂存。
func (g *Git) Restore(ctx context.Context, path string, opts *RestoreOptions) (*CommandResult, error) {
	if opts == nil {
		return nil, &InvalidArgumentError{Msg: "Restore options are required."}
	}
	args, err := buildRestoreArgs(opts.Paths, opts.Staged, opts.Worktree, opts.Source)
	if err != nil {
		return nil, err
	}
	return g.runGit(ctx, "restore", args, path, &opts.GitOptions)
}

// Push 将提交推送到远程。
// 当 opts.Username/Password 提供时，SDK 会临时把凭证写入目标 remote URL，命令完成后立即恢复原 URL。
//
// 当未显式指定 Remote 时，SDK 会尝试自动选中仓库唯一的 remote（与"自动选择单一 remote"
// 契约一致）；多 remote / 无 remote 时回退到 git 原生行为或返回带凭证场景下的明确错误。
//
// 当 SetUpstream=true 且未显式给 Branch 时，仅在 SDK 已确定 target remote 的前提下，
// 才会通过 git rev-parse 取出当前分支名再拼到 push 命令上（避免 `git push --set-upstream <remote>`
// 不带分支名被 git 拒绝）。target 未确定（多 remote / 无 remote）时保持原始 `git push` 形态，
// 让 git 自行报错或走默认语义，避免把分支名误当成 remote 名。
func (g *Git) Push(ctx context.Context, path string, opts *PushOptions) (*CommandResult, error) {
	if opts == nil {
		opts = &PushOptions{}
	}
	if opts.Password != "" && opts.Username == "" {
		return nil, &InvalidArgumentError{Msg: "Username is required when using a password or token for git push."}
	}
	setUpstream := true
	if opts.SetUpstream != nil {
		setUpstream = *opts.SetUpstream
	}

	// branch 补全延迟到 target 已经解析出来时再做：
	// 1) target 为空（多 remote / 无 remote）但调用方显式给了 Branch 时，直接返回错误：
	//    `git push <branch>` 在原生 git 里会被解析成 `git push <repository>`，把分支名当 remote 名，
	//    与 PushOptions.Branch 的对外语义不一致。要求调用方显式补 Remote。
	// 2) target 为空且 Branch 也未指定时保持原始 `git push` 形态，让 git 走默认语义。
	// 3) target 已确定且 setUpstream=true、Branch 未指定时，用 rev-parse 取当前分支名补上，
	//    避免 `git push --set-upstream <remote>` 不带分支名被 git 拒绝。
	buildArgs := func(target string) ([]string, error) {
		branch := opts.Branch
		if target == "" {
			if branch != "" {
				return nil, &InvalidArgumentError{Msg: "Remote is required when Branch is specified and the repository does not have a single remote to auto-select."}
			}
			return buildPushArgs("", "", false), nil
		}
		if setUpstream && branch == "" {
			name, err := g.currentBranch(ctx, path, &opts.GitOptions)
			if err != nil {
				return nil, err
			}
			branch = name
		}
		return buildPushArgs(target, branch, setUpstream), nil
	}
	return g.runWithOptionalCredentials(ctx, "push", path, opts.Remote, opts.Username, opts.Password, &opts.GitOptions, buildArgs)
}

// Pull 从远程拉取变更。
// 当 opts.Username/Password 提供时，SDK 会临时把凭证写入目标 remote URL，命令完成后立即恢复原 URL。
//
// 当未显式指定 Remote 时，SDK 会尝试自动选中仓库唯一的 remote（与"自动选择单一 remote"
// 契约一致）。当 Remote 与 Branch 都未指定时，SDK 会先检查当前分支是否配置了 upstream，
// 未配置则直接返回 GitUpstreamError 而不是把模糊错误抛给调用方。
func (g *Git) Pull(ctx context.Context, path string, opts *PullOptions) (*CommandResult, error) {
	if opts == nil {
		opts = &PullOptions{}
	}
	if opts.Password != "" && opts.Username == "" {
		return nil, &InvalidArgumentError{Msg: "Username is required when using a password or token for git pull."}
	}

	if opts.Remote == "" && opts.Branch == "" {
		ok, err := g.hasUpstream(ctx, path, &opts.GitOptions)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, &GitUpstreamError{Msg: buildUpstreamErrorMessage("pull")}
		}
	}
	return g.runWithOptionalCredentials(ctx, "pull", path, opts.Remote, opts.Username, opts.Password, &opts.GitOptions,
		func(target string) ([]string, error) {
			// target 为空但显式给了 Branch 时直接返回错误：
			// `git pull <branch>` 在 git 里会被解析成 `git pull <repository>`，与对外语义不一致。
			if target == "" && opts.Branch != "" {
				return nil, &InvalidArgumentError{Msg: "Remote is required when Branch is specified and the repository does not have a single remote to auto-select."}
			}
			return buildPullArgs(target, opts.Branch), nil
		})
}

// currentBranch 通过 git rev-parse 取出当前 HEAD 指向的分支名。
// detached HEAD 时 rev-parse 返回 "HEAD"，会作为错误向上抛出，避免错误地把字面 "HEAD" 拼到 push 命令上。
func (g *Git) currentBranch(ctx context.Context, path string, opts *GitOptions) (string, error) {
	result, err := g.runGit(ctx, "rev-parse", []string{"rev-parse", "--abbrev-ref", "HEAD"}, path, opts)
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(result.Stdout)
	if name == "" || name == "HEAD" {
		return "", &InvalidArgumentError{Msg: "Cannot push with SetUpstream=true on a detached HEAD; specify Branch explicitly or set SetUpstream=false."}
	}
	return name, nil
}

// runWithOptionalCredentials 执行 push/pull 等需要可选凭证注入的远程同步命令。
// buildArgs 接收已解析的 target remote 名（可为空），返回最终参数列表。
//
// remote 解析提到凭证分支之外，确保"带凭证"与"不带凭证"两种调用对单 remote 仓库
// 有一致的语义：当调用方未显式指定 remote 时，仓库恰好只有一个 remote 会被自动选中。
func (g *Git) runWithOptionalCredentials(
	ctx context.Context,
	sub, path, remote, username, password string,
	opts *GitOptions,
	buildArgs func(target string) ([]string, error),
) (*CommandResult, error) {
	withCreds := username != "" && password != ""

	// 不带凭证时，对显式 remote 直接使用；未指定 remote 时尝试自动选中唯一 remote，
	// 若仓库无 remote 或多 remote 则交回 git 自身的默认行为（保持向后兼容，让 git 报错）。
	if !withCreds {
		target := remote
		if target == "" {
			name, err := g.autoSelectRemote(ctx, path, opts)
			if err != nil {
				// `git remote` 自身失败（路径非仓库等）时直接返回原始错误，
				// 避免被后续 buildArgs 转成误导性的 InvalidArgumentError。
				return nil, err
			}
			target = name
		}
		args, err := buildArgs(target)
		if err != nil {
			return nil, err
		}
		result, err := g.runGit(ctx, sub, args, path, opts)
		if err != nil {
			return nil, mapPushPullError(err, sub, username, password)
		}
		return result, nil
	}

	// 带凭证时必须解析出确切的 remote 名（含 URL），用于注入/恢复凭证。
	remoteName, originalURL, err := g.resolveRemoteName(ctx, path, remote, opts)
	if err != nil {
		return nil, err
	}
	args, err := buildArgs(remoteName)
	if err != nil {
		return nil, err
	}
	var result *CommandResult
	err = g.withRemoteCredentials(ctx, path, remoteName, originalURL, username, password, opts, func() error {
		r, runErr := g.runGit(ctx, sub, args, path, opts)
		result = r
		return runErr
	})
	if err != nil {
		return nil, mapPushPullError(err, sub, username, password)
	}
	return result, nil
}

// autoSelectRemote 在仓库恰好有一个 remote 时返回 (name, nil)；
// 仓库为 0 / 多 remote 时返回 ("", nil)，由调用方决定是否兜底到 git 默认行为；
// `git remote` 自身执行失败（路径不是仓库 / 仓库不可访问 / git 异常）时返回 ("", err)，
// 避免把真实仓库错误掩盖成"需要显式传 Remote"。
func (g *Git) autoSelectRemote(ctx context.Context, path string, opts *GitOptions) (string, error) {
	result, err := g.runGit(ctx, "remote", []string{"remote"}, path, opts)
	if err != nil {
		return "", err
	}
	var remotes []string
	for _, line := range strings.Split(result.Stdout, "\n") {
		if s := strings.TrimSpace(line); s != "" {
			remotes = append(remotes, s)
		}
	}
	if len(remotes) == 1 {
		return remotes[0], nil
	}
	return "", nil
}

// RemoteAdd 为仓库添加（或在 opts.Overwrite=true 时覆盖）一个 remote。
func (g *Git) RemoteAdd(ctx context.Context, path, name, repoURL string, opts *RemoteAddOptions) (*CommandResult, error) {
	if opts == nil {
		opts = &RemoteAddOptions{}
	}
	addArgs, err := buildRemoteAddArgs(name, repoURL)
	if err != nil {
		return nil, err
	}

	// 构造 add-or-overwrite 阶段：不带 fetch，避免 add 失败后 set-url fallback 与 add -f 的语义混淆。
	var addPhase string
	if opts.Overwrite {
		addCmd := buildGitCommand(addArgs, path)
		setURLCmd := buildGitCommand(buildRemoteSetURLArgs(name, repoURL), path)
		addPhase = addCmd + " || " + setURLCmd
	} else {
		addPhase = buildGitCommand(addArgs, path)
	}

	// 不需要 fetch 时直接执行 add 阶段。
	if !opts.Fetch {
		if !opts.Overwrite {
			return g.runGit(ctx, "remote", addArgs, path, &opts.GitOptions)
		}
		return g.runShell(ctx, "remote", addPhase, &opts.GitOptions)
	}

	// 需要 fetch 时统一拼一次：add 阶段成功后仅 fetch 一次，避免重复网络请求。
	fetchCmd := buildGitCommand([]string{"fetch", name}, path)
	return g.runShell(ctx, "remote", "("+addPhase+") && "+fetchCmd, &opts.GitOptions)
}

// RemoteGet 获取指定 remote 的 URL，未配置时返回空字符串。
func (g *Git) RemoteGet(ctx context.Context, path, name string, opts *GitOptions) (string, error) {
	if name == "" {
		return "", &InvalidArgumentError{Msg: "Remote name is required."}
	}
	result, err := g.runGit(ctx, "remote", buildRemoteGetURLArgs(name), path, opts)
	if err != nil {
		// `git remote get-url <name>` 在 remote 不存在时退出码为 2；其他退出码视为真正的错误。
		var ce *gitCommandError
		if errors.As(err, &ce) && ce.Result != nil && ce.Result.ExitCode == 2 {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

// SetConfig 设置 git config 值。
// 使用 GitConfigScopeLocal 时必须同时提供 opts.Path。
func (g *Git) SetConfig(ctx context.Context, key, value string, opts *ConfigOptions) (*CommandResult, error) {
	if key == "" {
		return nil, &InvalidArgumentError{Msg: "Git config key is required."}
	}
	if opts == nil {
		opts = &ConfigOptions{}
	}
	scope, repoPath, err := resolveConfigScope(opts.Scope, opts.Path)
	if err != nil {
		return nil, err
	}
	return g.runGit(ctx, "config", []string{"config", scope, key, value}, repoPath, &opts.GitOptions)
}

// GetConfig 获取 git config 值，未配置时返回空字符串。
func (g *Git) GetConfig(ctx context.Context, key string, opts *ConfigOptions) (string, error) {
	if key == "" {
		return "", &InvalidArgumentError{Msg: "Git config key is required."}
	}
	if opts == nil {
		opts = &ConfigOptions{}
	}
	scope, repoPath, err := resolveConfigScope(opts.Scope, opts.Path)
	if err != nil {
		return "", err
	}
	cmd := buildGitCommand([]string{"config", scope, "--get", key}, repoPath)
	result, err := g.runShell(ctx, "config", cmd, &opts.GitOptions)
	if err != nil {
		// `git config --get` 未找到值时退出码为 1，stdout/stderr 为空；其他退出码视为真正的错误。
		var ce *gitCommandError
		if errors.As(err, &ce) && ce.Result != nil && ce.Result.ExitCode == 1 &&
			strings.TrimSpace(ce.Result.Stdout) == "" && strings.TrimSpace(ce.Result.Stderr) == "" {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

// ConfigureUser 设置 git 提交用户名与邮箱（一次 RPC 内完成）。
func (g *Git) ConfigureUser(ctx context.Context, name, email string, opts *ConfigOptions) (*CommandResult, error) {
	if name == "" || email == "" {
		return nil, &InvalidArgumentError{Msg: "Both name and email are required."}
	}
	if opts == nil {
		opts = &ConfigOptions{}
	}
	scope, repoPath, err := resolveConfigScope(opts.Scope, opts.Path)
	if err != nil {
		return nil, err
	}
	nameCmd := buildGitCommand([]string{"config", scope, "user.name", name}, repoPath)
	emailCmd := buildGitCommand([]string{"config", scope, "user.email", email}, repoPath)
	return g.runShell(ctx, "config", nameCmd+" && "+emailCmd, &opts.GitOptions)
}

// DangerouslyAuthenticate 通过 git credential helper 将凭证持久化到磁盘。
//
// 该方法会为后续所有 git 操作生效；建议优先使用短生命周期 token，并谨慎使用此方法。
func (g *Git) DangerouslyAuthenticate(ctx context.Context, opts *AuthenticateOptions) (*CommandResult, error) {
	if opts == nil || opts.Username == "" || opts.Password == "" {
		return nil, &InvalidArgumentError{Msg: "Both username and password are required to authenticate git."}
	}

	host := strings.TrimSpace(opts.Host)
	if host == "" {
		host = "github.com"
	}
	protocol := strings.TrimSpace(opts.Protocol)
	if protocol == "" {
		protocol = "https"
	}
	if protocol != "https" {
		return nil, &InvalidArgumentError{Msg: "Only https protocol is supported for git authentication."}
	}

	if _, err := g.runGit(ctx, "config", []string{"config", "--global", "credential.helper", "store"}, "", &opts.GitOptions); err != nil {
		return nil, err
	}

	credentialInput := strings.Join([]string{
		"protocol=" + protocol,
		"host=" + host,
		"username=" + opts.Username,
		"password=" + opts.Password,
		"",
		"",
	}, "\n")
	// 把格式串显式包成 '%s'，让 review 工具更容易判定 credentialInput 的 % 等字符
	// 不会被 printf 解析；shellEscape 已用单引号包裹，原本也不会触发任何二次解析。
	cmd := fmt.Sprintf("printf '%%s' %s | %s", shellEscape(credentialInput), buildGitCommand([]string{"credential", "approve"}, ""))
	return g.runShell(ctx, "credential", cmd, &opts.GitOptions)
}

// resolveConfigScope 校验并返回 git config 的 scope 标志与仓库路径。
func resolveConfigScope(scope GitConfigScope, repoPath string) (string, string, error) {
	if scope == "" {
		scope = GitConfigScopeGlobal
	}
	switch scope {
	case GitConfigScopeGlobal:
		return "--global", "", nil
	case GitConfigScopeSystem:
		return "--system", "", nil
	case GitConfigScopeLocal:
		if repoPath == "" {
			return "", "", &InvalidArgumentError{Msg: "Repository path is required for local git config scope."}
		}
		return "--local", repoPath, nil
	}
	return "", "", &InvalidArgumentError{Msg: "Unsupported git config scope: " + string(scope)}
}

// hasUpstream 检查当前分支是否配置了 upstream。
func (g *Git) hasUpstream(ctx context.Context, path string, opts *GitOptions) (bool, error) {
	_, err := g.runGit(ctx, "rev-parse", buildHasUpstreamArgs(), path, opts)
	if err == nil {
		return true, nil
	}
	// 未配置 upstream 时 rev-parse 退出码为 128，stderr 含 "no upstream" 等说明；
	// 其他失败（仓库路径错误、git 自身问题）应原样向上抛出。
	var ce *gitCommandError
	if errors.As(err, &ce) && ce.Result != nil && isMissingUpstream(err) {
		return false, nil
	}
	return false, err
}

// resolveRemoteName 在凭证注入流程中确定要使用的 remote 名，并一并返回其 URL。
//
// 解析规则与 E2B 对齐：
//   - 显式指定 remote 时直接使用。
//   - 未指定时通过 `git remote` 列出全部 remote：恰好一个时自动使用，多个时报错（要求显式指定）。
//
// 一并返回 URL 是为了让 withRemoteCredentials 复用，避免再发一次 RPC。
func (g *Git) resolveRemoteName(ctx context.Context, path, remote string, opts *GitOptions) (string, string, error) {
	name := remote
	if name == "" {
		result, err := g.runGit(ctx, "remote", []string{"remote"}, path, opts)
		if err != nil {
			return "", "", err
		}
		var remotes []string
		for _, line := range strings.Split(result.Stdout, "\n") {
			if s := strings.TrimSpace(line); s != "" {
				remotes = append(remotes, s)
			}
		}
		switch len(remotes) {
		case 0:
			return "", "", &InvalidArgumentError{Msg: "Repository has no remote configured."}
		case 1:
			name = remotes[0]
		default:
			return "", "", &InvalidArgumentError{Msg: "Remote is required when using username/password and the repository has multiple remotes."}
		}
	}

	url, err := g.RemoteGet(ctx, path, name, opts)
	if err != nil {
		return "", "", err
	}
	if url == "" {
		return "", "", &InvalidArgumentError{Msg: fmt.Sprintf("Remote %q is not configured.", name)}
	}
	return name, url, nil
}

// withRemoteCredentials 临时把 username/password 注入指定 remote 的 URL，
// 在 fn 执行完成后无论成功或失败都恢复原 URL。originalURL 由调用方传入以避免重复 RPC。
func (g *Git) withRemoteCredentials(ctx context.Context, path, remote, originalURL, username, password string, opts *GitOptions, fn func() error) (err error) {
	authedURL, err := withCredentials(originalURL, username, password)
	if err != nil {
		return err
	}
	if authedURL == originalURL {
		return fn()
	}

	if _, err = g.runGit(ctx, "remote", buildRemoteSetURLArgs(remote, authedURL), path, opts); err != nil {
		return err
	}
	defer func() {
		// 即使 ctx 已取消也要恢复原 URL，避免凭证遗留在 .git/config。
		// 恢复 URL 是安全相关步骤，主操作错误不应吞掉它；用 errors.Join 同时保留两者。
		_, restoreErr := g.runGit(context.Background(), "remote", buildRemoteSetURLArgs(remote, originalURL), path, opts)
		err = errors.Join(err, restoreErr)
	}()
	return fn()
}

// runGit 拼装并执行一条 git 命令。
func (g *Git) runGit(ctx context.Context, sub string, args []string, repoPath string, opts *GitOptions) (*CommandResult, error) {
	cmd := buildGitCommand(args, repoPath)
	return g.runShell(ctx, sub, cmd, opts)
}

// runShell 以 shell 形式执行命令，统一注入默认 git 环境变量。
func (g *Git) runShell(ctx context.Context, sub, cmd string, opts *GitOptions) (*CommandResult, error) {
	cmdOpts := buildCommandOptions(opts)
	result, err := g.commands.Run(ctx, cmd, cmdOpts...)
	if err != nil {
		return nil, fmt.Errorf("git %s: %w", sub, err)
	}
	if result.ExitCode != 0 {
		return result, &gitCommandError{Cmd: sub, Result: result}
	}
	return result, nil
}

// buildCommandOptions 将 GitOptions 转换为 CommandOption 列表。
func buildCommandOptions(opts *GitOptions) []CommandOption {
	capacity := len(defaultGitEnv)
	if opts != nil {
		capacity += len(opts.Envs)
	}
	mergedEnvs := make(map[string]string, capacity)
	cmdOpts := []CommandOption{}
	if opts != nil {
		maps.Copy(mergedEnvs, opts.Envs)
		if opts.User != "" {
			cmdOpts = append(cmdOpts, WithCommandUser(opts.User))
		}
		if opts.Cwd != "" {
			cmdOpts = append(cmdOpts, WithCwd(opts.Cwd))
		}
		if opts.Timeout > 0 {
			cmdOpts = append(cmdOpts, WithTimeout(opts.Timeout))
		}
	}
	// 默认环境必须最后写入，避免被调用方覆盖（例如禁用 GIT_TERMINAL_PROMPT）。
	maps.Copy(mergedEnvs, defaultGitEnv)
	cmdOpts = append(cmdOpts, WithEnvs(mergedEnvs))
	return cmdOpts
}

// mapPushPullError 将 push/pull 的命令错误映射为类型化 git 错误。
func mapPushPullError(err error, action, username, password string) error {
	if err == nil {
		return nil
	}
	if isAuthFailure(err) {
		return &GitAuthError{Msg: buildAuthErrorMessage(action, username != "" && password == "")}
	}
	if isMissingUpstream(err) {
		return &GitUpstreamError{Msg: buildUpstreamErrorMessage(action)}
	}
	return err
}
