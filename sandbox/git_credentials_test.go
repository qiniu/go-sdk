//go:build unit

package sandbox

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/qiniu/go-sdk/v7/sandbox/internal/envdapi/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitTemporaryCredentialsDisableInheritedHelpers(t *testing.T) {
	for _, operation := range []string{"clone", "pull", "push"} {
		for _, credentials := range []bool{false, true} {
			name := operation + "/without-credentials"
			if credentials {
				name = operation + "/temporary-credentials"
			}
			t.Run(name, func(t *testing.T) {
				var mu sync.Mutex
				var networkCommand string
				commands, server := newTestCommands(&testProcessHandler{
					startFn: func(_ context.Context, req *connect.Request[process.StartRequest], stream *connect.ServerStream[process.StartResponse]) error {
						command := req.Msg.Process.Args[2]
						if strings.Contains(command, "'"+operation+"'") {
							mu.Lock()
							networkCommand = command
							mu.Unlock()
						}
						if strings.Contains(command, "'remote' 'get-url'") {
							if err := stream.Send(&process.StartResponse{Event: &process.ProcessEvent{
								Event: &process.ProcessEvent_Data{Data: &process.ProcessEvent_DataEvent{
									Output: &process.ProcessEvent_DataEvent_Stdout{Stdout: []byte("https://example.com/repo.git\n")},
								}},
							}}); err != nil {
								return err
							}
						}
						return stream.Send(&process.StartResponse{Event: &process.ProcessEvent{
							Event: &process.ProcessEvent_End{End: &process.ProcessEvent_EndEvent{Exited: true}},
						}})
					},
				})
				defer server.Close()
				git := newGit(commands)
				var username, password string
				if credentials {
					username, password = "temporary-user", "temporary-token"
				}
				var err error
				switch operation {
				case "clone":
					_, err = git.Clone(context.Background(), "https://example.com/repo.git", &CloneOptions{Path: "/repo", Username: username, Password: password})
				case "pull":
					_, err = git.Pull(context.Background(), "/repo", &PullOptions{Remote: "origin", Branch: "main", Username: username, Password: password})
				case "push":
					_, err = git.Push(context.Background(), "/repo", &PushOptions{Remote: "origin", Branch: "main", Username: username, Password: password})
				}
				require.NoError(t, err)
				mu.Lock()
				defer mu.Unlock()
				require.NotEmpty(t, networkCommand)
				if credentials {
					assert.Contains(t, networkCommand, "'-c' 'credential.helper=' '"+operation+"'", "临时凭据命令应重置继承的 helper 列表")
				} else {
					assert.NotContains(t, networkCommand, "credential.helper", "无显式凭据时保留用户已有认证配置")
				}
			})
		}
	}
}

func TestCloneCredentialHelperIsolationMatchesPersistenceOption(t *testing.T) {
	for _, persistent := range []bool{false, true} {
		plan, err := buildClonePlan("https://user:token@example.com/repo.git", "/repo", "", 0, "", "", persistent)
		require.NoError(t, err)
		command := buildGitCommand(plan.args, "")
		if persistent {
			assert.NotContains(t, command, "credential.helper")
		} else {
			assert.Contains(t, command, "'-c' 'credential.helper=' 'clone'")
		}
	}
	for _, rawURL := range []string{"https://user@example.com/repo.git", "https://user:@example.com/repo.git"} {
		plan, err := buildClonePlan(rawURL, "/repo", "", 0, "", "", false)
		require.NoError(t, err)
		assert.NotContains(t, buildGitCommand(plan.args, ""), "credential.helper", "只有用户名时仍允许 helper 提供密码")
	}
}

// 使用真实 git 和本地 HTTPS bare 仓库，确认成功认证后的 credential approve 不会落盘。
func TestGitCloneDoesNotPersistTemporaryCredentialsInInheritedStore(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("本地没有 git")
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("本地没有 sh")
	}
	for _, persistent := range []bool{false, true} {
		name := "temporary"
		if persistent {
			name = "explicit-persistence"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			credentialFile := filepath.Join(root, "credentials")
			configFile := filepath.Join(root, "gitconfig")
			env := append(os.Environ(),
				"GIT_CONFIG_GLOBAL="+configFile, "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0",
				// 仅本地 httptest TLS 证书未受系统信任。
				"GIT_SSL_NO_VERIFY=1",
			)
			configure := exec.Command("git", "config", "--file", configFile, "credential.helper", "store --file="+shellEscape(filepath.ToSlash(credentialFile)))
			configure.Env = env
			output, err := configure.CombinedOutput()
			require.NoError(t, err, "%s", output)
			originalConfig, err := os.ReadFile(configFile)
			require.NoError(t, err)
			bare := filepath.Join(root, "remote.git")
			for _, args := range [][]string{{"init", "--bare", bare}, {"--git-dir=" + bare, "update-server-info"}} {
				cmd := exec.Command("git", args...)
				cmd.Env = env
				output, err := cmd.CombinedOutput()
				require.NoError(t, err, "%s", output)
			}
			files := http.FileServer(http.Dir(root))
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				username, password, ok := r.BasicAuth()
				if !ok || username != "fixture-user" || password != "fixture-token" {
					w.Header().Set("WWW-Authenticate", `Basic realm="fixture"`)
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				files.ServeHTTP(w, r)
			}))
			defer server.Close()
			commands, rpcServer := newTestCommands(&testProcessHandler{
				startFn: func(ctx context.Context, req *connect.Request[process.StartRequest], stream *connect.ServerStream[process.StartResponse]) error {
					cmd := exec.CommandContext(ctx, "sh", "-c", req.Msg.Process.Args[2])
					cmd.Dir = root
					cmd.Env = append([]string(nil), env...)
					for key, value := range req.Msg.Process.Envs {
						cmd.Env = append(cmd.Env, key+"="+value)
					}
					var stdout, stderr bytes.Buffer
					cmd.Stdout, cmd.Stderr = &stdout, &stderr
					exitCode := int32(0)
					if err := cmd.Run(); err != nil {
						var exitErr *exec.ExitError
						if !errors.As(err, &exitErr) {
							return connect.NewError(connect.CodeInternal, err)
						}
						exitCode = int32(exitErr.ExitCode())
					}
					for _, event := range []*process.ProcessEvent{
						{Event: &process.ProcessEvent_Data{Data: &process.ProcessEvent_DataEvent{Output: &process.ProcessEvent_DataEvent_Stdout{Stdout: stdout.Bytes()}}}},
						{Event: &process.ProcessEvent_Data{Data: &process.ProcessEvent_DataEvent{Output: &process.ProcessEvent_DataEvent_Stderr{Stderr: stderr.Bytes()}}}},
						{Event: &process.ProcessEvent_End{End: &process.ProcessEvent_EndEvent{Exited: true, ExitCode: exitCode}}},
					} {
						if err := stream.Send(&process.StartResponse{Event: event}); err != nil {
							return err
						}
					}
					return nil
				},
			})
			defer rpcServer.Close()
			git := newGit(commands)
			clonePath := filepath.Join(root, "clone")
			_, err = git.Clone(context.Background(), server.URL+"/remote.git", &CloneOptions{
				Path: clonePath, Username: "fixture-user", Password: "fixture-token", DangerouslyStoreCredentials: persistent,
			})
			require.NoError(t, err)
			stored, err := os.ReadFile(credentialFile)
			if persistent {
				require.NoError(t, err)
				assert.Contains(t, string(stored), "fixture-user:fixture-token@")
			} else {
				assert.True(t, errors.Is(err, os.ErrNotExist) || len(stored) == 0, "临时凭据不应写入 store: %v", err)
				assert.NotContains(t, string(stored), "fixture-token")
			}
			config, err := os.ReadFile(filepath.Join(clonePath, ".git", "config"))
			require.NoError(t, err)
			assert.Equal(t, persistent, strings.Contains(string(config), "fixture-token"))
			unchanged, err := os.ReadFile(configFile)
			require.NoError(t, err)
			assert.Equal(t, originalConfig, unchanged)
		})
	}
}
