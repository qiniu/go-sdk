package sandbox

import (
	"context"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"

	"github.com/qiniu/go-sdk/v7/sandbox/envdapi/process"
	"github.com/qiniu/go-sdk/v7/sandbox/envdapi/process/processconnect"
)

// CommandResult 命令执行结果。
type CommandResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Error    string
}

// CommandHandle 后台命令句柄。
type CommandHandle struct {
	PID uint32

	commands *Commands
	cancel   context.CancelFunc
	done     chan struct{}
	pidCh    chan struct{}
	result   *CommandResult

	mu        sync.Mutex
	onStdout  func(data []byte)
	onStderr  func(data []byte)
	onPtyData func(data []byte)
}

// Wait 等待命令完成并返回结果。
func (h *CommandHandle) Wait() (*CommandResult, error) {
	<-h.done
	if h.result == nil {
		return nil, fmt.Errorf("command terminated without result")
	}
	return h.result, nil
}

// Kill 终止命令。
func (h *CommandHandle) Kill(ctx context.Context) error {
	return h.commands.Kill(ctx, h.PID)
}

// WaitPID 等待进程 PID 被分配。
// 当进程流收到 Start 事件后返回 PID；若 ctx 取消则返回错误。
func (h *CommandHandle) WaitPID(ctx context.Context) (uint32, error) {
	select {
	case <-h.pidCh:
		return h.PID, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

// ProcessInfo 进程信息。
type ProcessInfo struct {
	PID  uint32
	Tag  *string
	Cmd  string
	Args []string
	Envs map[string]string
	Cwd  *string
}

// CommandOption 命令选项。
type CommandOption func(*commandOpts)

type commandOpts struct {
	envs      map[string]string
	cwd       string
	user      string
	tag       string
	onStdout  func(data []byte)
	onStderr  func(data []byte)
	onPtyData func(data []byte)
	timeout   time.Duration
}

// WithEnvs 设置命令的环境变量。
func WithEnvs(envs map[string]string) CommandOption {
	return func(o *commandOpts) { o.envs = envs }
}

// WithCwd 设置命令的工作目录。
func WithCwd(cwd string) CommandOption {
	return func(o *commandOpts) { o.cwd = cwd }
}

// WithCommandUser 设置执行命令的用户。
func WithCommandUser(user string) CommandOption {
	return func(o *commandOpts) { o.user = user }
}

// WithTag 设置进程标签，用于后续通过标签连接进程。
func WithTag(tag string) CommandOption {
	return func(o *commandOpts) { o.tag = tag }
}

// WithOnStdout 设置 stdout 数据回调。仅用于标准命令的 stdout 输出。
// PTY 会话应使用 WithOnPtyData 接收输出。
func WithOnStdout(fn func(data []byte)) CommandOption {
	return func(o *commandOpts) { o.onStdout = fn }
}

// WithOnStderr 设置 stderr 数据回调。
func WithOnStderr(fn func(data []byte)) CommandOption {
	return func(o *commandOpts) { o.onStderr = fn }
}

// WithOnPtyData 设置 PTY 数据回调。用于接收 PTY 会话的输出数据。
// 若未设置，Pty.Create 会回退使用 WithOnStdout 回调以保持兼容。
func WithOnPtyData(fn func(data []byte)) CommandOption {
	return func(o *commandOpts) { o.onPtyData = fn }
}

// WithTimeout 设置命令超时时间。
func WithTimeout(timeout time.Duration) CommandOption {
	return func(o *commandOpts) { o.timeout = timeout }
}

func applyCommandOpts(opts []CommandOption) *commandOpts {
	o := &commandOpts{user: "user"}
	for _, fn := range opts {
		fn(o)
	}
	return o
}

// Commands 提供沙箱命令执行能力。
type Commands struct {
	sandbox *Sandbox
	rpc     processconnect.ProcessClient
}

// newCommands 创建 Commands 实例。
func newCommands(s *Sandbox, rpc processconnect.ProcessClient) *Commands {
	return &Commands{sandbox: s, rpc: rpc}
}

// Run 在沙箱中执行命令并等待完成。返回执行结果。
// 注意: stdout 和 stderr 在内存中累积，长时间运行或大量输出的命令
// 应使用 Start() + WithOnStdout/WithOnStderr 流式回调处理输出。
func (c *Commands) Run(ctx context.Context, cmd string, opts ...CommandOption) (*CommandResult, error) {
	handle, err := c.Start(ctx, cmd, opts...)
	if err != nil {
		return nil, err
	}
	return handle.Wait()
}

// Start 在沙箱中后台启动命令。返回 CommandHandle 可用于等待完成。
// cmd 以 /bin/bash -l -c <cmd> 形式执行，支持 shell 语法（管道、重定向等），
// 会加载登录 shell 环境（/etc/profile 及用户 profile）。
func (c *Commands) Start(ctx context.Context, cmd string, opts ...CommandOption) (*CommandHandle, error) {
	o := applyCommandOpts(opts)

	cmdCtx := ctx
	var cmdCancel context.CancelFunc
	if o.timeout > 0 {
		cmdCtx, cmdCancel = context.WithTimeout(ctx, o.timeout)
	} else {
		cmdCtx, cmdCancel = context.WithCancel(ctx)
	}

	startReq := &process.StartRequest{
		Process: &process.ProcessConfig{
			Cmd:  "/bin/bash",
			Args: []string{"-l", "-c", cmd},
			Envs: o.envs,
		},
	}
	if o.cwd != "" {
		startReq.Process.Cwd = &o.cwd
	}
	if o.tag != "" {
		startReq.Tag = &o.tag
	}
	// 默认不启用 stdin
	stdinEnabled := false
	startReq.Stdin = &stdinEnabled

	req := connect.NewRequest(startReq)
	for k, vs := range envdAuthHeader(o.user) {
		for _, v := range vs {
			req.Header().Add(k, v)
		}
	}

	stream, err := c.rpc.Start(cmdCtx, req)
	if err != nil {
		cmdCancel()
		return nil, fmt.Errorf("start command: %w", err)
	}

	handle := &CommandHandle{
		commands: c,
		cancel:   cmdCancel,
		done:     make(chan struct{}),
		pidCh:    make(chan struct{}),
		onStdout: o.onStdout,
		onStderr: o.onStderr,
	}

	go processEventStream(stream, handle)

	return handle, nil
}

// eventMessage 是 StartResponse 和 ConnectResponse 的公共接口。
type eventMessage interface {
	GetEvent() *process.ProcessEvent
}

// streamReceiver 抽象 ConnectRPC 服务端流的读取操作。
type streamReceiver[T eventMessage] interface {
	Receive() bool
	Msg() T
	Err() error
}

// processEventStream 处理进程事件流（Start 和 Connect 共用）。
func processEventStream[T eventMessage](stream streamReceiver[T], handle *CommandHandle) {
	defer close(handle.done)

	var stdout, stderr []byte
	for stream.Receive() {
		event := stream.Msg().GetEvent()
		if event == nil {
			continue
		}
		switch ev := event.Event.(type) {
		case *process.ProcessEvent_Start:
			handle.PID = ev.Start.Pid
			close(handle.pidCh)
		case *process.ProcessEvent_Data:
			if data := ev.Data.GetStdout(); len(data) > 0 {
				stdout = append(stdout, data...)
				handle.mu.Lock()
				fn := handle.onStdout
				handle.mu.Unlock()
				if fn != nil {
					fn(data)
				}
			}
			if data := ev.Data.GetStderr(); len(data) > 0 {
				stderr = append(stderr, data...)
				handle.mu.Lock()
				fn := handle.onStderr
				handle.mu.Unlock()
				if fn != nil {
					fn(data)
				}
			}
			if data := ev.Data.GetPty(); len(data) > 0 {
				handle.mu.Lock()
				fn := handle.onPtyData
				handle.mu.Unlock()
				if fn != nil {
					fn(data)
				}
			}
		case *process.ProcessEvent_End:
			handle.result = &CommandResult{
				ExitCode: int(ev.End.ExitCode),
				Stdout:   string(stdout),
				Stderr:   string(stderr),
			}
			if ev.End.Error != nil {
				handle.result.Error = *ev.End.Error
			}
		}
	}

	// 如果流结束但没有收到 EndEvent，创建一个错误结果
	if handle.result == nil {
		errMsg := ""
		if err := stream.Err(); err != nil {
			errMsg = err.Error()
		}
		handle.result = &CommandResult{
			ExitCode: -1,
			Stdout:   string(stdout),
			Stderr:   string(stderr),
			Error:    errMsg,
		}
	}
}

// Connect 连接到正在运行的进程。
func (c *Commands) Connect(ctx context.Context, pid uint32) (*CommandHandle, error) {
	connectCtx, connectCancel := context.WithCancel(ctx)

	req := connect.NewRequest(&process.ConnectRequest{
		Process: &process.ProcessSelector{
			Selector: &process.ProcessSelector_Pid{Pid: pid},
		},
	})
	for k, vs := range envdAuthHeader("user") {
		for _, v := range vs {
			req.Header().Add(k, v)
		}
	}

	stream, err := c.rpc.Connect(connectCtx, req)
	if err != nil {
		connectCancel()
		return nil, fmt.Errorf("connect to process: %w", err)
	}

	pidCh := make(chan struct{})
	close(pidCh) // PID 已知，无需等待

	handle := &CommandHandle{
		PID:      pid,
		commands: c,
		cancel:   connectCancel,
		done:     make(chan struct{}),
		pidCh:    pidCh,
	}

	go processEventStream(stream, handle)

	return handle, nil
}

// List 列出所有运行中的进程。
func (c *Commands) List(ctx context.Context) ([]ProcessInfo, error) {
	req := connect.NewRequest(&process.ListRequest{})
	for k, vs := range envdAuthHeader("user") {
		for _, v := range vs {
			req.Header().Add(k, v)
		}
	}

	resp, err := c.rpc.List(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("list processes: %w", err)
	}

	var infos []ProcessInfo
	for _, p := range resp.Msg.Processes {
		info := ProcessInfo{
			PID: p.Pid,
			Tag: p.Tag,
		}
		if p.Config != nil {
			info.Cmd = p.Config.Cmd
			info.Args = p.Config.Args
			info.Envs = p.Config.Envs
			info.Cwd = p.Config.Cwd
		}
		infos = append(infos, info)
	}
	return infos, nil
}

// SendStdin 向进程发送标准输入。
func (c *Commands) SendStdin(ctx context.Context, pid uint32, data []byte) error {
	req := connect.NewRequest(&process.SendInputRequest{
		Process: &process.ProcessSelector{
			Selector: &process.ProcessSelector_Pid{Pid: pid},
		},
		Input: &process.ProcessInput{
			Input: &process.ProcessInput_Stdin{Stdin: data},
		},
	})
	for k, vs := range envdAuthHeader("user") {
		for _, v := range vs {
			req.Header().Add(k, v)
		}
	}

	_, err := c.rpc.SendInput(ctx, req)
	if err != nil {
		return fmt.Errorf("send stdin: %w", err)
	}
	return nil
}

// Kill 终止指定进程。
func (c *Commands) Kill(ctx context.Context, pid uint32) error {
	req := connect.NewRequest(&process.SendSignalRequest{
		Process: &process.ProcessSelector{
			Selector: &process.ProcessSelector_Pid{Pid: pid},
		},
		Signal: process.Signal_SIGNAL_SIGKILL,
	})
	for k, vs := range envdAuthHeader("user") {
		for _, v := range vs {
			req.Header().Add(k, v)
		}
	}

	_, err := c.rpc.SendSignal(ctx, req)
	if err != nil {
		return fmt.Errorf("kill process: %w", err)
	}
	return nil
}
