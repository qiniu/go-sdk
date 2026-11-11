//go:build unit

package sandbox

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

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
