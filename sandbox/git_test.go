//go:build unit

package sandbox

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShellEscape(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"simple", "'simple'"},
		{"a b", "'a b'"},
		{"it's", `'it'"'"'s'`},
		{"", "''"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, shellEscape(c.in), "input=%q", c.in)
	}
}

func TestBuildGitCommand(t *testing.T) {
	cmd := buildGitCommand([]string{"status", "--porcelain=1"}, "/repo path")
	assert.Equal(t, `'git' '-C' '/repo path' 'status' '--porcelain=1'`, cmd)

	cmd = buildGitCommand([]string{"init"}, "")
	assert.Equal(t, `'git' 'init'`, cmd)
}

func TestBuildPushPullArgs(t *testing.T) {
	assert.Equal(t, []string{"push", "--set-upstream", "origin", "main"},
		buildPushArgs("origin", "main", true))
	assert.Equal(t, []string{"push"}, buildPushArgs("", "", true))
	assert.Equal(t, []string{"push", "origin", "main"},
		buildPushArgs("origin", "main", false))

	assert.Equal(t, []string{"pull", "origin", "main"},
		buildPullArgs("origin", "main"))
	assert.Equal(t, []string{"pull"}, buildPullArgs("", ""))
}

func TestBuildAddArgs(t *testing.T) {
	assert.Equal(t, []string{"add", "."}, buildAddArgs(nil, false))
	assert.Equal(t, []string{"add", "-A"}, buildAddArgs(nil, true))
	assert.Equal(t, []string{"add", "--", "a.go", "b.go"}, buildAddArgs([]string{"a.go", "b.go"}, true))
}

func TestBuildCommitArgs(t *testing.T) {
	got := buildCommitArgs("msg", "Alice", "a@x.io", true)
	assert.Equal(t,
		[]string{"-c", "user.name=Alice", "-c", "user.email=a@x.io", "commit", "-m", "msg", "--allow-empty"},
		got,
	)
	assert.Equal(t, []string{"commit", "-m", "msg"}, buildCommitArgs("msg", "", "", false))
}

func TestBuildResetArgs(t *testing.T) {
	args, err := buildResetArgs(GitResetModeHard, "HEAD~1", nil)
	assert.NoError(t, err)
	assert.Equal(t, []string{"reset", "--hard", "HEAD~1"}, args)

	args, err = buildResetArgs("", "", []string{"a.go"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"reset", "--", "a.go"}, args)

	// mode 与 paths 同时给出会被拒绝（git reset 的两种用法互斥）
	_, err = buildResetArgs(GitResetModeHard, "HEAD~1", []string{"a.go"})
	assert.Error(t, err)
	var ie *InvalidArgumentError
	assert.True(t, errors.As(err, &ie))

	_, err = buildResetArgs(GitResetMode("invalid"), "", nil)
	assert.Error(t, err)
	assert.True(t, errors.As(err, &ie))
}

func TestBuildRestoreArgs(t *testing.T) {
	// 默认仅恢复工作区
	args, err := buildRestoreArgs([]string{"a.go"}, nil, nil, "")
	assert.NoError(t, err)
	assert.Equal(t, []string{"restore", "--worktree", "--", "a.go"}, args)

	// staged=true 时默认不恢复工作区
	tr := true
	args, err = buildRestoreArgs([]string{"a.go"}, &tr, nil, "")
	assert.NoError(t, err)
	assert.Equal(t, []string{"restore", "--staged", "--", "a.go"}, args)

	// 显式两者都开启
	args, err = buildRestoreArgs([]string{"a.go"}, &tr, &tr, "HEAD")
	assert.NoError(t, err)
	assert.Equal(t, []string{"restore", "--worktree", "--staged", "--source", "HEAD", "--", "a.go"}, args)

	_, err = buildRestoreArgs(nil, nil, nil, "")
	assert.Error(t, err)
}

func TestBuildClonePlan_NoCredentials(t *testing.T) {
	plan, err := buildClonePlan("https://github.com/o/r.git", "/tmp/r", "main", 1, "", "", false)
	assert.NoError(t, err)
	assert.False(t, plan.shouldStrip)
	assert.Equal(t,
		[]string{"clone", "https://github.com/o/r.git", "--branch", "main", "--single-branch", "--depth", "1", "/tmp/r"},
		plan.args,
	)
}

func TestBuildClonePlan_StripsCredentials(t *testing.T) {
	plan, err := buildClonePlan("https://github.com/o/r.git", "", "", 0, "alice", "tk", false)
	assert.NoError(t, err)
	assert.True(t, plan.shouldStrip)
	assert.Equal(t, "https://github.com/o/r.git", plan.sanitizedURL)
	assert.Equal(t, "r", plan.repoPath)
	// args[1] 应包含凭证
	assert.Contains(t, plan.args[1], "alice:tk@")
}

func TestBuildClonePlan_StoresCredentials(t *testing.T) {
	plan, err := buildClonePlan("https://github.com/o/r.git", "/tmp/r", "", 0, "alice", "tk", true)
	assert.NoError(t, err)
	assert.False(t, plan.shouldStrip)
}

func TestWithCredentials(t *testing.T) {
	got, err := withCredentials("https://github.com/o/r.git", "alice", "tk:1")
	assert.NoError(t, err)
	assert.Equal(t, "https://alice:tk%3A1@github.com/o/r.git", got)

	// 仅一个不为空 -> 错
	_, err = withCredentials("https://github.com/o/r.git", "alice", "")
	assert.Error(t, err)

	// 非 http(s) -> 错
	_, err = withCredentials("git@github.com:o/r.git", "alice", "tk")
	assert.Error(t, err)

	// 都为空 -> 原样返回
	got, err = withCredentials("https://github.com/o/r.git", "", "")
	assert.NoError(t, err)
	assert.Equal(t, "https://github.com/o/r.git", got)
}

func TestStripCredentials(t *testing.T) {
	assert.Equal(t, "https://github.com/o/r.git",
		stripCredentials("https://alice:tk@github.com/o/r.git"))
	assert.Equal(t, "https://github.com/o/r.git",
		stripCredentials("https://github.com/o/r.git"))
	assert.Equal(t, "git@github.com:o/r.git",
		stripCredentials("git@github.com:o/r.git"))
}

func TestDeriveRepoDirFromURL(t *testing.T) {
	cases := map[string]string{
		"https://github.com/o/r.git":        "r",
		"https://github.com/o/r":            "r",
		"git@github.com:o/r.git":            "r",
		"https://github.com/o/r.git?ref=v1": "r",
	}
	for in, want := range cases {
		assert.Equal(t, want, deriveRepoDirFromURL(in), "input=%s", in)
	}
}

func TestParseGitStatus_BranchHeader(t *testing.T) {
	out := "## main...origin/main [ahead 1, behind 2]\n"
	s := parseGitStatus(out)
	assert.Equal(t, "main", s.CurrentBranch)
	assert.Equal(t, "origin/main", s.Upstream)
	assert.Equal(t, 1, s.Ahead)
	assert.Equal(t, 2, s.Behind)
	assert.False(t, s.Detached)
	assert.True(t, s.IsClean())
}

func TestParseGitStatus_DetachedAndFiles(t *testing.T) {
	out := strings.Join([]string{
		"## HEAD (no branch)",
		"M  staged.go",
		" M dirty.go",
		"?? new.go",
		"R  old.go -> renamed.go",
		"UU conflict.go",
		"",
	}, "\n")
	s := parseGitStatus(out)
	assert.True(t, s.Detached)
	assert.Equal(t, 5, s.TotalCount())
	assert.Equal(t, 2, s.StagedCount()) // staged.go + renamed.go
	assert.True(t, s.HasUntracked())
	assert.True(t, s.HasConflicts())

	byName := map[string]GitFileStatus{}
	for _, f := range s.FileStatus {
		byName[f.Name] = f
	}
	assert.Equal(t, "modified", byName["staged.go"].Status)
	assert.True(t, byName["staged.go"].Staged)
	assert.Equal(t, "modified", byName["dirty.go"].Status)
	assert.False(t, byName["dirty.go"].Staged)
	assert.Equal(t, "untracked", byName["new.go"].Status)
	assert.Equal(t, "renamed", byName["renamed.go"].Status)
	assert.Equal(t, "old.go", byName["renamed.go"].RenamedFrom)
	assert.Equal(t, "conflict", byName["conflict.go"].Status)
}

func TestParseGitStatus_DetachedAtRef(t *testing.T) {
	s := parseGitStatus("## HEAD (detached at v1.2.3)\n")
	assert.True(t, s.Detached)
	assert.Empty(t, s.CurrentBranch)
	assert.Empty(t, s.Upstream)
}

func TestParseGitStatus_UnbornBranch(t *testing.T) {
	// "git status --porcelain=1 -b" 在仓库尚未首次提交时输出 "## No commits yet on <branch>"
	s := parseGitStatus("## No commits yet on main\n?? new.go\n")
	assert.False(t, s.Detached)
	assert.Equal(t, "main", s.CurrentBranch)
	assert.Empty(t, s.Upstream)
	assert.Equal(t, 1, s.UntrackedCount())

	// 旧版 git 用 "Initial commit on" 措辞
	s = parseGitStatus("## Initial commit on master\n")
	assert.Equal(t, "master", s.CurrentBranch)
	assert.False(t, s.Detached)
}

func TestParseGitStatus_AheadBehindOnly(t *testing.T) {
	s := parseGitStatus("## main...origin/main [ahead 3]\n")
	assert.Equal(t, "main", s.CurrentBranch)
	assert.Equal(t, 3, s.Ahead)
	assert.Equal(t, 0, s.Behind)
}

func TestParseGitBranches(t *testing.T) {
	out := "main\t*\nfeature\t \nrelease\t\n"
	b := parseGitBranches(out)
	assert.Equal(t, []string{"main", "feature", "release"}, b.Branches)
	assert.Equal(t, "main", b.CurrentBranch)
}

func TestParseGitBranches_DetachedHEAD(t *testing.T) {
	// detached HEAD 时 git 会输出 "(HEAD detached at <sha>)" 并标记为当前；
	// 该伪分支不应进入 Branches，CurrentBranch 也应保持为空。
	out := "(HEAD detached at 1234abc)\t*\nmain\t \nfeature\t \n"
	b := parseGitBranches(out)
	assert.Equal(t, []string{"main", "feature"}, b.Branches)
	assert.Empty(t, b.CurrentBranch)
}

func TestParseGitStatus_QuotedPaths(t *testing.T) {
	// porcelain v1 在文件名含空格、引号、非 ASCII 时会输出 C-style 引号路径。
	out := strings.Join([]string{
		`?? "with space.txt"`,
		` M "quote\"name.txt"`,
		`R  "old name.txt" -> "new name.txt"`,
		`A  "tab\there.txt"`,
		"",
	}, "\n")
	s := parseGitStatus(out)
	byName := map[string]GitFileStatus{}
	for _, f := range s.FileStatus {
		byName[f.Name] = f
	}
	assert.Contains(t, byName, "with space.txt")
	assert.Contains(t, byName, `quote"name.txt`)
	assert.Contains(t, byName, "new name.txt")
	assert.Equal(t, "old name.txt", byName["new name.txt"].RenamedFrom)
	assert.Contains(t, byName, "tab\there.txt")
}

func TestUnquoteCPath(t *testing.T) {
	cases := map[string]string{
		`plain.txt`:         `plain.txt`,
		`"with space.txt"`:  `with space.txt`,
		`"a\"b"`:            `a"b`,
		`"a\\b"`:            `a\b`,
		`"tab\there"`:       "tab\there",
		`"newline\nhere"`:   "newline\nhere",
		`"unicode\303\251"`: "unicode\xc3\xa9", // é (UTF-8)
	}
	for in, want := range cases {
		assert.Equal(t, want, unquoteCPath(in), "input=%q", in)
	}
}

func TestUnstagedCount_MMCountedOnce(t *testing.T) {
	// "MM file" 表示同一文件既有 staged 又有 unstaged 改动，应同时计入两边。
	s := parseGitStatus("## main\nMM file.go\n M dirty.go\n?? new.go\n")
	assert.Equal(t, 1, s.StagedCount())
	assert.Equal(t, 2, s.UnstagedCount())
	assert.Equal(t, 1, s.UntrackedCount())
}

func TestIsAuthFailure(t *testing.T) {
	err := &gitCommandError{Cmd: "push", Result: &CommandResult{ExitCode: 128, Stderr: "fatal: Authentication failed for foo"}}
	assert.True(t, isAuthFailure(err))
	assert.False(t, isMissingUpstream(err))

	err = &gitCommandError{Cmd: "pull", Result: &CommandResult{Stderr: "There is no tracking information for the current branch"}}
	assert.True(t, isMissingUpstream(err))
	assert.False(t, isAuthFailure(err))

	assert.False(t, isAuthFailure(errors.New("other")))
}

func TestResolveConfigScope(t *testing.T) {
	flag, p, err := resolveConfigScope("", "")
	assert.NoError(t, err)
	assert.Equal(t, "--global", flag)
	assert.Empty(t, p)

	flag, p, err = resolveConfigScope(GitConfigScopeLocal, "/repo")
	assert.NoError(t, err)
	assert.Equal(t, "--local", flag)
	assert.Equal(t, "/repo", p)

	_, _, err = resolveConfigScope(GitConfigScopeLocal, "")
	assert.Error(t, err)

	_, _, err = resolveConfigScope("weird", "")
	assert.Error(t, err)
}

func TestBuildAuthErrorMessage(t *testing.T) {
	assert.Contains(t, buildAuthErrorMessage("clone", true), "password/token")
	assert.Contains(t, buildAuthErrorMessage("push", false), "credentials")
	assert.Contains(t, buildUpstreamErrorMessage("push"), "upstream")
	assert.Contains(t, buildUpstreamErrorMessage("pull"), "upstream")
}
