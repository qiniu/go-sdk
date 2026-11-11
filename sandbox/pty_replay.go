package sandbox

import (
	"context"
	"errors"
	"io"

	"connectrpc.com/connect"
	"github.com/qiniu/go-sdk/v7/sandbox/internal/envdapi/process"
)

// ErrPtyReplayUnsupported 表示 Sandbox 的 envd 尚未支持可重放的 PTY。
var ErrPtyReplayUnsupported = errors.New("sandbox envd does not support PTY replay")

// PtyReference 标识一个 PTY 实例；StreamID 防止 PID 重用后操作到其他进程。
type PtyReference struct {
	PID      uint32
	StreamID string
}

// PtyReplayEvent 是按字节偏移标识的输出、心跳或退出事件。
type PtyReplayEvent struct {
	Data      []byte
	Offset    uint64
	Heartbeat bool
	End       *CommandResult
}

// PtyReplayStream 只持有当前 HTTP 订阅；关闭订阅不会终止 Sandbox 中的 PTY。
// Receive 不支持并发调用。Offset 是首个输出字节之前的位置。
type PtyReplayStream struct {
	Reference   PtyReference
	Offset      uint64
	ReplayStart uint64
	receive     func() (*process.ProcessEvent, error)
	close       func() error
}

// Close 断开订阅，保留 PTY 供后续重连。
func (s *PtyReplayStream) Close() error { return s.close() }

// Receive 接收一个事件；服务端明确报告重放缺口，调用方不得静默跳过。
func (s *PtyReplayStream) Receive() (PtyReplayEvent, error) {
	event, err := s.receive()
	if err != nil {
		return PtyReplayEvent{}, err
	}
	if event == nil {
		return PtyReplayEvent{}, errors.New("missing PTY replay event")
	}
	switch e := event.Event.(type) {
	case *process.ProcessEvent_Data:
		if _, ok := e.Data.GetOutput().(*process.ProcessEvent_DataEvent_Pty); !ok {
			return PtyReplayEvent{}, errors.New("unexpected PTY output type")
		}
		data := e.Data.GetPty()
		if e.Data.OutputOffset < s.Offset || e.Data.OutputOffset != s.Offset+uint64(len(data)) {
			return PtyReplayEvent{}, errors.New("invalid PTY replay offset")
		}
		s.Offset = e.Data.OutputOffset
		return PtyReplayEvent{Data: data, Offset: s.Offset}, nil
	case *process.ProcessEvent_Keepalive:
		return PtyReplayEvent{Heartbeat: true, Offset: s.Offset}, nil
	case *process.ProcessEvent_End:
		return PtyReplayEvent{Offset: s.Offset, End: &CommandResult{ExitCode: int(e.End.ExitCode), Error: e.End.GetError()}}, nil
	default:
		return PtyReplayEvent{}, errors.New("unexpected PTY replay event")
	}
}

func newPtyReplayStream(receive func() (*process.ProcessEvent, error), closeFn func() error) (*PtyReplayStream, error) {
	event, err := receive()
	if err != nil {
		_ = closeFn()
		return nil, err
	}
	start := event.GetStart()
	if start == nil || start.Pid == 0 || start.StreamId == "" || start.OutputOffset < start.ReplayStart {
		_ = closeFn()
		return nil, ErrPtyReplayUnsupported
	}
	return &PtyReplayStream{Reference: PtyReference{PID: start.Pid, StreamID: start.StreamId}, Offset: start.OutputOffset, ReplayStart: start.ReplayStart, receive: receive, close: closeFn}, nil
}

// SupportsReplay 查询当前 Sandbox 的能力，不使用版本号推断或本地缓存。
func (p *Pty) SupportsReplay(ctx context.Context) (bool, error) {
	req := connect.NewRequest(&process.ListRequest{})
	p.sandbox.setEnvdAuth(req, DefaultUser)
	resp, err := p.rpc.List(ctx, req)
	if err != nil {
		return false, err
	}
	return resp.Msg.PtyReplaySupported, nil
}

// CreateReplay 创建支持字节偏移重连的 PTY。旧 envd 会在创建前被拒绝。
// 输出通过返回的流读取；不会调用 CommandOption 中的输出回调。
func (p *Pty) CreateReplay(ctx context.Context, size PtySize, opts ...CommandOption) (*PtyReplayStream, error) {
	supported, err := p.SupportsReplay(ctx)
	if err != nil {
		return nil, err
	}
	if !supported {
		return nil, ErrPtyReplayUnsupported
	}
	o := applyCommandOpts(opts)
	envs := map[string]string{"TERM": "xterm", "LANG": "C.UTF-8", "LC_ALL": "C.UTF-8"}
	for k, v := range o.envs {
		envs[k] = v
	}
	msg := &process.StartRequest{PtyReplay: true, Process: &process.ProcessConfig{Cmd: "/bin/bash", Args: []string{"-i", "-l"}, Envs: envs}, Pty: &process.PTY{Size: &process.PTY_Size{Cols: size.Cols, Rows: size.Rows}}}
	if o.cwd != "" {
		msg.Process.Cwd = &o.cwd
	}
	if o.tag != "" {
		msg.Tag = &o.tag
	}
	req := connect.NewRequest(msg)
	p.sandbox.setEnvdAuth(req, o.user)
	req.Header().Set(keepalivePingHeader, "5")
	stream, err := p.rpc.Start(context.WithValue(ctx, keepaliveIntervalKey{}, "5"), req)
	if err != nil {
		return nil, err
	}
	return newPtyReplayStream(func() (*process.ProcessEvent, error) {
		if stream.Receive() {
			return stream.Msg().Event, nil
		}
		if err := stream.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}, stream.Close)
}

func replaySelector(ref PtyReference) (*process.ProcessSelector, error) {
	if ref.PID == 0 || ref.StreamID == "" {
		return nil, errors.New("PTY replay requires PID and stream identity")
	}
	return &process.ProcessSelector{Selector: &process.ProcessSelector_Pid{Pid: ref.PID}, StreamId: ref.StreamID}, nil
}

// ConnectReplay 从指定字节偏移续接同一个 PTY。nil 从当前保留输出的起点开始。
func (p *Pty) ConnectReplay(ctx context.Context, ref PtyReference, after *uint64) (*PtyReplayStream, error) {
	selector, err := replaySelector(ref)
	if err != nil {
		return nil, err
	}
	req := connect.NewRequest(&process.ConnectRequest{Process: selector, AfterOffset: after})
	p.sandbox.setEnvdAuth(req, DefaultUser)
	req.Header().Set(keepalivePingHeader, "5")
	stream, err := p.rpc.Connect(context.WithValue(ctx, keepaliveIntervalKey{}, "5"), req)
	if err != nil {
		return nil, err
	}
	result, err := newPtyReplayStream(func() (*process.ProcessEvent, error) {
		if stream.Receive() {
			return stream.Msg().Event, nil
		}
		if err := stream.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}, stream.Close)
	if err == nil && (result.Reference != ref || (after != nil && result.Offset != *after)) {
		_ = result.Close()
		return nil, errors.New("PTY replay identity or cursor mismatch")
	}
	return result, err
}

// SendReplayInput 向准确的 PTY 实例发送输入；调用方不得自动重试未知结果。
func (p *Pty) SendReplayInput(ctx context.Context, ref PtyReference, data []byte) error {
	selector, err := replaySelector(ref)
	if err != nil {
		return err
	}
	req := connect.NewRequest(&process.SendInputRequest{Process: selector, Input: &process.ProcessInput{Input: &process.ProcessInput_Pty{Pty: data}}})
	p.sandbox.setEnvdAuth(req, DefaultUser)
	_, err = p.rpc.SendInput(ctx, req)
	return err
}

// ResizeReplay 调整准确的 PTY 实例的窗口大小。
func (p *Pty) ResizeReplay(ctx context.Context, ref PtyReference, size PtySize) error {
	selector, err := replaySelector(ref)
	if err != nil {
		return err
	}
	req := connect.NewRequest(&process.UpdateRequest{Process: selector, Pty: &process.PTY{Size: &process.PTY_Size{Cols: size.Cols, Rows: size.Rows}}})
	p.sandbox.setEnvdAuth(req, DefaultUser)
	_, err = p.rpc.Update(ctx, req)
	return err
}

// KillReplay 终止准确的 PTY 实例，拒绝对重用 PID 的其他进程执行操作。
func (p *Pty) KillReplay(ctx context.Context, ref PtyReference) error {
	selector, err := replaySelector(ref)
	if err != nil {
		return err
	}
	req := connect.NewRequest(&process.SendSignalRequest{Process: selector, Signal: process.Signal_SIGNAL_SIGKILL})
	p.sandbox.setEnvdAuth(req, DefaultUser)
	_, err = p.rpc.SendSignal(ctx, req)
	return err
}

// FindReplay 根据创建前保存的唯一标签查询仍在运行的 PTY，用于恢复未知创建结果。
func (p *Pty) FindReplay(ctx context.Context, tag string) ([]PtyReference, error) {
	if tag == "" {
		return nil, errors.New("PTY tag is required")
	}
	req := connect.NewRequest(&process.ListRequest{})
	p.sandbox.setEnvdAuth(req, DefaultUser)
	resp, err := p.rpc.List(ctx, req)
	if err != nil {
		return nil, err
	}
	if !resp.Msg.PtyReplaySupported {
		return nil, ErrPtyReplayUnsupported
	}
	var refs []PtyReference
	for _, info := range resp.Msg.Processes {
		if info.GetTag() == tag && info.StreamId != "" {
			refs = append(refs, PtyReference{PID: info.Pid, StreamID: info.StreamId})
		}
	}
	return refs, nil
}
