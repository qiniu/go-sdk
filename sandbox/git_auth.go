package sandbox

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// GitAuthError 表示 git 操作因认证失败而中断。
type GitAuthError struct {
	// Msg 是错误消息。
	Msg string
}

// Error 实现 error 接口。
func (e *GitAuthError) Error() string { return e.Msg }

// GitUpstreamError 表示 git 操作因缺少 upstream 跟踪分支而中断。
type GitUpstreamError struct {
	// Msg 是错误消息。
	Msg string
}

// Error 实现 error 接口。
func (e *GitUpstreamError) Error() string { return e.Msg }

// InvalidArgumentError 表示传入的参数非法。
type InvalidArgumentError struct {
	// Msg 是错误消息。
	Msg string
}

// Error 实现 error 接口。
func (e *InvalidArgumentError) Error() string { return e.Msg }

// withCredentials 将 HTTPS 凭证嵌入 git 仓库 URL 中。
// 仅 https 协议有效；当 username 与 password 均为空时直接返回原 URL。
//
// 故意不允许 http：把 token / 密码塞进明文 URL 会让凭证暴露在非 TLS 链路上，
// 与对外"仅支持 HTTPS + username/password"的承诺一致。
func withCredentials(rawURL, username, password string) (string, error) {
	if username == "" && password == "" {
		return rawURL, nil
	}
	if username == "" || password == "" {
		return "", &InvalidArgumentError{Msg: "Both username and password are required when using Git credentials."}
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse git url: %w", err)
	}
	if u.Scheme != "https" {
		return "", &InvalidArgumentError{Msg: "Only https Git URLs support username/password credentials."}
	}

	u.User = url.UserPassword(username, password)
	return u.String(), nil
}

// stripCredentials 从 git URL 中移除已嵌入的凭证。
// 解析失败或非 http(s) URL 时原样返回。
func stripCredentials(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return rawURL
	}
	if u.User == nil {
		return rawURL
	}
	u.User = nil
	return u.String()
}

// requireHTTPSIfHasCredentials 校验 URL 在内嵌凭证时必须使用 https。
// 解析失败或未内嵌凭证时返回 nil，把校验机会留给后续命令本身。
func requireHTTPSIfHasCredentials(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil || u.User == nil {
		return nil
	}
	if u.Scheme != "https" {
		return &InvalidArgumentError{Msg: "Only https Git URLs support inline username/password credentials."}
	}
	return nil
}

// authFailureSnippets 是常见的 git 认证失败关键字（小写匹配）。
//
// 这里只放与"凭证"语义强相关的关键字，避免把通用的 "permission denied"
// （工作树目录无写权限、.git/config 权限异常、锁文件权限问题等）误判为
// 认证失败而把调用方导向错误的排障方向。
var authFailureSnippets = []string{
	"authentication failed",
	"terminal prompts disabled",
	"could not read username",
	"could not read password",
	"invalid username or password",
	"bad credentials",
	"requested url returned error: 401",
	"requested url returned error: 403",
	"http basic: access denied",
}

// upstreamFailureSnippets 是常见的 git 缺失 upstream 关键字（小写匹配）。
var upstreamFailureSnippets = []string{
	"has no upstream branch",
	"no upstream branch",
	"no upstream configured",
	"no tracking information for the current branch",
	"no tracking information",
	"set the remote as upstream",
	"set the upstream branch",
	"please specify which branch you want to merge with",
}

// gitCommandError 是内部使用的 git 命令失败错误，附带 stdout/stderr 以便后续匹配。
type gitCommandError struct {
	// Cmd 是失败的 git 子命令名（如 "clone"、"push"）。
	Cmd string
	// Result 是底层命令的执行结果，包含 ExitCode/Stdout/Stderr。
	Result *CommandResult
}

// Error 实现 error 接口。
func (e *gitCommandError) Error() string {
	if e.Result == nil {
		return fmt.Sprintf("git %s failed", e.Cmd)
	}
	if stderr := strings.TrimSpace(e.Result.Stderr); stderr != "" {
		return fmt.Sprintf("git %s failed (exit %d): %s", e.Cmd, e.Result.ExitCode, stderr)
	}
	return fmt.Sprintf("git %s failed (exit %d)", e.Cmd, e.Result.ExitCode)
}

// matchSnippets 在 git 命令错误的 stdout+stderr 中匹配关键字（小写匹配）。
func matchSnippets(err error, snippets []string) bool {
	var ge *gitCommandError
	if !errors.As(err, &ge) || ge.Result == nil {
		return false
	}
	message := strings.ToLower(ge.Result.Stderr + "\n" + ge.Result.Stdout)
	for _, s := range snippets {
		if strings.Contains(message, s) {
			return true
		}
	}
	return false
}

// isAuthFailure 判断错误是否由 git 认证失败引起。
func isAuthFailure(err error) bool { return matchSnippets(err, authFailureSnippets) }

// isMissingUpstream 判断错误是否由缺失 upstream 跟踪引起。
func isMissingUpstream(err error) bool { return matchSnippets(err, upstreamFailureSnippets) }

// buildAuthErrorMessage 根据动作名构造 git 认证错误消息。
func buildAuthErrorMessage(action string, missingPassword bool) string {
	if missingPassword {
		return fmt.Sprintf("Git %s requires a password/token for private repositories.", action)
	}
	return fmt.Sprintf("Git %s requires credentials for private repositories.", action)
}

// buildUpstreamErrorMessage 根据动作名构造缺失 upstream 的错误消息。
func buildUpstreamErrorMessage(action string) string {
	if action == "push" {
		return "Git push failed because no upstream branch is configured. " +
			"Set upstream once with SetUpstream=true (and optional Remote/Branch), " +
			"or pass Remote and Branch explicitly."
	}
	return "Git pull failed because no upstream branch is configured. " +
		"Pass Remote and Branch explicitly, or set upstream once (push with " +
		"SetUpstream=true or run: git branch --set-upstream-to=origin/<branch> <branch>)."
}
