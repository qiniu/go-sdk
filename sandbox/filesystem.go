package sandbox

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/qiniu/go-sdk/v7/sandbox/envdapi/filesystem"
	"github.com/qiniu/go-sdk/v7/sandbox/envdapi/filesystem/filesystemconnect"
)

// FileType 文件类型。
type FileType string

const (
	// FileTypeFile 表示普通文件。
	FileTypeFile FileType = "file"
	// FileTypeDirectory 表示目录。
	FileTypeDirectory FileType = "dir"
)

// EntryInfo 文件或目录的元信息。
type EntryInfo struct {
	Name          string
	Type          FileType
	Path          string
	Size          int64
	Mode          uint32
	Permissions   string
	Owner         string
	Group         string
	ModifiedTime  time.Time
	SymlinkTarget *string
}

// entryInfoFromProto 将 protobuf EntryInfo 转换为 SDK EntryInfo。
func entryInfoFromProto(e *filesystem.EntryInfo) *EntryInfo {
	if e == nil {
		return nil
	}
	info := &EntryInfo{
		Name:        e.Name,
		Path:        e.Path,
		Size:        e.Size,
		Mode:        e.Mode,
		Permissions: e.Permissions,
		Owner:       e.Owner,
		Group:       e.Group,
	}
	switch e.Type {
	case filesystem.FileType_FILE_TYPE_FILE:
		info.Type = FileTypeFile
	case filesystem.FileType_FILE_TYPE_DIRECTORY:
		info.Type = FileTypeDirectory
	}
	if e.ModifiedTime != nil {
		info.ModifiedTime = e.ModifiedTime.AsTime()
	}
	if e.SymlinkTarget != nil {
		t := *e.SymlinkTarget
		info.SymlinkTarget = &t
	}
	return info
}

// EventType 文件系统事件类型。
type EventType string

const (
	// EventCreate 文件或目录被创建。
	EventCreate EventType = "create"
	// EventWrite 文件被写入。
	EventWrite EventType = "write"
	// EventRemove 文件或目录被删除。
	EventRemove EventType = "remove"
	// EventRename 文件或目录被重命名。
	EventRename EventType = "rename"
	// EventChmod 文件或目录权限被修改。
	EventChmod EventType = "chmod"
)

// FilesystemEvent 文件系统事件。
type FilesystemEvent struct {
	Name string
	Type EventType
}

// filesystemEventFromProto 将 protobuf FilesystemEvent 转换为 SDK FilesystemEvent。
func filesystemEventFromProto(e *filesystem.FilesystemEvent) FilesystemEvent {
	ev := FilesystemEvent{Name: e.Name}
	switch e.Type {
	case filesystem.EventType_EVENT_TYPE_CREATE:
		ev.Type = EventCreate
	case filesystem.EventType_EVENT_TYPE_WRITE:
		ev.Type = EventWrite
	case filesystem.EventType_EVENT_TYPE_REMOVE:
		ev.Type = EventRemove
	case filesystem.EventType_EVENT_TYPE_RENAME:
		ev.Type = EventRename
	case filesystem.EventType_EVENT_TYPE_CHMOD:
		ev.Type = EventChmod
	}
	return ev
}

// FilesystemOption 文件系统操作选项。
type FilesystemOption func(*filesystemOpts)

type filesystemOpts struct {
	user string
}

// WithUser 设置文件系统操作的用户身份。
func WithUser(user string) FilesystemOption {
	return func(o *filesystemOpts) { o.user = user }
}

func applyFilesystemOpts(opts []FilesystemOption) *filesystemOpts {
	o := &filesystemOpts{user: "user"}
	for _, fn := range opts {
		fn(o)
	}
	return o
}

// ListOption 列目录选项。
type ListOption func(*listOpts)

type listOpts struct {
	filesystemOpts
	depth uint32
}

// WithDepth 设置列目录的递归深度，默认为 1。
func WithDepth(depth uint32) ListOption {
	return func(o *listOpts) { o.depth = depth }
}

// WithListUser 设置列目录操作的用户身份。
func WithListUser(user string) ListOption {
	return func(o *listOpts) { o.user = user }
}

func applyListOpts(opts []ListOption) *listOpts {
	o := &listOpts{
		filesystemOpts: filesystemOpts{user: "user"},
		depth:          1,
	}
	for _, fn := range opts {
		fn(o)
	}
	return o
}

// WatchOption 目录监听选项。
type WatchOption func(*watchOpts)

type watchOpts struct {
	filesystemOpts
	recursive bool
}

// WithRecursive 设置是否递归监听子目录。
func WithRecursive(recursive bool) WatchOption {
	return func(o *watchOpts) { o.recursive = recursive }
}

// WithWatchUser 设置监听操作的用户身份。
func WithWatchUser(user string) WatchOption {
	return func(o *watchOpts) { o.user = user }
}

func applyWatchOpts(opts []WatchOption) *watchOpts {
	o := &watchOpts{
		filesystemOpts: filesystemOpts{user: "user"},
	}
	for _, fn := range opts {
		fn(o)
	}
	return o
}

// WatchHandle 目录监听句柄。
type WatchHandle struct {
	events chan FilesystemEvent
	cancel context.CancelFunc
	done   chan struct{}
	err    error
}

// Events 返回文件系统事件通道。
func (w *WatchHandle) Events() <-chan FilesystemEvent {
	return w.events
}

// Err 返回监听过程中发生的错误。应在 Events 通道关闭后调用。
// 若通过 Stop() 正常停止，Err() 返回 nil；仅在流异常结束时返回错误。
func (w *WatchHandle) Err() error {
	return w.err
}

// Stop 停止监听。
func (w *WatchHandle) Stop() {
	w.cancel()
	<-w.done
}

// Filesystem 提供沙箱文件系统操作。
type Filesystem struct {
	sandbox *Sandbox
	rpc     filesystemconnect.FilesystemClient
}

// newFilesystem 创建 Filesystem 实例。
func newFilesystem(s *Sandbox) *Filesystem {
	httpClient := s.client.config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	rpc := filesystemconnect.NewFilesystemClient(
		httpClient,
		s.envdURL(),
	)
	return &Filesystem{sandbox: s, rpc: rpc}
}

// Read 读取指定路径的文件内容。
// 通过 envd HTTP API 下载文件。
func (fs *Filesystem) Read(ctx context.Context, path string, opts ...FilesystemOption) ([]byte, error) {
	resp, err := fs.doRead(ctx, path, opts...)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// ReadText 读取指定路径的文件内容，返回 string。
func (fs *Filesystem) ReadText(ctx context.Context, path string, opts ...FilesystemOption) (string, error) {
	data, err := fs.Read(ctx, path, opts...)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ReadStream 读取指定路径的文件内容，返回 io.ReadCloser 流。
// 调用方负责关闭返回的 ReadCloser。
func (fs *Filesystem) ReadStream(ctx context.Context, path string, opts ...FilesystemOption) (io.ReadCloser, error) {
	resp, err := fs.doRead(ctx, path, opts...)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// doRead 执行文件下载请求，返回原始 HTTP 响应。
// 调用方负责关闭 resp.Body。
func (fs *Filesystem) doRead(ctx context.Context, path string, opts ...FilesystemOption) (*http.Response, error) {
	o := applyFilesystemOpts(opts)
	downloadURL := fs.sandbox.DownloadURL(path, WithFileUser(o.user))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpClient := fs.sandbox.client.config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download file: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, &APIError{StatusCode: resp.StatusCode, Body: body}
	}

	return resp, nil
}

// Write 写入文件内容。如果文件已存在则覆盖，自动创建父目录。
// 通过 envd HTTP API 上传文件。
func (fs *Filesystem) Write(ctx context.Context, path string, data []byte, opts ...FilesystemOption) (*EntryInfo, error) {
	o := applyFilesystemOpts(opts)
	uploadURL := fs.sandbox.UploadURL(path, WithFileUser(o.user))

	pr, pw := io.Pipe()
	writer := newMultipartWriter(pw)

	go func() {
		if err := writer.writeFile("file", path, data); err != nil {
			pw.CloseWithError(err)
			return
		}
		if err := writer.close(); err != nil {
			pw.CloseWithError(err)
			return
		}
		pw.Close()
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, pr)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.contentType())

	httpClient := fs.sandbox.client.config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{StatusCode: resp.StatusCode, Body: body}
	}

	// 上传成功后通过 Stat 获取文件信息
	return fs.GetInfo(ctx, path, opts...)
}

// WriteEntry 批量写入的文件条目。
type WriteEntry struct {
	Path string
	Data []byte
}

// WriteFiles 批量写入多个文件。通过单次 multipart POST 请求上传。
// 单文件时 path 放在 query param（兼容现有 Write 行为）；
// 多文件时每个 part 的 filename 为完整路径，服务端从 part filename 获取路径。
// 上传成功后逐个调用 GetInfo 获取文件元信息返回。
//
// 注意: 当前实现会对每个文件额外发起一次 GetInfo 请求（N+1），
// 大批量上传时需注意延迟，适合小批量场景。
func (fs *Filesystem) WriteFiles(ctx context.Context, files []WriteEntry, opts ...FilesystemOption) ([]*EntryInfo, error) {
	if len(files) == 0 {
		return nil, nil
	}

	// 单文件回退到 Write
	if len(files) == 1 {
		info, err := fs.Write(ctx, files[0].Path, files[0].Data, opts...)
		if err != nil {
			return nil, err
		}
		return []*EntryInfo{info}, nil
	}

	o := applyFilesystemOpts(opts)
	uploadURL := fs.sandbox.batchUploadURL(o.user)

	pr, pw := io.Pipe()
	writer := newMultipartWriter(pw)

	go func() {
		for _, f := range files {
			if err := writer.writeFileFullPath("file", f.Path, f.Data); err != nil {
				pw.CloseWithError(err)
				return
			}
		}
		if err := writer.close(); err != nil {
			pw.CloseWithError(err)
			return
		}
		pw.Close()
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, pr)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.contentType())

	httpClient := fs.sandbox.client.config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload files: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{StatusCode: resp.StatusCode, Body: body}
	}

	// 逐个获取文件元信息
	infos := make([]*EntryInfo, 0, len(files))
	for _, f := range files {
		info, err := fs.GetInfo(ctx, f.Path, opts...)
		if err != nil {
			return nil, fmt.Errorf("get info for %s: %w", f.Path, err)
		}
		infos = append(infos, info)
	}
	return infos, nil
}

// List 列出目录内容。
func (fs *Filesystem) List(ctx context.Context, path string, opts ...ListOption) ([]EntryInfo, error) {
	o := applyListOpts(opts)
	req := connect.NewRequest(&filesystem.ListDirRequest{
		Path:  path,
		Depth: o.depth,
	})
	for k, vs := range envdAuthHeader(o.user) {
		for _, v := range vs {
			req.Header().Add(k, v)
		}
	}

	resp, err := fs.rpc.ListDir(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("list dir: %w", err)
	}

	entries := make([]EntryInfo, 0, len(resp.Msg.Entries))
	for _, e := range resp.Msg.Entries {
		info := entryInfoFromProto(e)
		if info == nil {
			continue
		}
		entries = append(entries, *info)
	}
	return entries, nil
}

// Exists 检查文件或目录是否存在。
func (fs *Filesystem) Exists(ctx context.Context, path string, opts ...FilesystemOption) (bool, error) {
	_, err := fs.GetInfo(ctx, path, opts...)
	if err != nil {
		if isNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetInfo 返回文件或目录的元信息。
func (fs *Filesystem) GetInfo(ctx context.Context, path string, opts ...FilesystemOption) (*EntryInfo, error) {
	o := applyFilesystemOpts(opts)
	req := connect.NewRequest(&filesystem.StatRequest{Path: path})
	for k, vs := range envdAuthHeader(o.user) {
		for _, v := range vs {
			req.Header().Add(k, v)
		}
	}

	resp, err := fs.rpc.Stat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}
	return entryInfoFromProto(resp.Msg.Entry), nil
}

// MakeDir 创建目录（包含父目录）。
func (fs *Filesystem) MakeDir(ctx context.Context, path string, opts ...FilesystemOption) (*EntryInfo, error) {
	o := applyFilesystemOpts(opts)
	req := connect.NewRequest(&filesystem.MakeDirRequest{Path: path})
	for k, vs := range envdAuthHeader(o.user) {
		for _, v := range vs {
			req.Header().Add(k, v)
		}
	}

	resp, err := fs.rpc.MakeDir(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	return entryInfoFromProto(resp.Msg.Entry), nil
}

// Remove 删除文件或目录。
func (fs *Filesystem) Remove(ctx context.Context, path string, opts ...FilesystemOption) error {
	o := applyFilesystemOpts(opts)
	req := connect.NewRequest(&filesystem.RemoveRequest{Path: path})
	for k, vs := range envdAuthHeader(o.user) {
		for _, v := range vs {
			req.Header().Add(k, v)
		}
	}

	_, err := fs.rpc.Remove(ctx, req)
	if err != nil {
		return fmt.Errorf("remove: %w", err)
	}
	return nil
}

// Rename 重命名或移动文件/目录。
func (fs *Filesystem) Rename(ctx context.Context, oldPath, newPath string, opts ...FilesystemOption) (*EntryInfo, error) {
	o := applyFilesystemOpts(opts)
	req := connect.NewRequest(&filesystem.MoveRequest{
		Source:      oldPath,
		Destination: newPath,
	})
	for k, vs := range envdAuthHeader(o.user) {
		for _, v := range vs {
			req.Header().Add(k, v)
		}
	}

	resp, err := fs.rpc.Move(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("move: %w", err)
	}
	return entryInfoFromProto(resp.Msg.Entry), nil
}

// WatchDir 监听目录变更。返回 WatchHandle 用于接收事件和停止监听。
func (fs *Filesystem) WatchDir(ctx context.Context, path string, opts ...WatchOption) (*WatchHandle, error) {
	o := applyWatchOpts(opts)

	watchCtx, cancel := context.WithCancel(ctx)
	req := connect.NewRequest(&filesystem.WatchDirRequest{
		Path:      path,
		Recursive: o.recursive,
	})
	for k, vs := range envdAuthHeader(o.user) {
		for _, v := range vs {
			req.Header().Add(k, v)
		}
	}

	stream, err := fs.rpc.WatchDir(watchCtx, req)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("watch dir: %w", err)
	}

	w := &WatchHandle{
		events: make(chan FilesystemEvent, 64),
		cancel: cancel,
		done:   make(chan struct{}),
	}

	go func() {
		defer close(w.done)
		defer close(w.events)
		for stream.Receive() {
			msg := stream.Msg()
			if fsEvent := msg.GetFilesystem(); fsEvent != nil {
				ev := filesystemEventFromProto(fsEvent)
				select {
				case w.events <- ev:
				case <-watchCtx.Done():
					return
				}
			}
		}
		if err := stream.Err(); err != nil && watchCtx.Err() == nil {
			w.err = err
		}
	}()

	return w, nil
}
