//go:build unit

package sandbox

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/qiniu/go-sdk/v7/sandbox/internal/apis"
	"github.com/qiniu/go-sdk/v7/sandbox/internal/envdapi/process"
)

func TestListPreservesFiltersAndCursor(t *testing.T) {
	metadata := "purpose=test&owner=alice"
	templates := []string{"t1", "t2"}
	states := []SandboxState{StateRunning, StatePaused}
	limit := int32(1)
	calls := 0
	client := newTestClient(&mockAPI{
		listSandboxesV2Fn: func(_ context.Context, params *apis.ListSandboxesV2Params, _ ...apis.RequestEditorFn) (*apis.ListSandboxesV2Response, error) {
			calls++
			require.NotNil(t, params)
			assert.Equal(t, &metadata, params.Metadata)
			assert.Equal(t, &templates, params.Template)
			assert.Equal(t, &limit, params.Limit)
			require.NotNil(t, params.State)
			assert.Equal(t, []apis.SandboxState{apis.Running, apis.Paused}, *params.State)
			response := httpResponse(http.StatusOK)
			items := []apis.ListedSandbox{{SandboxID: "second"}}
			if calls == 1 {
				assert.Nil(t, params.NextToken)
				response.Header.Set("X-Next-Token", "page2")
				items[0].SandboxID = "first"
			} else {
				assert.Equal(t, "page2", *params.NextToken)
			}
			return &apis.ListSandboxesV2Response{JSON200: &items, HTTPResponse: response}, nil
		},
	})
	params := &ListParams{Metadata: &metadata, Template: &templates, State: &states, Limit: &limit}
	first, err := client.List(context.Background(), params)
	require.NoError(t, err)
	require.Len(t, first.Items, 1)
	assert.Equal(t, "first", first.Items[0].SandboxID)
	assert.Equal(t, "page2", first.NextToken)
	assert.Nil(t, params.NextToken, "List must not advance the caller's cursor")
	params.NextToken = &first.NextToken
	last, err := client.List(context.Background(), params)
	require.NoError(t, err)
	require.Len(t, last.Items, 1)
	assert.Equal(t, "second", last.Items[0].SandboxID)
	assert.Empty(t, last.NextToken)
	assert.Equal(t, 2, calls)
}

func TestListEmptyPage(t *testing.T) {
	client := newTestClient(&mockAPI{
		listSandboxesV2Fn: func(_ context.Context, params *apis.ListSandboxesV2Params, _ ...apis.RequestEditorFn) (*apis.ListSandboxesV2Response, error) {
			assert.Nil(t, params)
			items := []apis.ListedSandbox{}
			return &apis.ListSandboxesV2Response{JSON200: &items, HTTPResponse: httpResponse(http.StatusOK)}, nil
		},
	})
	page, err := client.List(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, page.Items)
	assert.Empty(t, page.NextToken)
}

func TestListErrors(t *testing.T) {
	transportErr := errors.New("transport failed")
	for _, tc := range []struct {
		name     string
		response *apis.ListSandboxesV2Response
		err      error
	}{
		{name: "transport", err: transportErr},
		{name: "api", response: &apis.ListSandboxesV2Response{HTTPResponse: httpResponse(http.StatusForbidden), Body: []byte(`{"message":"forbidden"}`)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestClient(&mockAPI{listSandboxesV2Fn: func(context.Context, *apis.ListSandboxesV2Params, ...apis.RequestEditorFn) (*apis.ListSandboxesV2Response, error) {
				return tc.response, tc.err
			}})
			page, err := client.List(context.Background(), nil)
			assert.Nil(t, page)
			require.Error(t, err)
			if tc.err != nil {
				assert.ErrorIs(t, err, transportErr)
			} else {
				var apiErr *APIError
				require.ErrorAs(t, err, &apiErr)
				assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
			}
		})
	}
}

func TestTrafficAccessTokenFromCreateAndConnect(t *testing.T) {
	token, envdToken := "traffic-token", "envd-token"
	for _, tc := range []struct {
		name  string
		token *string
	}{
		{name: "provided", token: &token},
		{name: "absent"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			apiSandbox := &apis.Sandbox{SandboxID: "sb-test", EnvdAccessToken: &envdToken, TrafficAccessToken: tc.token}
			client := newTestClient(&mockAPI{
				createSandboxFn: func(context.Context, *apis.CreateSandboxParams, apis.CreateSandboxJSONRequestBody, ...apis.RequestEditorFn) (*apis.CreateSandboxResponse, error) {
					return &apis.CreateSandboxResponse{JSON201: apiSandbox, HTTPResponse: httpResponse(http.StatusCreated)}, nil
				},
				connectSandboxFn: func(context.Context, apis.SandboxID, apis.ConnectSandboxV2JSONRequestBody, ...apis.RequestEditorFn) (*apis.ConnectSandboxV2Response, error) {
					return &apis.ConnectSandboxV2Response{JSON201: apiSandbox, HTTPResponse: httpResponse(http.StatusCreated)}, nil
				},
			})
			want := ""
			if tc.token != nil {
				want = *tc.token
			}
			created, err := client.Create(context.Background(), CreateParams{TemplateID: "base"})
			require.NoError(t, err)
			assert.Equal(t, want, created.TrafficAccessToken())
			connected, err := client.Connect(context.Background(), "sb-test", ConnectParams{})
			require.NoError(t, err)
			assert.Equal(t, want, connected.TrafficAccessToken())
		})
	}
}

func connectedPtyData(data []byte) *process.ConnectResponse {
	return &process.ConnectResponse{Event: &process.ProcessEvent{
		Event: &process.ProcessEvent_Data{Data: &process.ProcessEvent_DataEvent{
			Output: &process.ProcessEvent_DataEvent_Pty{Pty: data},
		}},
	}}
}

func connectedProcessEnd(code int32) *process.ConnectResponse {
	return &process.ConnectResponse{Event: &process.ProcessEvent{
		Event: &process.ProcessEvent_End{End: &process.ProcessEvent_EndEvent{ExitCode: code, Exited: true}},
	}}
}

func TestPtyConnectCapturesOutput(t *testing.T) {
	want := []byte{'a', 0, 0xff, '\n'}
	pty, server := newTestPty(&testProcessHandler{
		connectFn: func(_ context.Context, req *connect.Request[process.ConnectRequest], stream *connect.ServerStream[process.ConnectResponse]) error {
			assert.Equal(t, uint32(42), req.Msg.GetProcess().GetPid())
			if err := stream.Send(connectedPtyData(want)); err != nil {
				return err
			}
			return stream.Send(connectedProcessEnd(7))
		},
	})
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	handle, err := pty.Connect(ctx, 42)
	require.NoError(t, err)
	result, err := handle.WaitContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, want, []byte(result.Stdout))
	assert.Equal(t, 7, result.ExitCode)
}

func TestPtyConnectWithOptionsStreamsRawBytes(t *testing.T) {
	first := []byte{'a', 0xff, 0}
	second := []byte{'b', '\n'}
	release := make(chan struct{})
	pty, server := newTestPty(&testProcessHandler{
		connectFn: func(ctx context.Context, req *connect.Request[process.ConnectRequest], stream *connect.ServerStream[process.ConnectResponse]) error {
			assert.Equal(t, envdBasicAuth("root"), req.Header().Get("Authorization"))
			if err := stream.Send(connectedPtyData(first)); err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-release:
			}
			if err := stream.Send(connectedPtyData(second)); err != nil {
				return err
			}
			return stream.Send(connectedProcessEnd(0))
		},
	})
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	received := make(chan []byte, 2)
	handle, err := pty.ConnectWithOptions(ctx, 42, &PtyConnectOptions{User: "root", OnData: func(data []byte) {
		received <- bytes.Clone(data)
		clear(data)
	}})
	require.NoError(t, err)
	select {
	case data := <-received:
		assert.Equal(t, first, data)
	case <-ctx.Done():
		t.Fatal("PTY callback did not run before the end event")
	}
	close(release)
	result, err := handle.WaitContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, append(bytes.Clone(first), second...), []byte(result.Stdout), "callback mutations must not change captured output")
	assert.Equal(t, second, <-received)
}

func TestPtyConnectDisconnectPreservesOutputAndAllowsReconnect(t *testing.T) {
	canceled := make(chan struct{})
	var calls atomic.Int32
	pty, server := newTestPty(&testProcessHandler{
		connectFn: func(ctx context.Context, _ *connect.Request[process.ConnectRequest], stream *connect.ServerStream[process.ConnectResponse]) error {
			if calls.Add(1) == 1 {
				if err := stream.Send(connectedPtyData([]byte("first"))); err != nil {
					return err
				}
				<-ctx.Done()
				close(canceled)
				return ctx.Err()
			}
			if err := stream.Send(connectedPtyData([]byte("reconnected"))); err != nil {
				return err
			}
			return stream.Send(connectedProcessEnd(0))
		},
		sendSignalFn: func(context.Context, *connect.Request[process.SendSignalRequest]) (*connect.Response[process.SendSignalResponse], error) {
			t.Error("disconnect must not kill the PTY")
			return connect.NewResponse(&process.SendSignalResponse{}), nil
		},
	})
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	callbackStarted := make(chan struct{})
	releaseCallback := make(chan struct{})
	defer close(releaseCallback)
	handle, err := pty.ConnectWithOptions(ctx, 42, &PtyConnectOptions{OnData: func([]byte) {
		close(callbackStarted)
		<-releaseCallback
	}})
	require.NoError(t, err)
	select {
	case <-callbackStarted:
	case <-ctx.Done():
		t.Fatal("PTY output did not arrive")
	}
	type waitResult struct {
		result *CommandResult
		err    error
	}
	waited := make(chan waitResult, 1)
	go func() {
		result, err := handle.Wait()
		waited <- waitResult{result, err}
	}()

	handle.Disconnect()
	handle.Disconnect()
	var result *CommandResult
	select {
	case got := <-waited:
		require.NoError(t, got.err)
		result = got.result
	case <-ctx.Done():
		t.Fatal("Wait remained blocked by the PTY callback after Disconnect")
	}
	require.NotNil(t, result)
	repeated, err := handle.WaitContext(ctx)
	require.NoError(t, err)
	assert.Same(t, result, repeated)
	assert.Equal(t, "first", result.Stdout)
	assert.Equal(t, -1, result.ExitCode)
	assert.NotEmpty(t, result.Error)
	select {
	case <-canceled:
	case <-ctx.Done():
		t.Fatal("disconnect did not cancel the stream")
	}
	reconnected, err := pty.Connect(ctx, 42)
	require.NoError(t, err)
	result, err = reconnected.WaitContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, "reconnected", result.Stdout)
	assert.Equal(t, 0, result.ExitCode)
}

func TestPtyCreateDisconnectWhileCallbackBlocked(t *testing.T) {
	pty, server := newTestPty(&testProcessHandler{
		startFn: func(ctx context.Context, _ *connect.Request[process.StartRequest], stream *connect.ServerStream[process.StartResponse]) error {
			if err := stream.Send(&process.StartResponse{Event: connectedPtyData([]byte("first")).Event}); err != nil {
				return err
			}
			<-ctx.Done()
			return ctx.Err()
		},
	})
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	callbackStarted := make(chan struct{})
	releaseCallback := make(chan struct{})
	defer close(releaseCallback)

	handle, err := pty.Create(ctx, PtySize{Cols: 80, Rows: 24}, WithOnPtyData(func([]byte) {
		close(callbackStarted)
		<-releaseCallback
	}))
	require.NoError(t, err)
	select {
	case <-callbackStarted:
	case <-ctx.Done():
		t.Fatal("PTY output did not arrive")
	}

	handle.Disconnect()
	result, err := handle.WaitContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, "first", result.Stdout)
	assert.Equal(t, -1, result.ExitCode)
}

func TestPtyConnectContextCancellation(t *testing.T) {
	canceled := make(chan struct{})
	pty, server := newTestPty(&testProcessHandler{
		connectFn: func(ctx context.Context, _ *connect.Request[process.ConnectRequest], stream *connect.ServerStream[process.ConnectResponse]) error {
			if err := stream.Send(connectedPtyData([]byte("partial"))); err != nil {
				return err
			}
			<-ctx.Done()
			close(canceled)
			return ctx.Err()
		},
		sendSignalFn: func(context.Context, *connect.Request[process.SendSignalRequest]) (*connect.Response[process.SendSignalResponse], error) {
			t.Error("context cancellation must not kill the PTY")
			return connect.NewResponse(&process.SendSignalResponse{}), nil
		},
	})
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()
	received := make(chan struct{}, 1)
	handle, err := pty.ConnectWithOptions(streamCtx, 42, &PtyConnectOptions{OnData: func([]byte) { received <- struct{}{} }})
	require.NoError(t, err)
	select {
	case <-received:
	case <-ctx.Done():
		t.Fatal("PTY output did not arrive")
	}
	streamCancel()
	result, err := handle.WaitContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, "partial", result.Stdout)
	assert.Equal(t, -1, result.ExitCode)
	assert.NotEmpty(t, result.Error)
	select {
	case <-canceled:
	case <-ctx.Done():
		t.Fatal("context cancellation did not close the stream")
	}
}
