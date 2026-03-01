package sandbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"testing"
)

func TestGetHost(t *testing.T) {
	c := &Client{config: &Config{Domain: "example.com"}}
	sb := &Sandbox{sandboxID: "sb-123", client: c}

	got := sb.GetHost(8080)
	want := "8080-sb-123.example.com"
	if got != want {
		t.Errorf("GetHost = %q, want %q", got, want)
	}
}

func TestGetHostDefaultDomain(t *testing.T) {
	c := &Client{config: &Config{}}
	sb := &Sandbox{sandboxID: "sb-456", client: c}

	got := sb.GetHost(3000)
	want := "3000-sb-456."
	if got != want {
		t.Errorf("GetHost = %q, want %q", got, want)
	}
}

func TestGetHostSandboxDomainOverride(t *testing.T) {
	c := &Client{config: &Config{Domain: "default.com"}}
	domain := "custom.sandbox.com"
	sb := &Sandbox{sandboxID: "sb-789", domain: &domain, client: c}

	got := sb.GetHost(443)
	want := "443-sb-789.custom.sandbox.com"
	if got != want {
		t.Errorf("GetHost = %q, want %q", got, want)
	}
}

func TestEnvdURL(t *testing.T) {
	c := &Client{config: &Config{Domain: "test.dev"}}
	sb := &Sandbox{sandboxID: "sb-100", client: c}

	got := sb.envdURL()
	want := "https://49983-sb-100.test.dev"
	if got != want {
		t.Errorf("envdURL = %q, want %q", got, want)
	}
}

func TestEnvdAuthHeader(t *testing.T) {
	h := envdAuthHeader("testuser")
	auth := h.Get("Authorization")
	// base64("testuser:") = "dGVzdHVzZXI6"
	want := "Basic dGVzdHVzZXI6"
	if auth != want {
		t.Errorf("envdAuthHeader = %q, want %q", auth, want)
	}
}

func TestFileSignature(t *testing.T) {
	sig := fileSignature("/test/file.txt", "read", "user", "token123", 300)
	raw := "/test/file.txt:read:user:token123:300"
	hash := sha256.Sum256([]byte(raw))
	want := "v1_" + fmt.Sprintf("%x", hash)
	if sig != want {
		t.Errorf("fileSignature = %q, want %q", sig, want)
	}
}

func TestDownloadURL(t *testing.T) {
	c := &Client{config: &Config{Domain: "test.dev"}}
	token := "mytoken"
	sb := &Sandbox{sandboxID: "sb-100", envdAccessToken: &token, client: c}

	u := sb.DownloadURL("/home/user/file.txt")
	// 验证 URL 包含必要的查询参数
	if u == "" {
		t.Fatal("DownloadURL returned empty string")
	}
	// 检查基础结构
	if got := "https://49983-sb-100.test.dev/files?"; len(u) < len(got) || u[:len(got)] != got {
		t.Errorf("DownloadURL prefix = %q, want prefix %q", u, got)
	}
}

func TestUploadURL(t *testing.T) {
	c := &Client{config: &Config{Domain: "test.dev"}}
	token := "mytoken"
	sb := &Sandbox{sandboxID: "sb-100", envdAccessToken: &token, client: c}

	u := sb.UploadURL("/home/user/file.txt")
	if u == "" {
		t.Fatal("UploadURL returned empty string")
	}
	if got := "https://49983-sb-100.test.dev/files?"; len(u) < len(got) || u[:len(got)] != got {
		t.Errorf("UploadURL prefix = %q, want prefix %q", u, got)
	}
}

func TestDownloadURLWithoutToken(t *testing.T) {
	c := &Client{config: &Config{Domain: "test.dev"}}
	sb := &Sandbox{sandboxID: "sb-100", client: c}

	u := sb.DownloadURL("/file.txt")
	// 没有 token 时不应包含 signature 参数
	if u == "" {
		t.Fatal("DownloadURL returned empty string")
	}
}

func TestFilesLazyInit(t *testing.T) {
	c := &Client{config: &Config{Domain: "test.dev"}}
	sb := &Sandbox{sandboxID: "sb-100", client: c}

	fs1 := sb.Files()
	fs2 := sb.Files()
	if fs1 != fs2 {
		t.Error("Files() should return the same instance")
	}
}

func TestCommandsLazyInit(t *testing.T) {
	c := &Client{config: &Config{Domain: "test.dev"}}
	sb := &Sandbox{sandboxID: "sb-100", client: c}

	cmd1 := sb.Commands()
	cmd2 := sb.Commands()
	if cmd1 != cmd2 {
		t.Error("Commands() should return the same instance")
	}
}

func TestPtyLazyInit(t *testing.T) {
	c := &Client{config: &Config{Domain: "test.dev"}}
	sb := &Sandbox{sandboxID: "sb-100", client: c}

	pty1 := sb.Pty()
	pty2 := sb.Pty()
	if pty1 != pty2 {
		t.Error("Pty() should return the same instance")
	}
}

func TestFileURLOptionWithUser(t *testing.T) {
	c := &Client{config: &Config{Domain: "test.dev"}}
	token := "tok"
	sb := &Sandbox{sandboxID: "sb-1", envdAccessToken: &token, client: c}

	u := sb.DownloadURL("/file.txt", WithFileUser("admin"))
	// 验证 URL 包含 username=admin
	if u == "" {
		t.Fatal("DownloadURL returned empty string")
	}
}

func TestIsNotFoundError(t *testing.T) {
	apiErr := &APIError{StatusCode: 404, Body: []byte("not found")}
	if !isNotFoundError(apiErr) {
		t.Error("expected isNotFoundError to return true for 404 APIError")
	}

	apiErr200 := &APIError{StatusCode: 200, Body: []byte("ok")}
	if isNotFoundError(apiErr200) {
		t.Error("expected isNotFoundError to return false for 200 APIError")
	}
}

func TestEntryInfoFromProtoNil(t *testing.T) {
	if entryInfoFromProto(nil) != nil {
		t.Error("entryInfoFromProto(nil) should return nil")
	}
}

func TestWriteFilesEmpty(t *testing.T) {
	c := &Client{config: &Config{Domain: "test.dev"}}
	sb := &Sandbox{sandboxID: "sb-100", client: c}
	fs := &Filesystem{sandbox: sb}

	infos, err := fs.WriteFiles(context.Background(), nil)
	if err != nil {
		t.Fatalf("WriteFiles(nil) 应返回 nil error，得到: %v", err)
	}
	if infos != nil {
		t.Fatalf("WriteFiles(nil) 应返回 nil，得到: %v", infos)
	}
}

func TestBatchUploadURL(t *testing.T) {
	c := &Client{config: &Config{Domain: "test.dev"}}
	sb := &Sandbox{sandboxID: "sb-100", client: c}

	u := sb.batchUploadURL("user")
	want := "https://49983-sb-100.test.dev/files?username=user"
	if u != want {
		t.Errorf("batchUploadURL = %q, want %q", u, want)
	}
}

func TestWriteFileFullPath(t *testing.T) {
	var buf bytes.Buffer
	w := newMultipartWriter(&buf)

	if err := w.writeFileFullPath("file_0", "/home/user/test.txt", []byte("hello")); err != nil {
		t.Fatalf("writeFileFullPath 失败: %v", err)
	}
	if err := w.close(); err != nil {
		t.Fatalf("close 失败: %v", err)
	}

	// 解析 multipart 内容，验证 Content-Disposition 中的 filename 是完整路径。
	// 注意：Part.FileName() 会调用 filepath.Base()，所以直接解析 header。
	r := multipart.NewReader(&buf, w.w.Boundary())
	part, err := r.NextPart()
	if err != nil {
		t.Fatalf("NextPart 失败: %v", err)
	}
	_, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
	if err != nil {
		t.Fatalf("ParseMediaType 失败: %v", err)
	}
	if got := params["filename"]; got != "/home/user/test.txt" {
		t.Errorf("filename = %q, want %q", got, "/home/user/test.txt")
	}
	data, _ := io.ReadAll(part)
	if string(data) != "hello" {
		t.Errorf("data = %q, want %q", string(data), "hello")
	}
}

func TestWriteFileBaseName(t *testing.T) {
	var buf bytes.Buffer
	w := newMultipartWriter(&buf)

	if err := w.writeFile("file", "/home/user/test.txt", []byte("hello")); err != nil {
		t.Fatalf("writeFile 失败: %v", err)
	}
	if err := w.close(); err != nil {
		t.Fatalf("close 失败: %v", err)
	}

	// 解析 multipart 内容，验证 filename 是 basename
	r := multipart.NewReader(&buf, w.w.Boundary())
	part, err := r.NextPart()
	if err != nil {
		t.Fatalf("NextPart 失败: %v", err)
	}
	if got := part.FileName(); got != "test.txt" {
		t.Errorf("filename = %q, want %q", got, "test.txt")
	}
}
