package sandbox

import (
	"strconv"
	"strings"
)

// shellEscape 对字符串做 shell 单引号转义，使其可安全嵌入命令字符串。
func shellEscape(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

// buildGitCommand 拼装一条 shell 安全的 git 命令字符串。
// repoPath 非空时附加 -C <repoPath>。
func buildGitCommand(args []string, repoPath string) string {
	parts := make([]string, 0, len(args)+3)
	parts = append(parts, "git")
	if repoPath != "" {
		parts = append(parts, "-C", repoPath)
	}
	parts = append(parts, args...)
	for i, p := range parts {
		parts[i] = shellEscape(p)
	}
	return strings.Join(parts, " ")
}

// buildPushArgs 构造 git push 命令的参数列表。
// target 为最终使用的 remote 名（已合并 remoteName 与 remote 选项）。
func buildPushArgs(target, branch string, setUpstream bool) []string {
	args := []string{"push"}
	if setUpstream && target != "" {
		args = append(args, "--set-upstream")
	}
	if target != "" {
		args = append(args, target)
	}
	if branch != "" {
		args = append(args, branch)
	}
	return args
}

// buildPullArgs 构造 git pull 命令的参数列表。
// target 为最终使用的 remote 名。
func buildPullArgs(target, branch string) []string {
	args := []string{"pull"}
	if target != "" {
		args = append(args, target)
	}
	if branch != "" {
		args = append(args, branch)
	}
	return args
}

// buildRemoteAddArgs 构造 git remote add 命令的参数列表。
func buildRemoteAddArgs(name, url string) ([]string, error) {
	if name == "" || url == "" {
		return nil, &InvalidArgumentError{Msg: "Both remote name and URL are required to add a git remote."}
	}
	return []string{"remote", "add", name, url}, nil
}

// buildRemoteSetURLArgs 构造 git remote set-url 命令的参数列表。
func buildRemoteSetURLArgs(name, url string) []string {
	return []string{"remote", "set-url", name, url}
}

// buildRemoteGetURLArgs 构造 git remote get-url 命令的参数列表。
func buildRemoteGetURLArgs(name string) []string {
	return []string{"remote", "get-url", name}
}

// buildHasUpstreamArgs 构造 upstream 检查命令的参数列表。
func buildHasUpstreamArgs() []string {
	return []string{"rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"}
}

// buildStatusArgs 构造 git status 命令的参数列表。
func buildStatusArgs() []string {
	return []string{"status", "--porcelain=1", "-b"}
}

// buildBranchesArgs 构造 git branch 列表命令的参数列表。
func buildBranchesArgs() []string {
	return []string{"branch", "--format=%(refname:short)\t%(HEAD)"}
}

// buildCreateBranchArgs 构造 git checkout -b 命令的参数列表。
func buildCreateBranchArgs(branch string) []string {
	return []string{"checkout", "-b", branch}
}

// buildCheckoutBranchArgs 构造 git checkout 命令的参数列表。
func buildCheckoutBranchArgs(branch string) []string {
	return []string{"checkout", branch}
}

// buildDeleteBranchArgs 构造 git branch 删除命令的参数列表。
func buildDeleteBranchArgs(branch string, force bool) []string {
	flag := "-d"
	if force {
		flag = "-D"
	}
	return []string{"branch", flag, branch}
}

// buildAddArgs 构造 git add 命令的参数列表。
func buildAddArgs(files []string, all bool) []string {
	args := []string{"add"}
	if len(files) == 0 {
		if all {
			args = append(args, "-A")
		} else {
			args = append(args, ".")
		}
		return args
	}
	args = append(args, "--")
	args = append(args, files...)
	return args
}

// buildCommitArgs 构造 git commit 命令的参数列表。
func buildCommitArgs(message, authorName, authorEmail string, allowEmpty bool) []string {
	args := []string{"commit", "-m", message}
	if allowEmpty {
		args = append(args, "--allow-empty")
	}
	var prefix []string
	if authorName != "" {
		prefix = append(prefix, "-c", "user.name="+authorName)
	}
	if authorEmail != "" {
		prefix = append(prefix, "-c", "user.email="+authorEmail)
	}
	if len(prefix) > 0 {
		return append(prefix, args...)
	}
	return args
}

// allowedResetModes 是 git reset 允许的模式集合。
var allowedResetModes = map[GitResetMode]struct{}{
	GitResetModeSoft:  {},
	GitResetModeMixed: {},
	GitResetModeHard:  {},
	GitResetModeMerge: {},
	GitResetModeKeep:  {},
}

// buildResetArgs 构造 git reset 命令的参数列表。
// git reset 的两种用法互斥：带 mode 时重置 HEAD/索引/工作区，带 paths 时仅取消暂存路径。
func buildResetArgs(mode GitResetMode, target string, paths []string) ([]string, error) {
	if mode != "" {
		if _, ok := allowedResetModes[mode]; !ok {
			return nil, &InvalidArgumentError{Msg: "Reset mode must be one of soft, mixed, hard, merge, keep."}
		}
	}
	if mode != "" && len(paths) > 0 {
		return nil, &InvalidArgumentError{Msg: "Reset mode and paths cannot be used together."}
	}
	args := []string{"reset"}
	if mode != "" {
		args = append(args, "--"+string(mode))
	}
	if target != "" {
		args = append(args, target)
	}
	if len(paths) > 0 {
		args = append(args, "--")
		args = append(args, paths...)
	}
	return args, nil
}

// buildRestoreArgs 构造 git restore 命令的参数列表。
// 与原生 git 对齐：未显式指定标志时默认仅恢复工作区；任一标志显式 false、另一标志未指定时
// 也退化到默认 worktree，避免对调用方抛"必须有一个为 true"的语义错。显式两者都为 false 才报错。
func buildRestoreArgs(paths []string, staged, worktree *bool, source string) ([]string, error) {
	if len(paths) == 0 {
		return nil, &InvalidArgumentError{Msg: "At least one path is required."}
	}

	stagedOn := staged != nil && *staged
	worktreeOn := worktree != nil && *worktree
	if !stagedOn && !worktreeOn && (staged == nil || worktree == nil) {
		worktreeOn = true
	}

	if !stagedOn && !worktreeOn {
		return nil, &InvalidArgumentError{Msg: "At least one of staged or worktree must be true."}
	}

	args := []string{"restore"}
	if worktreeOn {
		args = append(args, "--worktree")
	}
	if stagedOn {
		args = append(args, "--staged")
	}
	if source != "" {
		args = append(args, "--source", source)
	}
	args = append(args, "--")
	args = append(args, paths...)
	return args, nil
}

// clonePlan 描述一次 clone 的完整执行计划。
type clonePlan struct {
	// args 是 git clone 命令的参数列表。
	args []string
	// repoPath 是用于 post-clone 调整的仓库路径。
	repoPath string
	// sanitizedURL 是凭证剥离后的 URL，用于 clone 后重置 origin。
	sanitizedURL string
	// shouldStrip 表示是否需要在 clone 完成后重置 origin URL。
	shouldStrip bool
}

// buildClonePlan 构造 clone 命令的参数与凭证剥离元信息。
func buildClonePlan(rawURL, path, branch string, depth int, username, password string, dangerouslyStore bool) (*clonePlan, error) {
	// 调用方直接把凭证嵌入 URL 时也强制要求 https，与 withCredentials 注入路径保持一致的安全边界，
	// 避免 `http://user:pass@host/...` 这类绕过明文走 HTTP。
	if err := requireHTTPSIfHasCredentials(rawURL); err != nil {
		return nil, err
	}
	cloneURL := rawURL
	if username != "" && password != "" {
		var err error
		cloneURL, err = withCredentials(rawURL, username, password)
		if err != nil {
			return nil, err
		}
	}
	sanitized := stripCredentials(cloneURL)
	shouldStrip := !dangerouslyStore && sanitized != cloneURL

	repoPath := path
	if shouldStrip {
		if repoPath == "" {
			repoPath = deriveRepoDirFromURL(rawURL)
		}
		if repoPath == "" {
			return nil, &InvalidArgumentError{Msg: "A destination path is required when using credentials without storing them."}
		}
	}

	args := []string{"clone", cloneURL}
	if branch != "" {
		args = append(args, "--branch", branch, "--single-branch")
	}
	if depth > 0 {
		args = append(args, "--depth", strconv.Itoa(depth))
	}
	if path != "" {
		args = append(args, path)
	}

	plan := &clonePlan{args: args, repoPath: repoPath}
	if shouldStrip {
		plan.sanitizedURL = sanitized
		plan.shouldStrip = true
	}
	return plan, nil
}
