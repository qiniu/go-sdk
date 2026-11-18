//go:build unit

package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strconv"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/qiniu/go-sdk/v7/reqid"
	"github.com/qiniu/go-sdk/v7/sandbox/internal/envdapi/filesystem"
	"github.com/qiniu/go-sdk/v7/sandbox/internal/envdapi/filesystem/filesystemconnect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// envdFileAuthError 独立实现 envd internal/api/auth.go 的文件鉴权契约，
// 使用 SHA256 的无填充标准 base64 和绝对 Unix 秒，不能调用 SDK fileSignature。
func envdFileAuthError(r *http.Request, token string) error {
	if token == "" {
		return nil
	}
	if header := r.Header.Get("X-Access-Token"); header != "" {
		if header != token {
			return fmt.Errorf("access token present in header but does not match")
		}
		return nil
	}
	query := r.URL.Query()
	if !query.Has("signature") {
		return fmt.Errorf("missing signature query parameter")
	}
	operation := "read"
	if r.Method == http.MethodPost {
		operation = "write"
	}
	raw := fmt.Sprintf("%s:%s:%s:%s", query.Get("path"), operation, query.Get("username"), token)
	if query.Has("signature_expiration") {
		raw += ":" + query.Get("signature_expiration")
	}
	digest := sha256.Sum256([]byte(raw))
	expected := "v1_" + base64.RawStdEncoding.EncodeToString(digest[:])
	if query.Get("signature") != expected {
		return fmt.Errorf("invalid signature")
	}
	if query.Has("signature_expiration") {
		expires, err := strconv.ParseInt(query.Get("signature_expiration"), 10, 64)
		if err != nil {
			return err
		}
		if expires < time.Now().Unix() {
			return fmt.Errorf("signature is already expired")
		}
	}
	return nil
}

func TestFileURLsMatchEnvdSigning(t *testing.T) {
	domain, token := "test.dev", "test-envd-token"
	sb := &Sandbox{sandboxID: "sb-test", domain: &domain, envdAccessToken: &token}
	for name, makeURL := range map[string]func(string, ...FileURLOption) string{
		"download": sb.DownloadURL,
		"upload":   sb.UploadURL,
	} {
		t.Run(name, func(t *testing.T) {
			method := http.MethodGet
			if name == "upload" {
				method = http.MethodPost
			}
			before := time.Now().Unix()
			rawURL := makeURL("/tmp/a b+?.txt", WithFileUser("root"))
			after := time.Now().Unix()
			req, err := http.NewRequest(method, rawURL, nil)
			require.NoError(t, err)
			assert.Equal(t, "/tmp/a b+?.txt", req.URL.Query().Get("path"))
			assert.Equal(t, "root", req.URL.Query().Get("username"))
			expires, err := strconv.ParseInt(req.URL.Query().Get("signature_expiration"), 10, 64)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, expires, before+300)
			assert.LessOrEqual(t, expires, after+300)
			require.NoError(t, envdFileAuthError(req, token))

			future := int(time.Now().Unix()) + 600
			req, err = http.NewRequest(method, makeURL("/tmp/file", WithSignatureExpiration(future)), nil)
			require.NoError(t, err)
			assert.Equal(t, strconv.Itoa(future), req.URL.Query().Get("signature_expiration"))
			require.NoError(t, envdFileAuthError(req, token))
			query := req.URL.Query()
			query.Set("path", "/tmp/tampered")
			req.URL.RawQuery = query.Encode()
			assert.EqualError(t, envdFileAuthError(req, token), "invalid signature")

			past := int(time.Now().Unix()) - 1
			req, err = http.NewRequest(method, makeURL("/tmp/file", WithSignatureExpiration(past)), nil)
			require.NoError(t, err)
			assert.EqualError(t, envdFileAuthError(req, token), "signature is already expired")
		})
	}
}

func TestFileURLsWithoutEnvdToken(t *testing.T) {
	domain, emptyToken := "test.dev", ""
	for _, token := range []*string{nil, &emptyToken} {
		sb := &Sandbox{sandboxID: "sb-test", domain: &domain, envdAccessToken: token}
		for _, rawURL := range []string{sb.DownloadURL("/tmp/file"), sb.UploadURL("/tmp/file")} {
			parsed, err := url.Parse(rawURL)
			require.NoError(t, err)
			assert.False(t, parsed.Query().Has("signature"))
			assert.False(t, parsed.Query().Has("signature_expiration"))
		}
	}
}

func TestFilesystemHTTPAuthentication(t *testing.T) {
	for _, token := range []string{"test-envd-token", ""} {
		name := "secured"
		if token == "" {
			name = "without-token"
		}
		t.Run(name, func(t *testing.T) {
			operations := map[string]func(*testing.T, context.Context, *Filesystem){
				"read": func(t *testing.T, ctx context.Context, fs *Filesystem) {
					content, err := fs.Read(ctx, "/tmp/file", WithUser("root"))
					require.NoError(t, err)
					assert.Equal(t, []byte("content"), content)
				},
				"read-text": func(t *testing.T, ctx context.Context, fs *Filesystem) {
					content, err := fs.ReadText(ctx, "/tmp/file", WithUser("root"))
					require.NoError(t, err)
					assert.Equal(t, "content", content)
				},
				"read-stream": func(t *testing.T, ctx context.Context, fs *Filesystem) {
					stream, err := fs.ReadStream(ctx, "/tmp/file", WithUser("root"))
					require.NoError(t, err)
					defer stream.Close()
					content, err := io.ReadAll(stream)
					require.NoError(t, err)
					assert.Equal(t, "content", string(content))
				},
				"write": func(t *testing.T, ctx context.Context, fs *Filesystem) {
					info, err := fs.Write(ctx, "/tmp/written", []byte("written"), WithUser("root"))
					require.NoError(t, err)
					assert.Equal(t, int64(7), info.Size)
				},
				"write-files-single": func(t *testing.T, ctx context.Context, fs *Filesystem) {
					infos, err := fs.WriteFiles(ctx, []WriteEntry{{Path: "/tmp/written", Data: []byte("written")}}, WithUser("root"))
					require.NoError(t, err)
					require.Len(t, infos, 1)
					assert.Equal(t, int64(7), infos[0].Size)
				},
				"write-files-multiple": func(t *testing.T, ctx context.Context, fs *Filesystem) {
					infos, err := fs.WriteFiles(ctx, []WriteEntry{
						{Path: "/tmp/written", Data: []byte("written")},
						{Path: "/tmp/second", Data: []byte("second")},
					}, WithUser("root"))
					require.NoError(t, err)
					require.Len(t, infos, 2)
					assert.Equal(t, int64(7), infos[0].Size)
					assert.Equal(t, int64(6), infos[1].Size)
				},
			}
			for name, run := range operations {
				t.Run(name, func(t *testing.T) {
					var mu sync.Mutex
					files := map[string][]byte{"/tmp/file": []byte("content")}
					mux := http.NewServeMux()
					mux.HandleFunc("/files", func(w http.ResponseWriter, r *http.Request) {
						assert.Equal(t, token, r.Header.Get("X-Access-Token"))
						assert.Equal(t, "root", r.URL.Query().Get("username"))
						assert.Equal(t, "file-auth-reqid", r.Header.Get("X-Reqid"))
						assert.False(t, r.URL.Query().Has("signature"), "SDK 自有请求使用 header 鉴权")
						assert.False(t, r.URL.Query().Has("signature_expiration"))
						if err := envdFileAuthError(r, token); err != nil {
							http.Error(w, err.Error(), http.StatusUnauthorized)
							return
						}
						mu.Lock()
						defer mu.Unlock()
						if r.Method == http.MethodGet {
							_, err := w.Write(files[r.URL.Query().Get("path")])
							assert.NoError(t, err)
							return
						}
						reader, err := r.MultipartReader()
						if !assert.NoError(t, err) {
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						for {
							part, err := reader.NextPart()
							if err == io.EOF {
								break
							}
							if !assert.NoError(t, err) {
								w.WriteHeader(http.StatusBadRequest)
								return
							}
							_, parameters, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
							assert.NoError(t, err)
							filePath := r.URL.Query().Get("path")
							if filePath == "" {
								filePath = parameters["filename"]
							}
							files[filePath], err = io.ReadAll(part)
							assert.NoError(t, err)
						}
					})
					rpcPath, rpcHandler := filesystemconnect.NewFilesystemHandler(&testFilesystemHandler{
						statFn: func(_ context.Context, req *connect.Request[filesystem.StatRequest]) (*connect.Response[filesystem.StatResponse], error) {
							assert.Equal(t, token, req.Header().Get("X-Access-Token"))
							assert.Equal(t, envdBasicAuth("root"), req.Header().Get("Authorization"))
							mu.Lock()
							defer mu.Unlock()
							content, exists := files[req.Msg.Path]
							if !exists {
								return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("file missing"))
							}
							return connect.NewResponse(&filesystem.StatResponse{Entry: &filesystem.EntryInfo{
								Path: req.Msg.Path, Name: path.Base(req.Msg.Path), Size: int64(len(content)),
							}}), nil
						},
					})
					mux.Handle(rpcPath, rpcHandler)
					server := httptest.NewServer(mux)
					defer server.Close()
					sb := newTestSandboxWithHTTP(server)
					if token != "" {
						sb.envdAccessToken = &token
					}
					sb.client.config.HTTPClient.Transport = &rewriteTransport{base: server.Client().Transport, baseURL: server.URL}
					fs := &Filesystem{sandbox: sb, rpc: filesystemconnect.NewFilesystemClient(server.Client(), server.URL)}
					ctx := reqid.WithReqid(context.Background(), "file-auth-reqid")
					run(t, ctx, fs)
				})
			}
		})
	}
}

// envdOriginTransport 仅将沙箱域名映射到本地 fixture；外部重定向按实际 URL 访问。
type envdOriginTransport struct {
	origin string
	target *url.URL
	base   http.RoundTripper
}

func (tr *envdOriginTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host != tr.origin {
		return tr.base.RoundTrip(req)
	}
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = tr.target.Scheme
	cloned.URL.Host = tr.target.Host
	return tr.base.RoundTrip(cloned)
}

func TestFilesystemHTTPRedirectDoesNotForwardEnvdToken(t *testing.T) {
	for _, sameOriginFirst := range []bool{false, true} {
		name := "cross-origin"
		if sameOriginFirst {
			name = "same-origin-then-cross-origin"
		}
		t.Run(name, func(t *testing.T) {
			token := "test-envd-token"
			external := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Empty(t, r.Header.Get("X-Access-Token"), "外部 URL 不应收到 envd token")
				assert.NotContains(t, r.Referer(), "signature", "外部 URL 不应通过 Referer 收到有效文件签名")
				fmt.Fprint(w, "external content")
			}))
			defer external.Close()
			envd := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, token, r.Header.Get("X-Access-Token"))
				if sameOriginFirst && r.URL.Path == "/files" {
					http.Redirect(w, r, "/redirected?"+r.URL.RawQuery, http.StatusFound)
					return
				}
				http.Redirect(w, r, external.URL, http.StatusFound)
			}))
			defer envd.Close()
			target, err := url.Parse(envd.URL)
			require.NoError(t, err)
			sb := newTestSandboxWithHTTP(envd)
			sb.envdAccessToken = &token
			client := sb.client.config.HTTPClient
			client.Transport = &envdOriginTransport{origin: sb.GetHost(envdPort), target: target, base: external.Client().Transport}
			redirects := 0
			client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
				redirects++
				return nil
			}
			content, err := (&Filesystem{sandbox: sb}).Read(context.Background(), "/tmp/file")
			require.NoError(t, err)
			assert.Equal(t, "external content", string(content))
			wantRedirects := 1
			if sameOriginFirst {
				wantRedirects++
			}
			assert.Equal(t, wantRedirects, redirects, "保留调用方的重定向策略，同源跳转仍应携带 token")

			// 同一个用户 HTTPClient 后续访问外部 URL 也不应被全局注入 token。
			response, err := client.Get(external.URL)
			require.NoError(t, err)
			response.Body.Close()
		})
	}
}

type legacyFileTransport struct {
	canceled chan *http.Request
}

func (tr *legacyFileTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	select {
	case canceled := <-tr.canceled:
		if canceled != req {
			return nil, fmt.Errorf("CancelRequest received a different request")
		}
		return nil, fmt.Errorf("legacy file request canceled")
	case <-time.After(time.Second):
		return nil, fmt.Errorf("CancelRequest was not called")
	}
}

func (tr *legacyFileTransport) CancelRequest(req *http.Request) {
	tr.canceled <- req
}

func TestFilesystemHTTPPreservesLegacyTransportCancellation(t *testing.T) {
	tr := &legacyFileTransport{canceled: make(chan *http.Request, 1)}
	token := "test-envd-token"
	sb := &Sandbox{
		envdAccessToken: &token,
		client: &Client{config: &Config{HTTPClient: &http.Client{
			Transport: tr,
			Timeout:   25 * time.Millisecond,
		}}},
	}
	fs := &Filesystem{sandbox: sb}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.test/files", nil)
	require.NoError(t, err)
	_, err = fs.doFileRequest(req)
	require.ErrorContains(t, err, "legacy file request canceled")
	assert.Empty(t, req.Header.Get("X-Access-Token"))
}
