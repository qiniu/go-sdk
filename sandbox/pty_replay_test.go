//go:build unit

package sandbox

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/qiniu/go-sdk/v7/sandbox/internal/envdapi/process"
)

func TestReplayRejectsOldEnvdBeforeCreating(t *testing.T) {
	starts := 0
	p, server := newTestPty(&testProcessHandler{
		listFn: func(context.Context, *connect.Request[process.ListRequest]) (*connect.Response[process.ListResponse], error) {
			return connect.NewResponse(&process.ListResponse{}), nil
		},
		startFn: func(context.Context, *connect.Request[process.StartRequest], *connect.ServerStream[process.StartResponse]) error {
			starts++
			return nil
		},
	})
	defer server.Close()
	_, err := p.CreateReplay(context.Background(), PtySize{Cols: 80, Rows: 24})
	if !errors.Is(err, ErrPtyReplayUnsupported) || starts != 0 {
		t.Fatalf("err=%v starts=%d", err, starts)
	}
}
func TestReplayCursorAndGenerationOnWire(t *testing.T) {
	ref := PtyReference{PID: 42, StreamID: "unique-generation"}
	cursor := uint64(5)
	check := func(selector *process.ProcessSelector) {
		t.Helper()
		if selector.GetPid() != ref.PID || selector.StreamId != ref.StreamID {
			t.Errorf("wrong selector: %v", selector)
		}
	}
	p, server := newTestPty(&testProcessHandler{
		listFn: func(context.Context, *connect.Request[process.ListRequest]) (*connect.Response[process.ListResponse], error) {
			return connect.NewResponse(&process.ListResponse{PtyReplaySupported: true}), nil
		},
		connectFn: func(_ context.Context, req *connect.Request[process.ConnectRequest], stream *connect.ServerStream[process.ConnectResponse]) error {
			check(req.Msg.Process)
			if req.Msg.AfterOffset == nil || *req.Msg.AfterOffset != cursor || req.Header().Get(keepalivePingHeader) != "5" {
				t.Error("missing cursor or bounded heartbeat")
			}
			if err := stream.Send(&process.ConnectResponse{Event: &process.ProcessEvent{Event: &process.ProcessEvent_Start{Start: &process.ProcessEvent_StartEvent{Pid: ref.PID, StreamId: ref.StreamID, OutputOffset: cursor}}}}); err != nil {
				return err
			}
			return stream.Send(&process.ConnectResponse{Event: &process.ProcessEvent{Event: &process.ProcessEvent_Data{Data: &process.ProcessEvent_DataEvent{Output: &process.ProcessEvent_DataEvent_Pty{Pty: []byte("next")}, OutputOffset: 9}}}})
		},
		sendInputFn: func(_ context.Context, req *connect.Request[process.SendInputRequest]) (*connect.Response[process.SendInputResponse], error) {
			check(req.Msg.Process)
			return connect.NewResponse(&process.SendInputResponse{}), nil
		},
		updateFn: func(_ context.Context, req *connect.Request[process.UpdateRequest]) (*connect.Response[process.UpdateResponse], error) {
			check(req.Msg.Process)
			return connect.NewResponse(&process.UpdateResponse{}), nil
		},
		sendSignalFn: func(_ context.Context, req *connect.Request[process.SendSignalRequest]) (*connect.Response[process.SendSignalResponse], error) {
			check(req.Msg.Process)
			return connect.NewResponse(&process.SendSignalResponse{}), nil
		},
	})
	defer server.Close()
	p.sandbox.client.config.HTTPClient = &http.Client{Transport: &rewriteTransport{base: http.DefaultTransport, baseURL: server.URL}}
	p = p.sandbox.Pty()
	stream, err := p.ConnectReplay(context.Background(), ref, &cursor)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	event, err := stream.Receive()
	if err != nil || string(event.Data) != "next" || event.Offset != 9 {
		t.Fatalf("event=%+v err=%v", event, err)
	}
	if err := p.SendReplayInput(context.Background(), ref, []byte("ls\n")); err != nil {
		t.Fatal(err)
	}
	if err := p.ResizeReplay(context.Background(), ref, PtySize{Cols: 100, Rows: 28}); err != nil {
		t.Fatal(err)
	}
	if err := p.KillReplay(context.Background(), ref); err != nil {
		t.Fatal(err)
	}
}

func TestAttachReadsCredentialsWithoutExtendingLifetime(t *testing.T) {
	for _, status := range []string{"running", "paused"} {
		t.Run(status, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "GET" || r.URL.Path != "/sandboxes/sb" {
					t.Errorf("unexpected mutating request: %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"sandboxID":"sb","templateID":"tpl","clientID":"client","domain":"sandbox.test","state":%q,"envdAccessToken":"fixture-token"}`, status)
			}))
			defer server.Close()
			client, err := NewClient(&Config{APIKey: "fixture", Endpoint: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			sb, err := client.Attach(context.Background(), "sb")
			if status == "paused" {
				if err == nil {
					t.Fatal("paused sandbox was accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			req := connect.NewRequest(&process.ListRequest{})
			sb.setEnvdAuth(req, DefaultUser)
			if req.Header().Get("X-Access-Token") != "fixture-token" || calls != 1 {
				t.Fatal("Attach did not use the saved token from a single read")
			}
		})
	}
}

func TestReplayMutationsRejectUnsupportedOrUnavailableCapability(t *testing.T) {
	for _, unavailable := range []bool{false, true} {
		t.Run(fmt.Sprint(unavailable), func(t *testing.T) {
			mutations := 0
			p, server := newTestPty(&testProcessHandler{
				listFn: func(context.Context, *connect.Request[process.ListRequest]) (*connect.Response[process.ListResponse], error) {
					if unavailable {
						return nil, connect.NewError(connect.CodeUnavailable, errors.New("unavailable"))
					}
					return connect.NewResponse(&process.ListResponse{}), nil
				},
				sendInputFn: func(context.Context, *connect.Request[process.SendInputRequest]) (*connect.Response[process.SendInputResponse], error) {
					mutations++
					return connect.NewResponse(&process.SendInputResponse{}), nil
				},
				updateFn: func(context.Context, *connect.Request[process.UpdateRequest]) (*connect.Response[process.UpdateResponse], error) {
					mutations++
					return connect.NewResponse(&process.UpdateResponse{}), nil
				},
				sendSignalFn: func(context.Context, *connect.Request[process.SendSignalRequest]) (*connect.Response[process.SendSignalResponse], error) {
					mutations++
					return connect.NewResponse(&process.SendSignalResponse{}), nil
				},
			})
			defer server.Close()
			ref := PtyReference{PID: 42, StreamID: "persisted-generation"}
			for _, err := range []error{
				p.SendReplayInput(context.Background(), ref, []byte("exit\n")),
				p.ResizeReplay(context.Background(), ref, PtySize{Cols: 80, Rows: 24}),
				p.KillReplay(context.Background(), ref),
			} {
				if unavailable && connect.CodeOf(err) != connect.CodeUnavailable || !unavailable && !errors.Is(err, ErrPtyReplayUnsupported) {
					t.Fatalf("unexpected error: %v", err)
				}
			}
			if mutations != 0 {
				t.Fatalf("issued %d unsafe mutations", mutations)
			}
		})
	}
}

func TestReplayRootLifecycleAndCursor(t *testing.T) {
	ref := PtyReference{PID: 42, StreamID: "created-generation"}
	tag := "unique-create-tag"
	started := false
	closed := make(chan struct{})
	checkUser := func(header http.Header) {
		t.Helper()
		if header.Get("Authorization") != envdBasicAuth("root") {
			t.Error("replay request did not preserve root user")
		}
	}
	startEvent := func(offset uint64) *process.ProcessEvent {
		return &process.ProcessEvent{Event: &process.ProcessEvent_Start{Start: &process.ProcessEvent_StartEvent{Pid: ref.PID, StreamId: ref.StreamID, OutputOffset: offset}}}
	}
	dataEvent := func(data string, offset uint64) *process.ProcessEvent {
		return &process.ProcessEvent{Event: &process.ProcessEvent_Data{Data: &process.ProcessEvent_DataEvent{Output: &process.ProcessEvent_DataEvent_Pty{Pty: []byte(data)}, OutputOffset: offset}}}
	}
	p, server := newTestPty(&testProcessHandler{
		listFn: func(_ context.Context, req *connect.Request[process.ListRequest]) (*connect.Response[process.ListResponse], error) {
			checkUser(req.Header())
			return connect.NewResponse(&process.ListResponse{PtyReplaySupported: true, Processes: []*process.ProcessInfo{{Pid: ref.PID, StreamId: ref.StreamID, Tag: &tag}}}), nil
		},
		startFn: func(ctx context.Context, req *connect.Request[process.StartRequest], stream *connect.ServerStream[process.StartResponse]) error {
			checkUser(req.Header())
			if !req.Msg.PtyReplay || req.Msg.Pty == nil || req.Msg.GetTag() != tag {
				t.Error("missing replay PTY creation flags")
			}
			started = true
			if err := stream.Send(&process.StartResponse{Event: startEvent(0)}); err != nil {
				return err
			}
			if err := stream.Send(&process.StartResponse{Event: dataEvent("first", 5)}); err != nil {
				return err
			}
			<-ctx.Done()
			close(closed)
			return ctx.Err()
		},
		connectFn: func(_ context.Context, req *connect.Request[process.ConnectRequest], stream *connect.ServerStream[process.ConnectResponse]) error {
			checkUser(req.Header())
			if !started || req.Msg.Process.GetPid() != ref.PID || req.Msg.Process.StreamId != ref.StreamID || req.Msg.GetAfterOffset() != 5 {
				t.Error("reconnect lost creation identity or cursor")
			}
			if err := stream.Send(&process.ConnectResponse{Event: startEvent(5)}); err != nil {
				return err
			}
			return stream.Send(&process.ConnectResponse{Event: dataEvent("second", 11)})
		},
		sendInputFn: func(_ context.Context, req *connect.Request[process.SendInputRequest]) (*connect.Response[process.SendInputResponse], error) {
			checkUser(req.Header())
			return connect.NewResponse(&process.SendInputResponse{}), nil
		},
		updateFn: func(_ context.Context, req *connect.Request[process.UpdateRequest]) (*connect.Response[process.UpdateResponse], error) {
			checkUser(req.Header())
			return connect.NewResponse(&process.UpdateResponse{}), nil
		},
		sendSignalFn: func(_ context.Context, req *connect.Request[process.SendSignalRequest]) (*connect.Response[process.SendSignalResponse], error) {
			checkUser(req.Header())
			return connect.NewResponse(&process.SendSignalResponse{}), nil
		},
	})
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream, err := p.CreateReplay(ctx, PtySize{Cols: 80, Rows: 24}, WithCommandUser("root"), WithTag(tag))
	if err != nil {
		t.Fatal(err)
	}
	event, err := stream.Receive()
	if err != nil || string(event.Data) != "first" {
		t.Fatalf("first output: %+v %v", event, err)
	}
	cursor := event.Offset
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-closed:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	refs, err := p.FindReplay(ctx, tag, WithCommandUser("root"))
	if err != nil || len(refs) != 1 || refs[0] != ref {
		t.Fatalf("find: %v %v", refs, err)
	}
	resumed, err := p.ConnectReplay(ctx, ref, &cursor, WithCommandUser("root"))
	if err != nil {
		t.Fatal(err)
	}
	defer resumed.Close()
	event, err = resumed.Receive()
	if err != nil || string(event.Data) != "second" || event.Offset != 11 {
		t.Fatalf("resumed output: %+v %v", event, err)
	}
	if err := p.SendReplayInput(ctx, ref, []byte("echo ok\n"), WithCommandUser("root")); err != nil {
		t.Fatal(err)
	}
	if err := p.ResizeReplay(ctx, ref, PtySize{Cols: 100, Rows: 30}, WithCommandUser("root")); err != nil {
		t.Fatal(err)
	}
	if err := p.KillReplay(ctx, ref, WithCommandUser("root")); err != nil {
		t.Fatal(err)
	}
}

func TestReplayInitialGapPreservesCreatedIdentity(t *testing.T) {
	p, server := newTestPty(&testProcessHandler{
		listFn: func(context.Context, *connect.Request[process.ListRequest]) (*connect.Response[process.ListResponse], error) {
			return connect.NewResponse(&process.ListResponse{PtyReplaySupported: true}), nil
		},
		startFn: func(_ context.Context, _ *connect.Request[process.StartRequest], stream *connect.ServerStream[process.StartResponse]) error {
			if err := stream.Send(&process.StartResponse{Event: &process.ProcessEvent{Event: &process.ProcessEvent_Start{Start: &process.ProcessEvent_StartEvent{Pid: 42, StreamId: "generation", ReplayStart: 100}}}}); err != nil {
				return err
			}
			return connect.NewError(connect.CodeOutOfRange, errors.New("replay gap"))
		},
	})
	defer server.Close()
	stream, err := p.CreateReplay(context.Background(), PtySize{Cols: 80, Rows: 24})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if stream.Reference.PID != 42 || stream.Reference.StreamID != "generation" || stream.Offset != 0 || stream.ReplayStart != 100 {
		t.Fatalf("lost creation identity or cursor: %+v", stream)
	}
	if _, err := stream.Receive(); connect.CodeOf(err) != connect.CodeOutOfRange {
		t.Fatalf("expected replay gap, got %v", err)
	}
}
