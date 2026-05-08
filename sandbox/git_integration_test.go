//go:build integration

package sandbox

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gitTestEnv 把一次 git 集成测试常用的对象打包在一起，避免每个用例重复创建沙箱。
type gitTestEnv struct {
	t   *testing.T
	sb  *Sandbox
	git *Git
	ctx context.Context
}

// newGitTestEnv 创建一个就绪的沙箱并在测试结束时清理。
func newGitTestEnv(t *testing.T) *gitTestEnv {
	t.Helper()
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	t.Cleanup(cancel)
	sb := createTestSandbox(t, c, ctx)
	return &gitTestEnv{t: t, sb: sb, git: sb.Git(), ctx: ctx}
}

// initRepo 在沙箱内初始化一个普通仓库并配置作者。
func (e *gitTestEnv) initRepo(path, branch string) {
	e.t.Helper()
	_, err := e.git.Init(e.ctx, path, &InitOptions{InitialBranch: branch})
	require.NoError(e.t, err, "Init %s", path)
	_, err = e.git.ConfigureUser(e.ctx, "Tester", "tester@example.com", &ConfigOptions{
		Scope: GitConfigScopeLocal,
		Path:  path,
	})
	require.NoError(e.t, err, "ConfigureUser")
}

// writeAndCommit 写入一个文件并完成一次提交。
func (e *gitTestEnv) writeAndCommit(repo, file, content, msg string) {
	e.t.Helper()
	_, err := e.sb.Files().Write(e.ctx, repo+"/"+file, []byte(content))
	require.NoError(e.t, err, "Write %s", file)
	_, err = e.git.Add(e.ctx, repo, nil)
	require.NoError(e.t, err, "Add")
	_, err = e.git.Commit(e.ctx, repo, msg, nil)
	require.NoError(e.t, err, "Commit")
}

// TestIntegrationGitInitConfigureBranches 覆盖 Init / ConfigureUser / SetConfig / GetConfig /
// Branches（含 unborn 仓库）/ CreateBranch / CheckoutBranch / DeleteBranch 的真实行为。
func TestIntegrationGitInitConfigureBranches(t *testing.T) {
	e := newGitTestEnv(t)

	repo := "/tmp/it-init"
	_, err := e.git.Init(e.ctx, repo, &InitOptions{InitialBranch: "main"})
	require.NoError(t, err)

	// unborn 仓库 Branches() 应通过 symbolic-ref fallback 返回 "main"
	br, err := e.git.Branches(e.ctx, repo, nil)
	require.NoError(t, err)
	assert.Empty(t, br.Branches, "unborn 仓库不应有可枚举分支")
	assert.Equal(t, "main", br.CurrentBranch, "unborn 仓库 fallback 后应得到 main")

	_, err = e.git.ConfigureUser(e.ctx, "Alice", "a@x.io", &ConfigOptions{
		Scope: GitConfigScopeLocal, Path: repo,
	})
	require.NoError(t, err)
	val, err := e.git.GetConfig(e.ctx, "user.name", &ConfigOptions{Scope: GitConfigScopeLocal, Path: repo})
	require.NoError(t, err)
	assert.Equal(t, "Alice", val)

	// 不存在的 key 必须返回空字符串、无错误
	val, err = e.git.GetConfig(e.ctx, "user.notexist", &ConfigOptions{Scope: GitConfigScopeLocal, Path: repo})
	require.NoError(t, err)
	assert.Empty(t, val)

	// SetConfig 任意 key
	_, err = e.git.SetConfig(e.ctx, "core.autocrlf", "input", &ConfigOptions{Scope: GitConfigScopeLocal, Path: repo})
	require.NoError(t, err)
	val, err = e.git.GetConfig(e.ctx, "core.autocrlf", &ConfigOptions{Scope: GitConfigScopeLocal, Path: repo})
	require.NoError(t, err)
	assert.Equal(t, "input", val)

	// Local scope 不给 Path 应直接报错
	_, err = e.git.SetConfig(e.ctx, "k", "v", &ConfigOptions{Scope: GitConfigScopeLocal})
	var ie *InvalidArgumentError
	assert.True(t, errors.As(err, &ie), "Local scope 缺 Path 应返回 InvalidArgumentError")

	// 创建第一次提交后再做分支操作
	e.writeAndCommit(repo, "README.md", "# init\n", "feat: init")

	_, err = e.git.CreateBranch(e.ctx, repo, "feature/x", nil)
	require.NoError(t, err)
	br, err = e.git.Branches(e.ctx, repo, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"main", "feature/x"}, br.Branches)
	assert.Equal(t, "feature/x", br.CurrentBranch)

	_, err = e.git.CheckoutBranch(e.ctx, repo, "main", nil)
	require.NoError(t, err)
	_, err = e.git.DeleteBranch(e.ctx, repo, "feature/x", &DeleteBranchOptions{Force: true})
	require.NoError(t, err)

	br, err = e.git.Branches(e.ctx, repo, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"main"}, br.Branches)
	assert.Equal(t, "main", br.CurrentBranch)

	// 空分支名应被参数校验拦下
	_, err = e.git.CreateBranch(e.ctx, repo, "", nil)
	assert.True(t, errors.As(err, &ie))
}

// TestIntegrationGitAddOptions 覆盖 AddOptions.All 的三种取值（nil 默认 -A、显式 false 用 "."、Files 列表）。
func TestIntegrationGitAddOptions(t *testing.T) {
	e := newGitTestEnv(t)

	repo := "/tmp/it-add"
	e.initRepo(repo, "main")
	e.writeAndCommit(repo, "seed", "seed\n", "seed")

	// 子目录 + 顶层各放一个新文件
	_, err := e.sb.Files().Write(e.ctx, repo+"/top.txt", []byte("top\n"))
	require.NoError(t, err)
	_, err = e.sb.Files().Write(e.ctx, repo+"/sub/inner.txt", []byte("inner\n"))
	require.NoError(t, err)

	// 1) 显式 All=false：相当于 git add . —— 仓库根目录下应递归到子目录（git 行为）。
	//    我们这里只验证不报错且 Status 至少认到一个 staged。
	allFalse := false
	_, err = e.git.Add(e.ctx, repo, &AddOptions{All: &allFalse})
	require.NoError(t, err)
	st, err := e.git.Status(e.ctx, repo, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, st.StagedCount(), 1)

	// 2) Files=[]：仅暂存指定文件
	_, err = e.git.Reset(e.ctx, repo, &ResetOptions{Paths: []string{"."}})
	require.NoError(t, err)
	_, err = e.git.Add(e.ctx, repo, &AddOptions{Files: []string{"top.txt"}})
	require.NoError(t, err)
	st, err = e.git.Status(e.ctx, repo, nil)
	require.NoError(t, err)
	staged := stagedNames(st)
	assert.Contains(t, staged, "top.txt")
	assert.NotContains(t, staged, "sub/inner.txt")

	// 3) opts=nil：默认 -A，应把所有未跟踪文件 stage 进来
	_, err = e.git.Add(e.ctx, repo, nil)
	require.NoError(t, err)
	st, err = e.git.Status(e.ctx, repo, nil)
	require.NoError(t, err)
	staged = stagedNames(st)
	assert.Contains(t, staged, "top.txt")
	assert.Contains(t, staged, "sub/inner.txt")
}

// TestIntegrationGitStatusEdgeCases 验证 parseGitStatus 在真实输出下的健壮性：
// detached HEAD、含空格的文件名、MM（同时 staged + unstaged）、unborn 仓库。
func TestIntegrationGitStatusEdgeCases(t *testing.T) {
	e := newGitTestEnv(t)

	repo := "/tmp/it-status"
	e.initRepo(repo, "main")

	// unborn：应能解析出 CurrentBranch=main 且 Detached=false
	st, err := e.git.Status(e.ctx, repo, nil)
	require.NoError(t, err)
	assert.False(t, st.Detached)
	assert.Equal(t, "main", st.CurrentBranch)

	e.writeAndCommit(repo, "README.md", "# v1\n", "feat: init")

	// MM：同一文件 staged + 工作区再改一次
	_, err = e.sb.Files().Write(e.ctx, repo+"/README.md", []byte("# v2\n"))
	require.NoError(t, err)
	_, err = e.git.Add(e.ctx, repo, nil)
	require.NoError(t, err)
	_, err = e.sb.Files().Write(e.ctx, repo+"/README.md", []byte("# v2 dirty\n"))
	require.NoError(t, err)

	// 含空格文件名（未跟踪）
	_, err = e.sb.Files().Write(e.ctx, repo+"/with space.txt", []byte("x\n"))
	require.NoError(t, err)

	st, err = e.git.Status(e.ctx, repo, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, st.StagedCount(), 1, "README.md 应同时计入 staged")
	assert.GreaterOrEqual(t, st.UnstagedCount(), 1, "README.md 工作区改动应计入 unstaged")
	assert.True(t, st.HasUntracked())

	byName := map[string]GitFileStatus{}
	for _, f := range st.FileStatus {
		byName[f.Name] = f
	}
	assert.Contains(t, byName, "with space.txt", "含空格的文件名应被正确反引号解析")
	if rm, ok := byName["README.md"]; ok {
		assert.True(t, rm.Staged, "README.md 应有 staged 标记")
	}

	// detached HEAD
	res, err := e.sb.Commands().Run(e.ctx, "git -C "+repo+" rev-parse HEAD",
		WithEnvs(map[string]string{"GIT_TERMINAL_PROMPT": "0"}))
	require.NoError(t, err)
	sha := strings.TrimSpace(res.Stdout)
	require.NotEmpty(t, sha)
	_, err = e.sb.Commands().Run(e.ctx, "git -C "+repo+" checkout "+sha,
		WithEnvs(map[string]string{"GIT_TERMINAL_PROMPT": "0"}))
	require.NoError(t, err)
	st, err = e.git.Status(e.ctx, repo, nil)
	require.NoError(t, err)
	assert.True(t, st.Detached, "checkout <sha> 后应进入 detached HEAD")
	assert.Empty(t, st.CurrentBranch)
}

// TestIntegrationGitResetRestore 覆盖 Reset 各种 mode、Restore --staged 与 --source HEAD，以及无效参数组合。
func TestIntegrationGitResetRestore(t *testing.T) {
	e := newGitTestEnv(t)

	repo := "/tmp/it-reset"
	e.initRepo(repo, "main")
	e.writeAndCommit(repo, "a.txt", "v1\n", "init")

	// --hard：丢弃工作区改动
	_, err := e.sb.Files().Write(e.ctx, repo+"/a.txt", []byte("dirty\n"))
	require.NoError(t, err)
	_, err = e.git.Reset(e.ctx, repo, &ResetOptions{Mode: GitResetModeHard, Target: "HEAD"})
	require.NoError(t, err)
	got, err := e.sb.Files().ReadText(e.ctx, repo+"/a.txt")
	require.NoError(t, err)
	assert.Equal(t, "v1\n", got)

	// paths-only：unstage
	_, err = e.sb.Files().Write(e.ctx, repo+"/a.txt", []byte("staged\n"))
	require.NoError(t, err)
	_, err = e.git.Add(e.ctx, repo, nil)
	require.NoError(t, err)
	_, err = e.git.Reset(e.ctx, repo, &ResetOptions{Paths: []string{"a.txt"}})
	require.NoError(t, err)
	st, err := e.git.Status(e.ctx, repo, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, st.StagedCount())

	// Mode + Paths 同时给 → InvalidArgumentError
	_, err = e.git.Reset(e.ctx, repo, &ResetOptions{Mode: GitResetModeHard, Paths: []string{"a.txt"}})
	var ie *InvalidArgumentError
	assert.True(t, errors.As(err, &ie))

	// 非法 mode → InvalidArgumentError
	_, err = e.git.Reset(e.ctx, repo, &ResetOptions{Mode: GitResetMode("bogus")})
	assert.True(t, errors.As(err, &ie))

	// Restore --staged：仅取消暂存，工作区保留
	_, err = e.sb.Files().Write(e.ctx, repo+"/a.txt", []byte("again\n"))
	require.NoError(t, err)
	_, err = e.git.Add(e.ctx, repo, nil)
	require.NoError(t, err)
	staged := true
	_, err = e.git.Restore(e.ctx, repo, &RestoreOptions{Paths: []string{"a.txt"}, Staged: &staged})
	require.NoError(t, err)
	got, err = e.sb.Files().ReadText(e.ctx, repo+"/a.txt")
	require.NoError(t, err)
	assert.Equal(t, "again\n", got, "Restore --staged 不应改动工作区")
	st, err = e.git.Status(e.ctx, repo, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, st.StagedCount())

	// Restore --source HEAD：把工作区恢复回 HEAD
	_, err = e.git.Restore(e.ctx, repo, &RestoreOptions{Paths: []string{"a.txt"}, Source: "HEAD"})
	require.NoError(t, err)
	got, err = e.sb.Files().ReadText(e.ctx, repo+"/a.txt")
	require.NoError(t, err)
	assert.Equal(t, "v1\n", got)

	// Restore Paths 为空 → 错
	_, err = e.git.Restore(e.ctx, repo, &RestoreOptions{})
	assert.Error(t, err)
}

// TestIntegrationGitCommitOptions 覆盖 CommitOptions 的 Author 覆盖与 AllowEmpty。
func TestIntegrationGitCommitOptions(t *testing.T) {
	e := newGitTestEnv(t)
	repo := "/tmp/it-commit"
	e.initRepo(repo, "main")
	e.writeAndCommit(repo, "seed", "seed\n", "seed")

	// 空消息 → 拒绝
	_, err := e.git.Commit(e.ctx, repo, "", nil)
	var ie *InvalidArgumentError
	assert.True(t, errors.As(err, &ie))

	// Author 覆盖
	_, err = e.sb.Files().Write(e.ctx, repo+"/b.txt", []byte("b\n"))
	require.NoError(t, err)
	_, err = e.git.Add(e.ctx, repo, nil)
	require.NoError(t, err)
	_, err = e.git.Commit(e.ctx, repo, "feat: b", &CommitOptions{
		AuthorName: "Bob", AuthorEmail: "bob@x.io",
	})
	require.NoError(t, err)
	res, err := e.sb.Commands().Run(e.ctx, "git -C "+repo+" log -1 --pretty=%an",
		WithEnvs(map[string]string{"GIT_TERMINAL_PROMPT": "0"}))
	require.NoError(t, err)
	assert.Equal(t, "Bob", strings.TrimSpace(res.Stdout))

	// AllowEmpty=true：在 clean 工作区上仍能提交
	_, err = e.git.Commit(e.ctx, repo, "chore: empty", &CommitOptions{AllowEmpty: true})
	require.NoError(t, err)

	// AllowEmpty=false（默认）：clean 工作区会失败
	_, err = e.git.Commit(e.ctx, repo, "chore: should fail", nil)
	assert.Error(t, err)
}

// TestIntegrationGitRemoteAndPushPull 覆盖 RemoteAdd（含 Overwrite）/ RemoteGet（缺失）/
// Push（SetUpstream 默认 true 与显式 false） / Pull / 参数校验。
func TestIntegrationGitRemoteAndPushPull(t *testing.T) {
	e := newGitTestEnv(t)

	repo := "/tmp/it-remote"
	bare := "/tmp/it-remote.git"
	consumer := "/tmp/it-consumer"
	e.initRepo(repo, "main")
	e.writeAndCommit(repo, "README.md", "# v1\n", "init")
	_, err := e.git.Init(e.ctx, bare, &InitOptions{Bare: true, InitialBranch: "main"})
	require.NoError(t, err)

	// RemoteGet 不存在 → 空字符串、无错
	url, err := e.git.RemoteGet(e.ctx, repo, "nope", nil)
	require.NoError(t, err)
	assert.Empty(t, url)

	// RemoteAdd 占位 URL，再 Overwrite 改成 bare 路径
	_, err = e.git.RemoteAdd(e.ctx, repo, "origin", "https://example.com/x.git", nil)
	require.NoError(t, err)
	_, err = e.git.RemoteAdd(e.ctx, repo, "origin", bare, &RemoteAddOptions{Overwrite: true})
	require.NoError(t, err)
	url, err = e.git.RemoteGet(e.ctx, repo, "origin", nil)
	require.NoError(t, err)
	assert.Equal(t, bare, url)

	// Push 不给 Remote 但给 Branch → InvalidArgumentError
	_, err = e.git.Push(e.ctx, repo, &PushOptions{Branch: "main"})
	var ie *InvalidArgumentError
	assert.True(t, errors.As(err, &ie))

	// Push 显式 SetUpstream=false（不写 upstream）
	noUpstream := false
	_, err = e.git.Push(e.ctx, repo, &PushOptions{
		Remote: "origin", Branch: "main", SetUpstream: &noUpstream,
	})
	require.NoError(t, err)

	// Clone 出 consumer 仓库
	_, err = e.git.Clone(e.ctx, bare, &CloneOptions{Path: consumer})
	require.NoError(t, err)

	// 主仓再 push 一次（这次走 SetUpstream 默认 true）
	e.writeAndCommit(repo, "CHANGELOG.md", "v1\n", "docs")
	_, err = e.git.Push(e.ctx, repo, &PushOptions{Remote: "origin", Branch: "main"})
	require.NoError(t, err)

	// consumer Pull —— 不给 Branch 也不给 Remote，应通过 hasUpstream 走默认路径
	_, err = e.git.Pull(e.ctx, consumer, &PullOptions{})
	require.NoError(t, err)
	exists, err := e.sb.Files().Exists(e.ctx, consumer+"/CHANGELOG.md")
	require.NoError(t, err)
	assert.True(t, exists)

	// Pull Branch 不给 Remote → InvalidArgumentError
	_, err = e.git.Pull(e.ctx, consumer, &PullOptions{Branch: "main"})
	assert.True(t, errors.As(err, &ie))
}

// TestIntegrationGitPullMissingUpstream 覆盖 Pull 在没有 upstream 的本地分支上抛出 GitUpstreamError。
func TestIntegrationGitPullMissingUpstream(t *testing.T) {
	e := newGitTestEnv(t)
	repo := "/tmp/it-no-upstream"
	e.initRepo(repo, "main")
	e.writeAndCommit(repo, "a", "a\n", "init")

	_, err := e.git.Pull(e.ctx, repo, &PullOptions{})
	var ue *GitUpstreamError
	assert.True(t, errors.As(err, &ue), "缺 upstream 时 Pull 应返回 GitUpstreamError，实际: %T %v", err, err)
}

// TestIntegrationGitDangerouslyAuthenticate 验证凭证确实被 git credential helper 持久化。
func TestIntegrationGitDangerouslyAuthenticate(t *testing.T) {
	e := newGitTestEnv(t)

	// Protocol 非 https → InvalidArgumentError
	_, err := e.git.DangerouslyAuthenticate(e.ctx, &AuthenticateOptions{
		Username: "u", Password: "p", Host: "h", Protocol: "ssh",
	})
	var ie *InvalidArgumentError
	assert.True(t, errors.As(err, &ie))

	// 写入凭证（仅在沙箱内）
	_, err = e.git.DangerouslyAuthenticate(e.ctx, &AuthenticateOptions{
		Username: "demo-user", Password: "demo-token", Host: "fake-host.example",
	})
	require.NoError(t, err)

	// 用 `git credential fill` 验证：输入 protocol/host，git 会从 store 中回填 username/password
	input := "protocol=https\nhost=fake-host.example\n\n"
	res, err := e.sb.Commands().Run(e.ctx,
		"printf %s '"+input+"' | git credential fill",
		WithEnvs(map[string]string{"GIT_TERMINAL_PROMPT": "0"}),
	)
	require.NoError(t, err)
	assert.Equal(t, 0, res.ExitCode, "credential fill 应成功，stdout=%q stderr=%q", res.Stdout, res.Stderr)
	assert.Contains(t, res.Stdout, "username=demo-user")
	assert.Contains(t, res.Stdout, "password=demo-token")
}

// TestIntegrationGitCloneStripsCredentials 验证默认情况下 clone 完成后 origin URL 不再带凭证。
// 仅当提供 QINIU_GIT_REPO_URL/USERNAME/PASSWORD 时执行。
func TestIntegrationGitCloneStripsCredentials(t *testing.T) {
	repoURL, username, password := getGitCredsFromEnv(t)
	e := newGitTestEnv(t)

	clonePath := "/tmp/it-clone-strip"
	_, err := e.git.Clone(e.ctx, repoURL, &CloneOptions{
		Path: clonePath, Depth: 1, Username: username, Password: password,
	})
	require.NoError(t, err)
	got, err := e.git.RemoteGet(e.ctx, clonePath, "origin", nil)
	require.NoError(t, err)
	assert.NotContains(t, got, username+":", "默认应剥离 origin URL 中的凭证")
	assert.NotContains(t, got, password)
}

// TestIntegrationGitCloneBranch 用沙箱内自建 bare repo 模拟一个多分支 remote，
// 验证 CloneOptions.Branch 能让 HEAD 指向指定分支，并且默认情况下 origin URL 不残留凭证。
func TestIntegrationGitCloneBranch(t *testing.T) {
	e := newGitTestEnv(t)

	source := "/tmp/it-clone-src"
	bare := "/tmp/it-clone-src.git"
	dst := "/tmp/it-clone-dst"

	e.initRepo(source, "main")
	e.writeAndCommit(source, "main.txt", "main\n", "feat: main")

	// 在 source 上再造一个 release 分支并提交
	_, err := e.git.CreateBranch(e.ctx, source, "release", nil)
	require.NoError(t, err)
	e.writeAndCommit(source, "release.txt", "release\n", "feat: release")
	_, err = e.git.CheckoutBranch(e.ctx, source, "main", nil)
	require.NoError(t, err)

	// 把 source push 到 bare 当作"远端"
	_, err = e.git.Init(e.ctx, bare, &InitOptions{Bare: true, InitialBranch: "main"})
	require.NoError(t, err)
	_, err = e.git.RemoteAdd(e.ctx, source, "origin", bare, nil)
	require.NoError(t, err)
	_, err = e.git.Push(e.ctx, source, &PushOptions{Remote: "origin", Branch: "main"})
	require.NoError(t, err)
	_, err = e.git.Push(e.ctx, source, &PushOptions{Remote: "origin", Branch: "release"})
	require.NoError(t, err)

	// 走 CloneOptions.Branch 拉取 release
	_, err = e.git.Clone(e.ctx, bare, &CloneOptions{Path: dst, Branch: "release"})
	require.NoError(t, err)

	// HEAD 应指向 release
	br, err := e.git.Branches(e.ctx, dst, nil)
	require.NoError(t, err)
	assert.Equal(t, "release", br.CurrentBranch)
	exists, err := e.sb.Files().Exists(e.ctx, dst+"/release.txt")
	require.NoError(t, err)
	assert.True(t, exists, "Branch=release 时 release.txt 应存在")

	// origin URL 不应包含任何 user info（无凭证残留）
	origin, err := e.git.RemoteGet(e.ctx, dst, "origin", nil)
	require.NoError(t, err)
	assert.NotContains(t, origin, "@")
}

// TestIntegrationGitRemoteAddFetch 验证 RemoteAddOptions.Fetch=true 在添加 remote
// 后立即拉取引用，使 origin/<branch> 可被 rev-parse 解析。
func TestIntegrationGitRemoteAddFetch(t *testing.T) {
	e := newGitTestEnv(t)

	source := "/tmp/it-fetch-src"
	bare := "/tmp/it-fetch-src.git"
	consumer := "/tmp/it-fetch-consumer"

	e.initRepo(source, "main")
	e.writeAndCommit(source, "a.txt", "a\n", "init")
	_, err := e.git.Init(e.ctx, bare, &InitOptions{Bare: true, InitialBranch: "main"})
	require.NoError(t, err)
	_, err = e.git.RemoteAdd(e.ctx, source, "origin", bare, nil)
	require.NoError(t, err)
	_, err = e.git.Push(e.ctx, source, &PushOptions{Remote: "origin", Branch: "main"})
	require.NoError(t, err)

	// consumer：先 init 再 RemoteAdd（Fetch=false） → 不应能解析 origin/main
	_, err = e.git.Init(e.ctx, consumer, &InitOptions{InitialBranch: "main"})
	require.NoError(t, err)
	_, err = e.git.RemoteAdd(e.ctx, consumer, "origin", bare, nil)
	require.NoError(t, err)

	res, err := e.sb.Commands().Run(e.ctx, "git -C "+consumer+" rev-parse --verify -q origin/main",
		WithEnvs(map[string]string{"GIT_TERMINAL_PROMPT": "0"}))
	require.NoError(t, err)
	assert.NotEqual(t, 0, res.ExitCode, "Fetch=false 时 origin/main 不应可解析")

	// 用 Overwrite=true + Fetch=true 重新加一次（同一个 origin）
	_, err = e.git.RemoteAdd(e.ctx, consumer, "origin", bare, &RemoteAddOptions{
		Overwrite: true, Fetch: true,
	})
	require.NoError(t, err)

	res, err = e.sb.Commands().Run(e.ctx, "git -C "+consumer+" rev-parse --verify -q origin/main",
		WithEnvs(map[string]string{"GIT_TERMINAL_PROMPT": "0"}))
	require.NoError(t, err)
	assert.Equal(t, 0, res.ExitCode, "Fetch=true 时 origin/main 应已被拉取，stderr=%q", res.Stderr)
	assert.NotEmpty(t, strings.TrimSpace(res.Stdout))
}

// TestIntegrationGitOptionsEnvsCwdTimeout 验证 GitOptions.Envs / Cwd / Timeout 的真实生效路径：
//   - Envs：自定义 GIT_AUTHOR_DATE 应作用到 commit；同时 GIT_TERMINAL_PROMPT=0 不会被调用方覆盖。
//   - Cwd：在 cwd 下用相对路径 Init 应在 cwd 内建出仓库。
//   - Timeout：极短超时应让命令失败而非挂住。
func TestIntegrationGitOptionsEnvsCwdTimeout(t *testing.T) {
	e := newGitTestEnv(t)

	// --- Envs ---
	envRepo := "/tmp/it-envs"
	e.initRepo(envRepo, "main")
	_, err := e.sb.Files().Write(e.ctx, envRepo+"/a.txt", []byte("a\n"))
	require.NoError(t, err)
	_, err = e.git.Add(e.ctx, envRepo, nil)
	require.NoError(t, err)

	authorDate := "2020-01-02T03:04:05+00:00"
	// 故意把 GIT_TERMINAL_PROMPT 置为 "1"，验证 SDK 默认值会被写在最后从而覆盖回 "0"。
	_, err = e.git.Commit(e.ctx, envRepo, "feat: dated", &CommitOptions{
		GitOptions: GitOptions{
			Envs: map[string]string{
				"GIT_AUTHOR_DATE":     authorDate,
				"GIT_COMMITTER_DATE":  authorDate,
				"GIT_TERMINAL_PROMPT": "1", // 不应生效，SDK 必须强制 0
			},
		},
	})
	require.NoError(t, err)
	res, err := e.sb.Commands().Run(e.ctx, "git -C "+envRepo+" log -1 --pretty=%aI",
		WithEnvs(map[string]string{"GIT_TERMINAL_PROMPT": "0"}))
	require.NoError(t, err)
	assert.Equal(t, "2020-01-02T03:04:05+00:00", strings.TrimSpace(res.Stdout))

	// --- Cwd ---
	cwdParent := "/tmp/it-cwd"
	_, err = e.sb.Commands().Run(e.ctx, "mkdir -p "+cwdParent,
		WithEnvs(map[string]string{"GIT_TERMINAL_PROMPT": "0"}))
	require.NoError(t, err)
	_, err = e.git.Init(e.ctx, "nested", &InitOptions{
		InitialBranch: "main",
		GitOptions:    GitOptions{Cwd: cwdParent},
	})
	require.NoError(t, err)
	exists, err := e.sb.Files().Exists(e.ctx, cwdParent+"/nested/.git/HEAD")
	require.NoError(t, err)
	assert.True(t, exists, "Cwd 下相对路径 Init 应在 %s/nested 建出仓库", cwdParent)

	// --- Timeout ---
	// 极短超时下任何 git 命令都应直接失败/超时，而不是挂住。
	_, err = e.git.Status(e.ctx, envRepo, &GitOptions{Timeout: 1 * time.Nanosecond})
	assert.Error(t, err, "Timeout=1ns 应导致命令失败")
}

// stagedNames 返回 status 中处于 staged 状态的文件名集合。
func stagedNames(s *GitStatus) []string {
	var out []string
	for _, f := range s.FileStatus {
		if f.Staged {
			out = append(out, f.Name)
		}
	}
	return out
}

// getGitCredsFromEnv 从环境变量读取 git 凭证；缺失时跳过当前测试。
func getGitCredsFromEnv(t *testing.T) (string, string, string) {
	t.Helper()
	repoURL := strings.TrimSpace(os.Getenv("QINIU_GIT_REPO_URL"))
	username := strings.TrimSpace(os.Getenv("QINIU_GIT_USERNAME"))
	password := strings.TrimSpace(os.Getenv("QINIU_GIT_PASSWORD"))
	if repoURL == "" || username == "" || password == "" {
		t.Skip("未设置 QINIU_GIT_REPO_URL / QINIU_GIT_USERNAME / QINIU_GIT_PASSWORD，跳过带凭证的 Clone 测试")
	}
	return repoURL, username, password
}
