package sandbox

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/qiniu/go-sdk/v7/sandbox/envdapi/filesystem"
	"github.com/qiniu/go-sdk/v7/sandbox/envdapi/filesystem/filesystemconnect"
	"github.com/qiniu/go-sdk/v7/sandbox/envdapi/process"
	"github.com/qiniu/go-sdk/v7/sandbox/envdapi/process/processconnect"
)

// ---------------------------------------------------------------------------
// testFilesystemHandler — mock ConnectRPC server for Filesystem service
// ---------------------------------------------------------------------------

type testFilesystemHandler struct {
	filesystemconnect.UnimplementedFilesystemHandler

	statFn     func(context.Context, *connect.Request[filesystem.StatRequest]) (*connect.Response[filesystem.StatResponse], error)
	makeDirFn  func(context.Context, *connect.Request[filesystem.MakeDirRequest]) (*connect.Response[filesystem.MakeDirResponse], error)
	moveFn     func(context.Context, *connect.Request[filesystem.MoveRequest]) (*connect.Response[filesystem.MoveResponse], error)
	listDirFn  func(context.Context, *connect.Request[filesystem.ListDirRequest]) (*connect.Response[filesystem.ListDirResponse], error)
	removeFn   func(context.Context, *connect.Request[filesystem.RemoveRequest]) (*connect.Response[filesystem.RemoveResponse], error)
	watchDirFn func(context.Context, *connect.Request[filesystem.WatchDirRequest], *connect.ServerStream[filesystem.WatchDirResponse]) error
}

func (h *testFilesystemHandler) Stat(ctx context.Context, req *connect.Request[filesystem.StatRequest]) (*connect.Response[filesystem.StatResponse], error) {
	if h.statFn != nil {
		return h.statFn(ctx, req)
	}
	return h.UnimplementedFilesystemHandler.Stat(ctx, req)
}

func (h *testFilesystemHandler) MakeDir(ctx context.Context, req *connect.Request[filesystem.MakeDirRequest]) (*connect.Response[filesystem.MakeDirResponse], error) {
	if h.makeDirFn != nil {
		return h.makeDirFn(ctx, req)
	}
	return h.UnimplementedFilesystemHandler.MakeDir(ctx, req)
}

func (h *testFilesystemHandler) Move(ctx context.Context, req *connect.Request[filesystem.MoveRequest]) (*connect.Response[filesystem.MoveResponse], error) {
	if h.moveFn != nil {
		return h.moveFn(ctx, req)
	}
	return h.UnimplementedFilesystemHandler.Move(ctx, req)
}

func (h *testFilesystemHandler) ListDir(ctx context.Context, req *connect.Request[filesystem.ListDirRequest]) (*connect.Response[filesystem.ListDirResponse], error) {
	if h.listDirFn != nil {
		return h.listDirFn(ctx, req)
	}
	return h.UnimplementedFilesystemHandler.ListDir(ctx, req)
}

func (h *testFilesystemHandler) Remove(ctx context.Context, req *connect.Request[filesystem.RemoveRequest]) (*connect.Response[filesystem.RemoveResponse], error) {
	if h.removeFn != nil {
		return h.removeFn(ctx, req)
	}
	return h.UnimplementedFilesystemHandler.Remove(ctx, req)
}

func (h *testFilesystemHandler) WatchDir(ctx context.Context, req *connect.Request[filesystem.WatchDirRequest], stream *connect.ServerStream[filesystem.WatchDirResponse]) error {
	if h.watchDirFn != nil {
		return h.watchDirFn(ctx, req, stream)
	}
	return h.UnimplementedFilesystemHandler.WatchDir(ctx, req, stream)
}

// ---------------------------------------------------------------------------
// testProcessHandler — mock ConnectRPC server for Process service
// ---------------------------------------------------------------------------

type testProcessHandler struct {
	processconnect.UnimplementedProcessHandler

	startFn      func(context.Context, *connect.Request[process.StartRequest], *connect.ServerStream[process.StartResponse]) error
	listFn       func(context.Context, *connect.Request[process.ListRequest]) (*connect.Response[process.ListResponse], error)
	sendInputFn  func(context.Context, *connect.Request[process.SendInputRequest]) (*connect.Response[process.SendInputResponse], error)
	sendSignalFn func(context.Context, *connect.Request[process.SendSignalRequest]) (*connect.Response[process.SendSignalResponse], error)
	updateFn     func(context.Context, *connect.Request[process.UpdateRequest]) (*connect.Response[process.UpdateResponse], error)
}

func (h *testProcessHandler) Start(ctx context.Context, req *connect.Request[process.StartRequest], stream *connect.ServerStream[process.StartResponse]) error {
	if h.startFn != nil {
		return h.startFn(ctx, req, stream)
	}
	return h.UnimplementedProcessHandler.Start(ctx, req, stream)
}

func (h *testProcessHandler) List(ctx context.Context, req *connect.Request[process.ListRequest]) (*connect.Response[process.ListResponse], error) {
	if h.listFn != nil {
		return h.listFn(ctx, req)
	}
	return h.UnimplementedProcessHandler.List(ctx, req)
}

func (h *testProcessHandler) SendInput(ctx context.Context, req *connect.Request[process.SendInputRequest]) (*connect.Response[process.SendInputResponse], error) {
	if h.sendInputFn != nil {
		return h.sendInputFn(ctx, req)
	}
	return h.UnimplementedProcessHandler.SendInput(ctx, req)
}

func (h *testProcessHandler) SendSignal(ctx context.Context, req *connect.Request[process.SendSignalRequest]) (*connect.Response[process.SendSignalResponse], error) {
	if h.sendSignalFn != nil {
		return h.sendSignalFn(ctx, req)
	}
	return h.UnimplementedProcessHandler.SendSignal(ctx, req)
}

func (h *testProcessHandler) Update(ctx context.Context, req *connect.Request[process.UpdateRequest]) (*connect.Response[process.UpdateResponse], error) {
	if h.updateFn != nil {
		return h.updateFn(ctx, req)
	}
	return h.UnimplementedProcessHandler.Update(ctx, req)
}

// ---------------------------------------------------------------------------
// helpers — start mock servers and create Filesystem / Commands / Pty
// ---------------------------------------------------------------------------

// newTestFilesystem starts a ConnectRPC mock server and returns a Filesystem
// whose rpc client points to it. The caller should call ts.Close() when done.
func newTestFilesystem(handler *testFilesystemHandler) (*Filesystem, *httptest.Server) {
	mux := http.NewServeMux()
	path, h := filesystemconnect.NewFilesystemHandler(handler)
	mux.Handle(path, h)
	ts := httptest.NewServer(mux)

	rpcClient := filesystemconnect.NewFilesystemClient(ts.Client(), ts.URL)
	c := &Client{config: &Config{Domain: "test.dev"}}
	sb := &Sandbox{sandboxID: "sb-test", client: c}
	fs := &Filesystem{sandbox: sb, rpc: rpcClient}
	return fs, ts
}

// newTestCommands starts a ConnectRPC mock server and returns Commands + server.
func newTestCommands(handler *testProcessHandler) (*Commands, *httptest.Server) {
	mux := http.NewServeMux()
	path, h := processconnect.NewProcessHandler(handler)
	mux.Handle(path, h)
	ts := httptest.NewServer(mux)

	rpcClient := processconnect.NewProcessClient(ts.Client(), ts.URL)
	c := &Client{config: &Config{Domain: "test.dev"}}
	sb := &Sandbox{sandboxID: "sb-test", client: c}
	cmd := &Commands{sandbox: sb, rpc: rpcClient}
	return cmd, ts
}

// newTestPty starts a ConnectRPC mock server and returns Pty + server.
func newTestPty(handler *testProcessHandler) (*Pty, *httptest.Server) {
	mux := http.NewServeMux()
	path, h := processconnect.NewProcessHandler(handler)
	mux.Handle(path, h)
	ts := httptest.NewServer(mux)

	rpcClient := processconnect.NewProcessClient(ts.Client(), ts.URL)
	c := &Client{config: &Config{Domain: "test.dev"}}
	sb := &Sandbox{sandboxID: "sb-test", client: c}
	p := &Pty{sandbox: sb, rpc: rpcClient}
	return p, ts
}

// newTestSandboxWithHTTP creates a Sandbox whose HTTPClient routes to the given
// test server. Useful for Filesystem.Read/Write tests.
func newTestSandboxWithHTTP(ts *httptest.Server) *Sandbox {
	c := &Client{config: &Config{Domain: "test.dev", HTTPClient: ts.Client()}}
	sb := &Sandbox{sandboxID: "sb-test", client: c}
	return sb
}

// =========================================================================
// 1. Filesystem RPC tests
// =========================================================================

func TestFilesystemList(t *testing.T) {
	now := timestamppb.Now()
	handler := &testFilesystemHandler{
		listDirFn: func(_ context.Context, req *connect.Request[filesystem.ListDirRequest]) (*connect.Response[filesystem.ListDirResponse], error) {
			if req.Msg.Path != "/home/user" {
				t.Errorf("ListDir path = %q, want %q", req.Msg.Path, "/home/user")
			}
			if req.Msg.Depth != 2 {
				t.Errorf("ListDir depth = %d, want 2", req.Msg.Depth)
			}
			return connect.NewResponse(&filesystem.ListDirResponse{
				Entries: []*filesystem.EntryInfo{
					{Name: "a.txt", Type: filesystem.FileType_FILE_TYPE_FILE, Path: "/home/user/a.txt", Size: 100, ModifiedTime: now},
					{Name: "sub", Type: filesystem.FileType_FILE_TYPE_DIRECTORY, Path: "/home/user/sub", ModifiedTime: now},
				},
			}), nil
		},
	}
	fs, ts := newTestFilesystem(handler)
	defer ts.Close()

	entries, err := fs.List(context.Background(), "/home/user", WithDepth(2))
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("List returned %d entries, want 2", len(entries))
	}
	if entries[0].Name != "a.txt" || entries[0].Type != FileTypeFile {
		t.Errorf("entries[0] = %+v, want name=a.txt type=file", entries[0])
	}
	if entries[1].Name != "sub" || entries[1].Type != FileTypeDirectory {
		t.Errorf("entries[1] = %+v, want name=sub type=dir", entries[1])
	}
}

func TestFilesystemGetInfo(t *testing.T) {
	now := timestamppb.New(time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC))
	symTarget := "/real/path"
	handler := &testFilesystemHandler{
		statFn: func(_ context.Context, req *connect.Request[filesystem.StatRequest]) (*connect.Response[filesystem.StatResponse], error) {
			if req.Msg.Path != "/tmp/test.txt" {
				t.Errorf("Stat path = %q, want %q", req.Msg.Path, "/tmp/test.txt")
			}
			return connect.NewResponse(&filesystem.StatResponse{
				Entry: &filesystem.EntryInfo{
					Name:          "test.txt",
					Type:          filesystem.FileType_FILE_TYPE_FILE,
					Path:          "/tmp/test.txt",
					Size:          256,
					Mode:          0644,
					Permissions:   "rw-r--r--",
					Owner:         "user",
					Group:         "user",
					ModifiedTime:  now,
					SymlinkTarget: &symTarget,
				},
			}), nil
		},
	}
	fs, ts := newTestFilesystem(handler)
	defer ts.Close()

	info, err := fs.GetInfo(context.Background(), "/tmp/test.txt")
	if err != nil {
		t.Fatalf("GetInfo error: %v", err)
	}
	if info.Name != "test.txt" {
		t.Errorf("Name = %q, want %q", info.Name, "test.txt")
	}
	if info.Type != FileTypeFile {
		t.Errorf("Type = %q, want %q", info.Type, FileTypeFile)
	}
	if info.Size != 256 {
		t.Errorf("Size = %d, want 256", info.Size)
	}
	if info.SymlinkTarget == nil || *info.SymlinkTarget != "/real/path" {
		t.Errorf("SymlinkTarget = %v, want /real/path", info.SymlinkTarget)
	}
	wantTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	if !info.ModifiedTime.Equal(wantTime) {
		t.Errorf("ModifiedTime = %v, want %v", info.ModifiedTime, wantTime)
	}
}

func TestFilesystemMakeDir(t *testing.T) {
	handler := &testFilesystemHandler{
		makeDirFn: func(_ context.Context, req *connect.Request[filesystem.MakeDirRequest]) (*connect.Response[filesystem.MakeDirResponse], error) {
			if req.Msg.Path != "/home/user/newdir" {
				t.Errorf("MakeDir path = %q, want %q", req.Msg.Path, "/home/user/newdir")
			}
			return connect.NewResponse(&filesystem.MakeDirResponse{
				Entry: &filesystem.EntryInfo{
					Name: "newdir",
					Type: filesystem.FileType_FILE_TYPE_DIRECTORY,
					Path: "/home/user/newdir",
				},
			}), nil
		},
	}
	fs, ts := newTestFilesystem(handler)
	defer ts.Close()

	info, err := fs.MakeDir(context.Background(), "/home/user/newdir")
	if err != nil {
		t.Fatalf("MakeDir error: %v", err)
	}
	if info.Name != "newdir" || info.Type != FileTypeDirectory {
		t.Errorf("MakeDir result = %+v, want name=newdir type=dir", info)
	}
}

func TestFilesystemRemove(t *testing.T) {
	handler := &testFilesystemHandler{
		removeFn: func(_ context.Context, req *connect.Request[filesystem.RemoveRequest]) (*connect.Response[filesystem.RemoveResponse], error) {
			if req.Msg.Path != "/tmp/old.txt" {
				t.Errorf("Remove path = %q, want %q", req.Msg.Path, "/tmp/old.txt")
			}
			return connect.NewResponse(&filesystem.RemoveResponse{}), nil
		},
	}
	fs, ts := newTestFilesystem(handler)
	defer ts.Close()

	if err := fs.Remove(context.Background(), "/tmp/old.txt"); err != nil {
		t.Fatalf("Remove error: %v", err)
	}
}

func TestFilesystemRename(t *testing.T) {
	handler := &testFilesystemHandler{
		moveFn: func(_ context.Context, req *connect.Request[filesystem.MoveRequest]) (*connect.Response[filesystem.MoveResponse], error) {
			if req.Msg.Source != "/a.txt" || req.Msg.Destination != "/b.txt" {
				t.Errorf("Move src=%q dst=%q, want src=/a.txt dst=/b.txt", req.Msg.Source, req.Msg.Destination)
			}
			return connect.NewResponse(&filesystem.MoveResponse{
				Entry: &filesystem.EntryInfo{
					Name: "b.txt",
					Type: filesystem.FileType_FILE_TYPE_FILE,
					Path: "/b.txt",
				},
			}), nil
		},
	}
	fs, ts := newTestFilesystem(handler)
	defer ts.Close()

	info, err := fs.Rename(context.Background(), "/a.txt", "/b.txt")
	if err != nil {
		t.Fatalf("Rename error: %v", err)
	}
	if info.Name != "b.txt" || info.Path != "/b.txt" {
		t.Errorf("Rename result = %+v", info)
	}
}

func TestFilesystemExists(t *testing.T) {
	handler := &testFilesystemHandler{
		statFn: func(_ context.Context, req *connect.Request[filesystem.StatRequest]) (*connect.Response[filesystem.StatResponse], error) {
			if req.Msg.Path == "/exists" {
				return connect.NewResponse(&filesystem.StatResponse{
					Entry: &filesystem.EntryInfo{Name: "exists", Path: "/exists"},
				}), nil
			}
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("not found"))
		},
	}
	fs, ts := newTestFilesystem(handler)
	defer ts.Close()

	ok, err := fs.Exists(context.Background(), "/exists")
	if err != nil {
		t.Fatalf("Exists error: %v", err)
	}
	if !ok {
		t.Error("Exists(/exists) = false, want true")
	}

	ok, err = fs.Exists(context.Background(), "/missing")
	if err != nil {
		t.Fatalf("Exists error: %v", err)
	}
	if ok {
		t.Error("Exists(/missing) = true, want false")
	}
}

func TestFilesystemWatchDir(t *testing.T) {
	handler := &testFilesystemHandler{
		watchDirFn: func(_ context.Context, req *connect.Request[filesystem.WatchDirRequest], stream *connect.ServerStream[filesystem.WatchDirResponse]) error {
			if req.Msg.Path != "/watch" {
				t.Errorf("WatchDir path = %q, want %q", req.Msg.Path, "/watch")
			}
			if !req.Msg.Recursive {
				t.Error("WatchDir recursive = false, want true")
			}
			// 发送 start 事件
			if err := stream.Send(&filesystem.WatchDirResponse{
				Event: &filesystem.WatchDirResponse_Start{
					Start: &filesystem.WatchDirResponse_StartEvent{},
				},
			}); err != nil {
				return err
			}
			// 发送文件系统事件
			events := []struct {
				name string
				typ  filesystem.EventType
			}{
				{"created.txt", filesystem.EventType_EVENT_TYPE_CREATE},
				{"modified.txt", filesystem.EventType_EVENT_TYPE_WRITE},
				{"removed.txt", filesystem.EventType_EVENT_TYPE_REMOVE},
			}
			for _, e := range events {
				if err := stream.Send(&filesystem.WatchDirResponse{
					Event: &filesystem.WatchDirResponse_Filesystem{
						Filesystem: &filesystem.FilesystemEvent{
							Name: e.name,
							Type: e.typ,
						},
					},
				}); err != nil {
					return err
				}
			}
			return nil
		},
	}
	fs, ts := newTestFilesystem(handler)
	defer ts.Close()

	w, err := fs.WatchDir(context.Background(), "/watch", WithRecursive(true))
	if err != nil {
		t.Fatalf("WatchDir error: %v", err)
	}

	var received []FilesystemEvent
	for ev := range w.Events() {
		received = append(received, ev)
	}

	if len(received) != 3 {
		t.Fatalf("received %d events, want 3", len(received))
	}

	wantEvents := []struct {
		name string
		typ  EventType
	}{
		{"created.txt", EventCreate},
		{"modified.txt", EventWrite},
		{"removed.txt", EventRemove},
	}
	for i, want := range wantEvents {
		if received[i].Name != want.name || received[i].Type != want.typ {
			t.Errorf("event[%d] = %+v, want name=%q type=%q", i, received[i], want.name, want.typ)
		}
	}

	if err := w.Err(); err != nil {
		t.Errorf("WatchHandle.Err() = %v, want nil", err)
	}
}

// =========================================================================
// 2. Filesystem HTTP (Read / Write) tests
// =========================================================================

func TestFilesystemRead(t *testing.T) {
	fileContent := []byte("hello world from sandbox")
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write(fileContent)
	}))
	defer httpServer.Close()

	sb := newTestSandboxWithHTTP(httpServer)
	// 创建 Filesystem 并注入 sandbox 使其使用 mock HTTPClient
	fs := &Filesystem{sandbox: sb}

	// 由于 Filesystem.Read 通过 sb.DownloadURL 构造 URL，其域名不匹配 test server，
	// 需要让 HTTPClient 的 transport 将请求路由到 test server。
	sb.client.config.HTTPClient = httpServer.Client()
	// 覆盖 sandbox 的 DownloadURL 使其指向 test server
	// 使用自定义 transport 将所有请求重定向到 test server
	sb.client.config.HTTPClient.Transport = &rewriteTransport{
		base:    httpServer.Client().Transport,
		baseURL: httpServer.URL,
	}

	data, err := fs.Read(context.Background(), "/test/file.txt")
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if string(data) != string(fileContent) {
		t.Errorf("Read = %q, want %q", string(data), string(fileContent))
	}
}

func TestFilesystemReadText(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "text content")
	}))
	defer httpServer.Close()

	sb := newTestSandboxWithHTTP(httpServer)
	sb.client.config.HTTPClient.Transport = &rewriteTransport{
		base:    httpServer.Client().Transport,
		baseURL: httpServer.URL,
	}
	fs := &Filesystem{sandbox: sb}

	text, err := fs.ReadText(context.Background(), "/readme.md")
	if err != nil {
		t.Fatalf("ReadText error: %v", err)
	}
	if text != "text content" {
		t.Errorf("ReadText = %q, want %q", text, "text content")
	}
}

func TestFilesystemReadError(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("file not found"))
	}))
	defer httpServer.Close()

	sb := newTestSandboxWithHTTP(httpServer)
	sb.client.config.HTTPClient.Transport = &rewriteTransport{
		base:    httpServer.Client().Transport,
		baseURL: httpServer.URL,
	}
	fs := &Filesystem{sandbox: sb}

	_, err := fs.Read(context.Background(), "/missing.txt")
	if err == nil {
		t.Fatal("Read expected error for 404, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

func TestFilesystemWrite(t *testing.T) {
	var uploadedBody []byte
	// mux 处理 HTTP 上传和 RPC Stat
	mux := http.NewServeMux()

	// HTTP 文件上传端点
	mux.HandleFunc("/files", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		uploadedBody = body
		w.WriteHeader(http.StatusOK)
	})

	// ConnectRPC Stat 端点（Write 成功后会调用 GetInfo）
	fsHandler := &testFilesystemHandler{
		statFn: func(_ context.Context, req *connect.Request[filesystem.StatRequest]) (*connect.Response[filesystem.StatResponse], error) {
			return connect.NewResponse(&filesystem.StatResponse{
				Entry: &filesystem.EntryInfo{
					Name: "data.bin",
					Type: filesystem.FileType_FILE_TYPE_FILE,
					Path: req.Msg.Path,
					Size: 5,
				},
			}), nil
		},
	}
	rpcPath, rpcHandler := filesystemconnect.NewFilesystemHandler(fsHandler)
	mux.Handle(rpcPath, rpcHandler)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	sb := newTestSandboxWithHTTP(ts)
	sb.client.config.HTTPClient.Transport = &rewriteTransport{
		base:    ts.Client().Transport,
		baseURL: ts.URL,
	}
	rpcClient := filesystemconnect.NewFilesystemClient(ts.Client(), ts.URL)
	fs := &Filesystem{sandbox: sb, rpc: rpcClient}

	info, err := fs.Write(context.Background(), "/data.bin", []byte("hello"))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if info == nil {
		t.Fatal("Write returned nil info")
	}
	if info.Name != "data.bin" || info.Size != 5 {
		t.Errorf("Write info = %+v", info)
	}
	if len(uploadedBody) == 0 {
		t.Error("upload body was empty")
	}
}

func TestFilesystemWriteFiles(t *testing.T) {
	uploadCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/files", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			uploadCount++
			io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	})

	statCalls := 0
	fsHandler := &testFilesystemHandler{
		statFn: func(_ context.Context, req *connect.Request[filesystem.StatRequest]) (*connect.Response[filesystem.StatResponse], error) {
			statCalls++
			return connect.NewResponse(&filesystem.StatResponse{
				Entry: &filesystem.EntryInfo{
					Name: "file",
					Type: filesystem.FileType_FILE_TYPE_FILE,
					Path: req.Msg.Path,
					Size: 3,
				},
			}), nil
		},
	}
	rpcPath, rpcHandler := filesystemconnect.NewFilesystemHandler(fsHandler)
	mux.Handle(rpcPath, rpcHandler)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	sb := newTestSandboxWithHTTP(ts)
	sb.client.config.HTTPClient.Transport = &rewriteTransport{
		base:    ts.Client().Transport,
		baseURL: ts.URL,
	}
	rpcClient := filesystemconnect.NewFilesystemClient(ts.Client(), ts.URL)
	fs := &Filesystem{sandbox: sb, rpc: rpcClient}

	files := []WriteEntry{
		{Path: "/a.txt", Data: []byte("aaa")},
		{Path: "/b.txt", Data: []byte("bbb")},
		{Path: "/c.txt", Data: []byte("ccc")},
	}
	infos, err := fs.WriteFiles(context.Background(), files)
	if err != nil {
		t.Fatalf("WriteFiles error: %v", err)
	}
	if len(infos) != 3 {
		t.Fatalf("WriteFiles returned %d infos, want 3", len(infos))
	}
	// 多文件使用单次 batch upload
	if uploadCount != 1 {
		t.Errorf("upload count = %d, want 1", uploadCount)
	}
	// 每个文件调用 GetInfo
	if statCalls != 3 {
		t.Errorf("stat calls = %d, want 3", statCalls)
	}
}

// =========================================================================
// 3. Commands tests
// =========================================================================

func TestCommandsRun(t *testing.T) {
	handler := &testProcessHandler{
		startFn: func(_ context.Context, req *connect.Request[process.StartRequest], stream *connect.ServerStream[process.StartResponse]) error {
			// Start 事件
			if err := stream.Send(&process.StartResponse{
				Event: &process.ProcessEvent{
					Event: &process.ProcessEvent_Start{
						Start: &process.ProcessEvent_StartEvent{Pid: 42},
					},
				},
			}); err != nil {
				return err
			}
			// Data 事件 (stdout)
			if err := stream.Send(&process.StartResponse{
				Event: &process.ProcessEvent{
					Event: &process.ProcessEvent_Data{
						Data: &process.ProcessEvent_DataEvent{
							Output: &process.ProcessEvent_DataEvent_Stdout{
								Stdout: []byte("hello\n"),
							},
						},
					},
				},
			}); err != nil {
				return err
			}
			// Data 事件 (stderr)
			if err := stream.Send(&process.StartResponse{
				Event: &process.ProcessEvent{
					Event: &process.ProcessEvent_Data{
						Data: &process.ProcessEvent_DataEvent{
							Output: &process.ProcessEvent_DataEvent_Stderr{
								Stderr: []byte("warn\n"),
							},
						},
					},
				},
			}); err != nil {
				return err
			}
			// End 事件
			return stream.Send(&process.StartResponse{
				Event: &process.ProcessEvent{
					Event: &process.ProcessEvent_End{
						End: &process.ProcessEvent_EndEvent{
							ExitCode: 0,
							Exited:   true,
							Status:   "exited",
						},
					},
				},
			})
		},
	}
	cmd, ts := newTestCommands(handler)
	defer ts.Close()

	result, err := cmd.Run(context.Background(), "echo hello")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if result.Stdout != "hello\n" {
		t.Errorf("Stdout = %q, want %q", result.Stdout, "hello\n")
	}
	if result.Stderr != "warn\n" {
		t.Errorf("Stderr = %q, want %q", result.Stderr, "warn\n")
	}
}

func TestCommandsStart(t *testing.T) {
	handler := &testProcessHandler{
		startFn: func(_ context.Context, _ *connect.Request[process.StartRequest], stream *connect.ServerStream[process.StartResponse]) error {
			stream.Send(&process.StartResponse{
				Event: &process.ProcessEvent{
					Event: &process.ProcessEvent_Start{
						Start: &process.ProcessEvent_StartEvent{Pid: 100},
					},
				},
			})
			stream.Send(&process.StartResponse{
				Event: &process.ProcessEvent{
					Event: &process.ProcessEvent_End{
						End: &process.ProcessEvent_EndEvent{ExitCode: 1, Status: "exited"},
					},
				},
			})
			return nil
		},
	}
	cmd, ts := newTestCommands(handler)
	defer ts.Close()

	handle, err := cmd.Start(context.Background(), "false")
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}

	result, err := handle.Wait()
	if err != nil {
		t.Fatalf("Wait error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", result.ExitCode)
	}
	if handle.PID() != 100 {
		t.Errorf("PID = %d, want 100", handle.PID())
	}
}

func TestCommandsRunWithCallbacks(t *testing.T) {
	handler := &testProcessHandler{
		startFn: func(_ context.Context, _ *connect.Request[process.StartRequest], stream *connect.ServerStream[process.StartResponse]) error {
			stream.Send(&process.StartResponse{
				Event: &process.ProcessEvent{
					Event: &process.ProcessEvent_Start{
						Start: &process.ProcessEvent_StartEvent{Pid: 50},
					},
				},
			})
			stream.Send(&process.StartResponse{
				Event: &process.ProcessEvent{
					Event: &process.ProcessEvent_Data{
						Data: &process.ProcessEvent_DataEvent{
							Output: &process.ProcessEvent_DataEvent_Stdout{
								Stdout: []byte("out1"),
							},
						},
					},
				},
			})
			stream.Send(&process.StartResponse{
				Event: &process.ProcessEvent{
					Event: &process.ProcessEvent_Data{
						Data: &process.ProcessEvent_DataEvent{
							Output: &process.ProcessEvent_DataEvent_Stderr{
								Stderr: []byte("err1"),
							},
						},
					},
				},
			})
			stream.Send(&process.StartResponse{
				Event: &process.ProcessEvent{
					Event: &process.ProcessEvent_End{
						End: &process.ProcessEvent_EndEvent{ExitCode: 0},
					},
				},
			})
			return nil
		},
	}
	cmd, ts := newTestCommands(handler)
	defer ts.Close()

	var mu sync.Mutex
	var stdoutParts, stderrParts []string

	result, err := cmd.Run(context.Background(), "test",
		WithOnStdout(func(data []byte) {
			mu.Lock()
			stdoutParts = append(stdoutParts, string(data))
			mu.Unlock()
		}),
		WithOnStderr(func(data []byte) {
			mu.Lock()
			stderrParts = append(stderrParts, string(data))
			mu.Unlock()
		}),
	)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(stdoutParts) != 1 || stdoutParts[0] != "out1" {
		t.Errorf("stdout callback = %v, want [out1]", stdoutParts)
	}
	if len(stderrParts) != 1 || stderrParts[0] != "err1" {
		t.Errorf("stderr callback = %v, want [err1]", stderrParts)
	}
}

func TestCommandsList(t *testing.T) {
	tag := "mytag"
	cwd := "/home/user"
	handler := &testProcessHandler{
		listFn: func(_ context.Context, _ *connect.Request[process.ListRequest]) (*connect.Response[process.ListResponse], error) {
			return connect.NewResponse(&process.ListResponse{
				Processes: []*process.ProcessInfo{
					{
						Pid: 1,
						Tag: &tag,
						Config: &process.ProcessConfig{
							Cmd:  "/bin/bash",
							Args: []string{"-c", "sleep 100"},
							Envs: map[string]string{"A": "B"},
							Cwd:  &cwd,
						},
					},
					{
						Pid: 2,
					},
				},
			}), nil
		},
	}
	cmd, ts := newTestCommands(handler)
	defer ts.Close()

	infos, err := cmd.List(context.Background())
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("List returned %d, want 2", len(infos))
	}

	if infos[0].PID != 1 {
		t.Errorf("PID = %d, want 1", infos[0].PID)
	}
	if infos[0].Tag == nil || *infos[0].Tag != "mytag" {
		t.Errorf("Tag = %v, want mytag", infos[0].Tag)
	}
	if infos[0].Cmd != "/bin/bash" {
		t.Errorf("Cmd = %q, want /bin/bash", infos[0].Cmd)
	}
	if infos[0].Cwd == nil || *infos[0].Cwd != "/home/user" {
		t.Errorf("Cwd = %v, want /home/user", infos[0].Cwd)
	}
	if infos[0].Envs["A"] != "B" {
		t.Errorf("Envs = %v, want A=B", infos[0].Envs)
	}

	// 第二个进程没有 config
	if infos[1].PID != 2 {
		t.Errorf("PID = %d, want 2", infos[1].PID)
	}
	if infos[1].Cmd != "" {
		t.Errorf("Cmd = %q, want empty", infos[1].Cmd)
	}
}

func TestCommandsSendStdin(t *testing.T) {
	var receivedPID uint32
	var receivedData []byte
	handler := &testProcessHandler{
		sendInputFn: func(_ context.Context, req *connect.Request[process.SendInputRequest]) (*connect.Response[process.SendInputResponse], error) {
			if sel, ok := req.Msg.Process.Selector.(*process.ProcessSelector_Pid); ok {
				receivedPID = sel.Pid
			}
			if stdin := req.Msg.Input.GetStdin(); len(stdin) > 0 {
				receivedData = stdin
			}
			return connect.NewResponse(&process.SendInputResponse{}), nil
		},
	}
	cmd, ts := newTestCommands(handler)
	defer ts.Close()

	err := cmd.SendStdin(context.Background(), 77, []byte("input data"))
	if err != nil {
		t.Fatalf("SendStdin error: %v", err)
	}
	if receivedPID != 77 {
		t.Errorf("PID = %d, want 77", receivedPID)
	}
	if string(receivedData) != "input data" {
		t.Errorf("data = %q, want %q", string(receivedData), "input data")
	}
}

func TestCommandsKill(t *testing.T) {
	var receivedPID uint32
	var receivedSignal process.Signal
	handler := &testProcessHandler{
		sendSignalFn: func(_ context.Context, req *connect.Request[process.SendSignalRequest]) (*connect.Response[process.SendSignalResponse], error) {
			if sel, ok := req.Msg.Process.Selector.(*process.ProcessSelector_Pid); ok {
				receivedPID = sel.Pid
			}
			receivedSignal = req.Msg.Signal
			return connect.NewResponse(&process.SendSignalResponse{}), nil
		},
	}
	cmd, ts := newTestCommands(handler)
	defer ts.Close()

	err := cmd.Kill(context.Background(), 99)
	if err != nil {
		t.Fatalf("Kill error: %v", err)
	}
	if receivedPID != 99 {
		t.Errorf("PID = %d, want 99", receivedPID)
	}
	if receivedSignal != process.Signal_SIGNAL_SIGKILL {
		t.Errorf("Signal = %v, want SIGKILL", receivedSignal)
	}
}

func TestCommandsRunWithError(t *testing.T) {
	errMsg := "command failed"
	handler := &testProcessHandler{
		startFn: func(_ context.Context, _ *connect.Request[process.StartRequest], stream *connect.ServerStream[process.StartResponse]) error {
			stream.Send(&process.StartResponse{
				Event: &process.ProcessEvent{
					Event: &process.ProcessEvent_Start{
						Start: &process.ProcessEvent_StartEvent{Pid: 10},
					},
				},
			})
			stream.Send(&process.StartResponse{
				Event: &process.ProcessEvent{
					Event: &process.ProcessEvent_End{
						End: &process.ProcessEvent_EndEvent{
							ExitCode: 127,
							Error:    &errMsg,
						},
					},
				},
			})
			return nil
		},
	}
	cmd, ts := newTestCommands(handler)
	defer ts.Close()

	result, err := cmd.Run(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.ExitCode != 127 {
		t.Errorf("ExitCode = %d, want 127", result.ExitCode)
	}
	if result.Error != "command failed" {
		t.Errorf("Error = %q, want %q", result.Error, "command failed")
	}
}

// =========================================================================
// 4. PTY tests
// =========================================================================

func TestPtyCreate(t *testing.T) {
	handler := &testProcessHandler{
		startFn: func(_ context.Context, req *connect.Request[process.StartRequest], stream *connect.ServerStream[process.StartResponse]) error {
			// 验证 PTY 配置
			if req.Msg.Pty == nil || req.Msg.Pty.Size == nil {
				t.Error("expected PTY size in request")
				return nil
			}
			if req.Msg.Pty.Size.Cols != 120 || req.Msg.Pty.Size.Rows != 40 {
				t.Errorf("PTY size = %dx%d, want 120x40", req.Msg.Pty.Size.Cols, req.Msg.Pty.Size.Rows)
			}
			// 验证 bash -i -l 命令
			if req.Msg.Process.Cmd != "/bin/bash" {
				t.Errorf("Cmd = %q, want /bin/bash", req.Msg.Process.Cmd)
			}
			if len(req.Msg.Process.Args) < 2 || req.Msg.Process.Args[0] != "-i" || req.Msg.Process.Args[1] != "-l" {
				t.Errorf("Args = %v, want [-i -l]", req.Msg.Process.Args)
			}
			// 验证默认环境变量
			if req.Msg.Process.Envs["TERM"] != "xterm" {
				t.Errorf("TERM = %q, want xterm", req.Msg.Process.Envs["TERM"])
			}

			// Start 事件
			stream.Send(&process.StartResponse{
				Event: &process.ProcessEvent{
					Event: &process.ProcessEvent_Start{
						Start: &process.ProcessEvent_StartEvent{Pid: 200},
					},
				},
			})
			// PTY data 事件
			stream.Send(&process.StartResponse{
				Event: &process.ProcessEvent{
					Event: &process.ProcessEvent_Data{
						Data: &process.ProcessEvent_DataEvent{
							Output: &process.ProcessEvent_DataEvent_Pty{
								Pty: []byte("bash$ "),
							},
						},
					},
				},
			})
			// End 事件
			stream.Send(&process.StartResponse{
				Event: &process.ProcessEvent{
					Event: &process.ProcessEvent_End{
						End: &process.ProcessEvent_EndEvent{ExitCode: 0},
					},
				},
			})
			return nil
		},
	}
	pty, ts := newTestPty(handler)
	defer ts.Close()

	var ptyOutput []byte
	var mu sync.Mutex
	handle, err := pty.Create(context.Background(), PtySize{Cols: 120, Rows: 40},
		WithOnStdout(func(data []byte) {
			mu.Lock()
			ptyOutput = append(ptyOutput, data...)
			mu.Unlock()
		}),
	)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}

	result, err := handle.Wait()
	if err != nil {
		t.Fatalf("Wait error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}

	mu.Lock()
	defer mu.Unlock()
	if string(ptyOutput) != "bash$ " {
		t.Errorf("pty output = %q, want %q", string(ptyOutput), "bash$ ")
	}
}

func TestPtySendInput(t *testing.T) {
	var receivedPID uint32
	var receivedPtyData []byte
	handler := &testProcessHandler{
		sendInputFn: func(_ context.Context, req *connect.Request[process.SendInputRequest]) (*connect.Response[process.SendInputResponse], error) {
			if sel, ok := req.Msg.Process.Selector.(*process.ProcessSelector_Pid); ok {
				receivedPID = sel.Pid
			}
			if ptyData := req.Msg.Input.GetPty(); len(ptyData) > 0 {
				receivedPtyData = ptyData
			}
			return connect.NewResponse(&process.SendInputResponse{}), nil
		},
	}
	pty, ts := newTestPty(handler)
	defer ts.Close()

	err := pty.SendInput(context.Background(), 200, []byte("ls\n"))
	if err != nil {
		t.Fatalf("SendInput error: %v", err)
	}
	if receivedPID != 200 {
		t.Errorf("PID = %d, want 200", receivedPID)
	}
	if string(receivedPtyData) != "ls\n" {
		t.Errorf("pty data = %q, want %q", string(receivedPtyData), "ls\n")
	}
}

func TestPtyResize(t *testing.T) {
	var receivedCols, receivedRows uint32
	handler := &testProcessHandler{
		updateFn: func(_ context.Context, req *connect.Request[process.UpdateRequest]) (*connect.Response[process.UpdateResponse], error) {
			if req.Msg.Pty != nil && req.Msg.Pty.Size != nil {
				receivedCols = req.Msg.Pty.Size.Cols
				receivedRows = req.Msg.Pty.Size.Rows
			}
			return connect.NewResponse(&process.UpdateResponse{}), nil
		},
	}
	pty, ts := newTestPty(handler)
	defer ts.Close()

	err := pty.Resize(context.Background(), 200, PtySize{Cols: 200, Rows: 50})
	if err != nil {
		t.Fatalf("Resize error: %v", err)
	}
	if receivedCols != 200 || receivedRows != 50 {
		t.Errorf("resize = %dx%d, want 200x50", receivedCols, receivedRows)
	}
}

func TestPtyKill(t *testing.T) {
	var receivedPID uint32
	var receivedSignal process.Signal
	handler := &testProcessHandler{
		sendSignalFn: func(_ context.Context, req *connect.Request[process.SendSignalRequest]) (*connect.Response[process.SendSignalResponse], error) {
			if sel, ok := req.Msg.Process.Selector.(*process.ProcessSelector_Pid); ok {
				receivedPID = sel.Pid
			}
			receivedSignal = req.Msg.Signal
			return connect.NewResponse(&process.SendSignalResponse{}), nil
		},
	}
	pty, ts := newTestPty(handler)
	defer ts.Close()

	err := pty.Kill(context.Background(), 300)
	if err != nil {
		t.Fatalf("Kill error: %v", err)
	}
	if receivedPID != 300 {
		t.Errorf("PID = %d, want 300", receivedPID)
	}
	if receivedSignal != process.Signal_SIGNAL_SIGKILL {
		t.Errorf("Signal = %v, want SIGKILL", receivedSignal)
	}
}

// =========================================================================
// 6. 纯函数 / Option 补充测试
// =========================================================================

func TestEntryInfoFromProtoFull(t *testing.T) {
	ts := timestamppb.New(time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC))
	symlink := "/link/target"
	proto := &filesystem.EntryInfo{
		Name:          "link.txt",
		Type:          filesystem.FileType_FILE_TYPE_FILE,
		Path:          "/home/user/link.txt",
		Size:          512,
		Mode:          0777,
		Permissions:   "rwxrwxrwx",
		Owner:         "root",
		Group:         "root",
		ModifiedTime:  ts,
		SymlinkTarget: &symlink,
	}

	info := entryInfoFromProto(proto)
	if info == nil {
		t.Fatal("entryInfoFromProto returned nil")
	}
	if info.Name != "link.txt" {
		t.Errorf("Name = %q", info.Name)
	}
	if info.Type != FileTypeFile {
		t.Errorf("Type = %q", info.Type)
	}
	if info.Path != "/home/user/link.txt" {
		t.Errorf("Path = %q", info.Path)
	}
	if info.Size != 512 {
		t.Errorf("Size = %d", info.Size)
	}
	if info.Mode != 0777 {
		t.Errorf("Mode = %o", info.Mode)
	}
	if info.Permissions != "rwxrwxrwx" {
		t.Errorf("Permissions = %q", info.Permissions)
	}
	if info.Owner != "root" {
		t.Errorf("Owner = %q", info.Owner)
	}
	if info.Group != "root" {
		t.Errorf("Group = %q", info.Group)
	}
	wantTime := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	if !info.ModifiedTime.Equal(wantTime) {
		t.Errorf("ModifiedTime = %v, want %v", info.ModifiedTime, wantTime)
	}
	if info.SymlinkTarget == nil || *info.SymlinkTarget != "/link/target" {
		t.Errorf("SymlinkTarget = %v", info.SymlinkTarget)
	}
}

func TestEntryInfoFromProtoDirectory(t *testing.T) {
	proto := &filesystem.EntryInfo{
		Name: "docs",
		Type: filesystem.FileType_FILE_TYPE_DIRECTORY,
		Path: "/docs",
	}
	info := entryInfoFromProto(proto)
	if info.Type != FileTypeDirectory {
		t.Errorf("Type = %q, want %q", info.Type, FileTypeDirectory)
	}
}

func TestFilesystemEventFromProto(t *testing.T) {
	tests := []struct {
		protoType filesystem.EventType
		wantType  EventType
	}{
		{filesystem.EventType_EVENT_TYPE_CREATE, EventCreate},
		{filesystem.EventType_EVENT_TYPE_WRITE, EventWrite},
		{filesystem.EventType_EVENT_TYPE_REMOVE, EventRemove},
		{filesystem.EventType_EVENT_TYPE_RENAME, EventRename},
		{filesystem.EventType_EVENT_TYPE_CHMOD, EventChmod},
	}

	for _, tt := range tests {
		ev := filesystemEventFromProto(&filesystem.FilesystemEvent{
			Name: "test.txt",
			Type: tt.protoType,
		})
		if ev.Name != "test.txt" {
			t.Errorf("Name = %q, want test.txt", ev.Name)
		}
		if ev.Type != tt.wantType {
			t.Errorf("EventType for %v = %q, want %q", tt.protoType, ev.Type, tt.wantType)
		}
	}
}

func TestCommandOptions(t *testing.T) {
	called := false
	opts := applyCommandOpts([]CommandOption{
		WithEnvs(map[string]string{"FOO": "BAR"}),
		WithCwd("/tmp"),
		WithCommandUser("admin"),
		WithTag("test-tag"),
		WithOnStdout(func([]byte) { called = true }),
		WithOnStderr(func([]byte) {}),
		WithTimeout(5 * time.Second),
	})

	if opts.user != "admin" {
		t.Errorf("user = %q, want admin", opts.user)
	}
	if opts.cwd != "/tmp" {
		t.Errorf("cwd = %q, want /tmp", opts.cwd)
	}
	if opts.tag != "test-tag" {
		t.Errorf("tag = %q, want test-tag", opts.tag)
	}
	if opts.envs["FOO"] != "BAR" {
		t.Errorf("envs = %v", opts.envs)
	}
	if opts.timeout != 5*time.Second {
		t.Errorf("timeout = %v, want 5s", opts.timeout)
	}
	if opts.onStdout == nil {
		t.Error("onStdout is nil")
	} else {
		opts.onStdout(nil)
		if !called {
			t.Error("onStdout callback not invoked")
		}
	}
	if opts.onStderr == nil {
		t.Error("onStderr is nil")
	}
}

func TestCommandOptionsDefaults(t *testing.T) {
	opts := applyCommandOpts(nil)
	if opts.user != "user" {
		t.Errorf("default user = %q, want user", opts.user)
	}
	if opts.cwd != "" {
		t.Errorf("default cwd = %q, want empty", opts.cwd)
	}
	if opts.timeout != 0 {
		t.Errorf("default timeout = %v, want 0", opts.timeout)
	}
}

func TestFilesystemListOptions(t *testing.T) {
	opts := applyListOpts([]ListOption{
		WithDepth(5),
		WithListUser("admin"),
	})
	if opts.depth != 5 {
		t.Errorf("depth = %d, want 5", opts.depth)
	}
	if opts.user != "admin" {
		t.Errorf("user = %q, want admin", opts.user)
	}
}

func TestFilesystemListOptionsDefaults(t *testing.T) {
	opts := applyListOpts(nil)
	if opts.depth != 1 {
		t.Errorf("default depth = %d, want 1", opts.depth)
	}
	if opts.user != "user" {
		t.Errorf("default user = %q, want user", opts.user)
	}
}

func TestWatchOptions(t *testing.T) {
	opts := applyWatchOpts([]WatchOption{
		WithRecursive(true),
		WithWatchUser("admin"),
	})
	if !opts.recursive {
		t.Error("recursive = false, want true")
	}
	if opts.user != "admin" {
		t.Errorf("user = %q, want admin", opts.user)
	}
}

func TestWatchOptionsDefaults(t *testing.T) {
	opts := applyWatchOpts(nil)
	if opts.recursive {
		t.Error("default recursive = true, want false")
	}
	if opts.user != "user" {
		t.Errorf("default user = %q, want user", opts.user)
	}
}

func TestFilesystemOptions(t *testing.T) {
	opts := applyFilesystemOpts([]FilesystemOption{
		WithUser("root"),
	})
	if opts.user != "root" {
		t.Errorf("user = %q, want root", opts.user)
	}
}

func TestFilesystemOptionsDefaults(t *testing.T) {
	opts := applyFilesystemOpts(nil)
	if opts.user != "user" {
		t.Errorf("default user = %q, want user", opts.user)
	}
}

// =========================================================================
// helpers
// =========================================================================

// rewriteTransport 将所有 HTTP 请求重定向到指定的 baseURL（test server）。
type rewriteTransport struct {
	base    http.RoundTripper
	baseURL string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// 保留原始 path 和 query，替换 scheme + host
	orig := req.URL.String()
	_ = orig
	req.URL.Scheme = "http"
	req.URL.Host = t.baseURL[len("http://"):]
	return t.base.RoundTrip(req)
}
