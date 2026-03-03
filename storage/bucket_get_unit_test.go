//go:build unit
// +build unit

package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/qiniu/go-sdk/v7/auth"
)

// getTestGetServers 创建 mock 下载服务器和 mock UC 服务器，
// 并将 UC hosts 指向 mock UC 服务器，返回 cleanup 函数。
//
// mock UC 在 /v4/query 响应中将 io_src.domains 设置为 mock 下载服务器的 host:port，
// 让 defaultSrcURLsProvider 查询后直接命中 mock 下载服务器，无需真实凭据。
func getTestGetServers(
	t *testing.T,
	key, content string,
	withContentLength bool,
	extraHeaders http.Header,
) (dlServer *httptest.Server, cleanup func()) {
	t.Helper()

	// --- mock 下载服务器 ---
	dlMux := http.NewServeMux()
	dlMux.HandleFunc("/"+key, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("ETag", `"testetag"`)
		w.Header().Set("Last-Modified", time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC).UTC().Format(http.TimeFormat))
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Add("X-Reqid", "fakereqid")
		for k, vs := range extraHeaders {
			for _, v := range vs {
				w.Header().Set(k, v)
			}
		}
		if withContentLength {
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
		}
		switch r.Method {
		case http.MethodHead:
		case http.MethodGet:
			w.Write([]byte(content))
		default:
			t.Errorf("unexpected method: %s", r.Method)
		}
	})
	dlServer = httptest.NewServer(dlMux)

	// 从 "http://127.0.0.1:PORT" 提取 "127.0.0.1:PORT"，
	// 写入 UC 响应的 io_src.domains（无 scheme），
	// 由 staticDomainBasedURLsProvider 根据 UseInsecureProtocol 补 http://。
	dlHost := strings.TrimPrefix(dlServer.URL, "http://")

	// --- mock UC 服务器，响应 /v4/query ---
	ucMux := http.NewServeMux()
	ucMux.HandleFunc("/v4/query", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Add("X-Reqid", "fakereqid")
		// 将 io_src.domains 指向 mock 下载服务器，
		// 使 defaultSrcURLsProvider 拿到 mock 下载服务器地址。
		fmt.Fprintf(w, `{
			"hosts": [{
				"region": "test",
				"ttl": 86400,
				"io_src": {"domains": [%q]},
				"io":     {"domains": [%q]},
				"up":     {"domains": ["upload.fake.com"]},
				"uc":     {"domains": ["uc.fake.com"]},
				"rs":     {"domains": ["rs.fake.com"]},
				"rsf":    {"domains": ["rsf.fake.com"]},
				"api":    {"domains": ["api.fake.com"]}
			}]
		}`, dlHost, dlHost)
	})
	ucServer := httptest.NewServer(ucMux)

	// 保存原始 UC hosts，覆盖为 mock UC，测试结束后还原。
	origUCHosts := getUCHosts()
	SetUcHosts(ucServer.URL)

	cleanup = func() {
		SetUcHosts(origUCHosts...)
		ucServer.Close()
		dlServer.Close()
	}
	return dlServer, cleanup
}

// newGetTestBucketManager 创建使用假凭据的 BucketManager，仅供单元测试使用。
func newGetTestBucketManager() *BucketManager {
	return NewBucketManager(auth.New("fake-ak", "fake-sk"), &Config{UseHTTPS: false})
}

// TestGet_ContentLengthSetFromHeaderBeforeBodyRead 验证关键修复：
// ContentLength 在 Get() 返回时就已从 HEAD 响应头解析完毕，调用方无需等待 Body
// 读取完成即可获得准确的文件大小；配合 -race 可验证不存在数据竞争。
func TestGet_ContentLengthSetFromHeaderBeforeBodyRead(t *testing.T) {
	const content = "hello world"
	_, cleanup := getTestGetServers(t, "testfile", content, true, nil)
	defer cleanup()

	resp, err := newGetTestBucketManager().Get("fakebucket", "testfile", &GetObjectInput{})
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	defer resp.Close()

	// 在读取 Body 之前断言：ContentLength 来自响应头，不是下载结束后赋值。
	if want := int64(len(content)); resp.ContentLength != want {
		t.Errorf("ContentLength before body read = %d, want %d", resp.ContentLength, want)
	}

	// Body 内容仍然正确。
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll error: %v", err)
	}
	if string(got) != content {
		t.Errorf("Body = %q, want %q", string(got), content)
	}
}

// TestGet_ContentLengthNegativeOneWhenHeaderAbsent 验证当服务端未返回 Content-Length 时，
// ContentLength 被设置为 -1，与 GetObjectOutput 注释约定（-1 表示未知）一致。
func TestGet_ContentLengthNegativeOneWhenHeaderAbsent(t *testing.T) {
	const content = "hello world"
	// withContentLength=false：HEAD 响应头不包含 Content-Length。
	_, cleanup := getTestGetServers(t, "testfile", content, false, nil)
	defer cleanup()

	resp, err := newGetTestBucketManager().Get("fakebucket", "testfile", &GetObjectInput{})
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	defer resp.Close()

	if resp.ContentLength != -1 {
		t.Errorf("ContentLength = %d, want -1 when Content-Length header is absent", resp.ContentLength)
	}
}

// TestGet_AllHeaderFieldsPopulatedBeforeReturn 验证 Get() 返回时所有响应头字段
// （ContentType、ETag、LastModified、ContentLength、x-qn-meta-* Metadata）
// 均已完整填充，不依赖 Body 读取完成。
func TestGet_AllHeaderFieldsPopulatedBeforeReturn(t *testing.T) {
	const content = "response body"
	wantLastModified := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	extraHeaders := http.Header{
		"X-Qn-Meta-Sync-Index": []string{"99"},
		"X-Qn-Meta-Author":     []string{"alice"},
	}
	_, cleanup := getTestGetServers(t, "testfile", content, true, extraHeaders)
	defer cleanup()

	resp, err := newGetTestBucketManager().Get("fakebucket", "testfile", &GetObjectInput{})
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	defer resp.Close()

	// 以下全部断言在读取 Body 之前执行，确认字段在 Get() 返回时就已就绪。
	if resp.ContentType != "text/plain" {
		t.Errorf("ContentType = %q, want %q", resp.ContentType, "text/plain")
	}
	if resp.ETag != "testetag" {
		t.Errorf("ETag = %q, want %q (quotes should be stripped)", resp.ETag, "testetag")
	}
	if !resp.LastModified.Equal(wantLastModified) {
		t.Errorf("LastModified = %v, want %v", resp.LastModified, wantLastModified)
	}
	if want := int64(len(content)); resp.ContentLength != want {
		t.Errorf("ContentLength = %d, want %d", resp.ContentLength, want)
	}
	// x-qn-meta-* 头以 Go canonical 形式（首字母大写）存储于 Metadata。
	for wantKey, wantVal := range map[string]string{
		"X-Qn-Meta-Sync-Index": "99",
		"X-Qn-Meta-Author":     "alice",
	} {
		if gotVal, ok := resp.Metadata[wantKey]; !ok {
			t.Errorf("Metadata missing key %q", wantKey)
		} else if gotVal != wantVal {
			t.Errorf("Metadata[%q] = %q, want %q", wantKey, gotVal, wantVal)
		}
	}
}

// TestGet_NoPanicOnConcurrentCalls 以高并发方式验证修复后的 Get() 不再触发
// "send on closed channel" panic；配合 -race 标志可同时检测数据竞争。
func TestGet_NoPanicOnConcurrentCalls(t *testing.T) {
	const content = "hello world"
	_, cleanup := getTestGetServers(t, "testfile", content, true, nil)
	defer cleanup()

	bm := newGetTestBucketManager()

	const (
		concurrency = 32
		loops       = 64
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		wg      sync.WaitGroup
		panicCh = make(chan interface{}, 1)
	)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < loops; j++ {
				select {
				case <-ctx.Done():
					return
				default:
				}
				func() {
					defer func() {
						if r := recover(); r != nil {
							if strings.Contains(fmt.Sprint(r), "send on closed channel") {
								select {
								case panicCh <- r:
								default:
								}
								cancel()
							} else {
								panic(r)
							}
						}
					}()
					resp, err := bm.Get("fakebucket", "testfile", &GetObjectInput{Context: ctx})
					if resp != nil {
						resp.Close()
					}
					_ = err
				}()
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case p := <-panicCh:
		t.Fatalf("Get() triggered 'send on closed channel' panic: %v", p)
	case <-done:
		// 全部并发请求正常结束，无 panic，测试通过。
	case <-time.After(60 * time.Second):
		t.Fatal("timeout: concurrent Get() calls did not complete in time")
	}
}
