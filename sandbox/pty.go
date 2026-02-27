package sandbox

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"

	"github.com/qiniu/go-sdk/v7/sandbox/envdapi/process"
	"github.com/qiniu/go-sdk/v7/sandbox/envdapi/process/processconnect"
)

// PtySize PTY 终端大小。
type PtySize struct {
	Cols uint32
	Rows uint32
}

// Pty 提供沙箱 PTY（伪终端）操作。
type Pty struct {
	sandbox *Sandbox
	rpc     processconnect.ProcessClient
}

// newPty 创建 Pty 实例。
func newPty(s *Sandbox) *Pty {
	httpClient := s.client.config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	rpc := processconnect.NewProcessClient(
		httpClient,
		s.envdURL(),
	)
	return &Pty{sandbox: s, rpc: rpc}
}

// Create 创建一个 PTY 终端会话。
func (p *Pty) Create(ctx context.Context, size PtySize, opts ...CommandOption) (*CommandHandle, error) {
	o := applyCommandOpts(opts)

	ptyCtx, ptyCancel := context.WithCancel(ctx)

	startReq := &process.StartRequest{
		Process: &process.ProcessConfig{
			Cmd:  "/bin/bash",
			Envs: o.envs,
		},
		Pty: &process.PTY{
			Size: &process.PTY_Size{
				Cols: size.Cols,
				Rows: size.Rows,
			},
		},
	}
	if o.cwd != "" {
		startReq.Process.Cwd = &o.cwd
	}
	if o.tag != "" {
		startReq.Tag = &o.tag
	}

	req := connect.NewRequest(startReq)
	for k, vs := range envdAuthHeader(o.user) {
		for _, v := range vs {
			req.Header().Add(k, v)
		}
	}

	stream, err := p.rpc.Start(ptyCtx, req)
	if err != nil {
		ptyCancel()
		return nil, fmt.Errorf("create pty: %w", err)
	}

	commands := &Commands{sandbox: p.sandbox, rpc: p.rpc}
	handle := &CommandHandle{
		commands:  commands,
		cancel:    ptyCancel,
		done:      make(chan struct{}),
		onPtyData: o.onStdout, // PTY 输出通过 onStdout 回调
	}

	go commands.processStream(stream, handle)

	return handle, nil
}

// Connect 连接到已有的 PTY 会话。
func (p *Pty) Connect(ctx context.Context, pid uint32) (*CommandHandle, error) {
	commands := &Commands{sandbox: p.sandbox, rpc: p.rpc}
	return commands.Connect(ctx, pid)
}

// SendInput 向 PTY 发送输入。
func (p *Pty) SendInput(ctx context.Context, pid uint32, data []byte) error {
	req := connect.NewRequest(&process.SendInputRequest{
		Process: &process.ProcessSelector{
			Selector: &process.ProcessSelector_Pid{Pid: pid},
		},
		Input: &process.ProcessInput{
			Input: &process.ProcessInput_Pty{Pty: data},
		},
	})
	for k, vs := range envdAuthHeader("user") {
		for _, v := range vs {
			req.Header().Add(k, v)
		}
	}

	_, err := p.rpc.SendInput(ctx, req)
	if err != nil {
		return fmt.Errorf("send pty input: %w", err)
	}
	return nil
}

// Resize 调整 PTY 终端大小。
func (p *Pty) Resize(ctx context.Context, pid uint32, size PtySize) error {
	req := connect.NewRequest(&process.UpdateRequest{
		Process: &process.ProcessSelector{
			Selector: &process.ProcessSelector_Pid{Pid: pid},
		},
		Pty: &process.PTY{
			Size: &process.PTY_Size{
				Cols: size.Cols,
				Rows: size.Rows,
			},
		},
	})
	for k, vs := range envdAuthHeader("user") {
		for _, v := range vs {
			req.Header().Add(k, v)
		}
	}

	_, err := p.rpc.Update(ctx, req)
	if err != nil {
		return fmt.Errorf("resize pty: %w", err)
	}
	return nil
}

// Kill 终止 PTY 会话。
func (p *Pty) Kill(ctx context.Context, pid uint32) error {
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

	_, err := p.rpc.SendSignal(ctx, req)
	if err != nil {
		return fmt.Errorf("kill pty: %w", err)
	}
	return nil
}
