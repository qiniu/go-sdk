//go:build unit

package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotRetryAndPagination(t *testing.T) {
	var key, firstBody string
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/sandboxes/sb-1/snapshots":
			calls++
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			encoded, _ := json.Marshal(body)
			if calls == 1 {
				key = r.Header.Get("Idempotency-Key")
				firstBody = string(encoded)
			}
			assert.NotEmpty(t, key)
			assert.Equal(t, key, r.Header.Get("Idempotency-Key"))
			assert.Equal(t, firstBody, string(encoded))
			assert.Equal(t, false, body["memory"])
			if calls == 1 {
				w.WriteHeader(408)
				_, _ = w.Write([]byte(`{"code":408,"error_code":"snapshot_in_progress","message":"retry"}`))
				return
			}
			w.WriteHeader(201)
			_, _ = w.Write([]byte(`{"snapshotID":"snap-1","names":["team/demo:v1"]}`))
		case "/snapshots":
			assert.Equal(t, "sb-1", r.URL.Query().Get("sandboxID"))
			assert.Equal(t, "cursor", r.URL.Query().Get("nextToken"))
			w.Header().Set("X-Next-Token", "next")
			_, _ = w.Write([]byte(`[{"snapshotID":"snap-1","names":[]}]`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	c, err := NewClient(&Config{Endpoint: server.URL, RetryBackoff: func(int) time.Duration { return 0 }})
	require.NoError(t, err)
	memory, name := false, "demo:v1"
	snap, err := c.CreateSnapshot(context.Background(), "sb-1", CreateSnapshotParams{Memory: &memory, Name: &name, IdempotencyKey: "capture-1"})
	require.NoError(t, err)
	assert.Equal(t, "snap-1", snap.SnapshotID)
	assert.Equal(t, 2, calls)
	id, cursor := "sb-1", "cursor"
	page, err := c.ListSnapshots(context.Background(), &ListSnapshotsParams{SandboxID: &id, NextToken: &cursor})
	require.NoError(t, err)
	assert.Equal(t, "next", page.NextToken)
	require.Len(t, page.Items, 1)
}

func TestSemanticAPIErrorWithNumericCode(t *testing.T) {
	e := newAPIError(&http.Response{StatusCode: 503, Header: http.Header{}}, []byte(`{"code":503,"error_code":"api_shutdown","message":"retry later"}`))
	assert.Equal(t, "api_shutdown", e.Code)
	assert.Equal(t, "retry later", e.Message)
}

func TestUpdatedControlPlaneRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var body map[string]any
		if r.Method == http.MethodPost {
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		}
		switch r.URL.Path {
		case "/v2/sandboxes/sb-1/connect":
			assert.Empty(t, body)
			_, _ = w.Write([]byte(`{"sandboxID":"sb-1","templateID":"tmpl","clientID":"c","envdAccessToken":"token","envdVersion":"1"}`))
		case "/sandboxes/sb-1/pause":
			assert.Equal(t, false, body["memory"])
			w.WriteHeader(204)
		case "/templates/tmpl/tags":
			_, _ = w.Write([]byte(`[{"tag":"v1","buildID":"00000000-0000-0000-0000-000000000001","createdAt":"2026-11-02T00:00:00Z"}]`))
		case "/v3/templates":
			assert.Equal(t, float64(0), body["minFreeDiskMb"])
			assert.NotContains(t, body, "diskSizeMB")
			w.WriteHeader(202)
			_, _ = w.Write([]byte(`{"templateID":"tmpl","buildID":"00000000-0000-0000-0000-000000000001"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	c, err := NewClient(&Config{Endpoint: server.URL})
	require.NoError(t, err)
	sb, err := c.Connect(context.Background(), "sb-1", ConnectParams{})
	require.NoError(t, err)
	memory := false
	require.NoError(t, sb.PauseWithParams(context.Background(), PauseParams{Memory: &memory}))
	tags, err := c.ListTemplateTags(context.Background(), "tmpl")
	require.NoError(t, err)
	require.Len(t, tags, 1)
	assert.Equal(t, "v1", tags[0].Tag)
	zero := int32(0)
	_, err = c.CreateTemplate(context.Background(), CreateTemplateParams{MinFreeDiskMB: &zero})
	require.NoError(t, err)
}

func TestNewCreateAndInjectionFields(t *testing.T) {
	memory := false
	body, err := (&CreateParams{TemplateID: "tmpl", AutoPauseMemory: &memory, AutoResume: &AutoResumeConfig{Enabled: false}}).toAPI()
	require.NoError(t, err)
	data, err := json.Marshal(body)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"autoPauseMemory":false`)
	assert.Contains(t, string(data), `"autoResume":{"enabled":false}`)
	conditions, secrets := map[string]string{"model": "demo"}, map[string]string{"api_key": "secret"}
	spec := &HTTPInjection{BaseURL: "https://example.com", IfBody: &conditions, Body: &secrets}
	saved, err := injectionSpecToAPI(InjectionSpec{HTTP: spec})
	require.NoError(t, err)
	data, err = json.Marshal(saved)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"body":{"api_key":"secret"}`)
	runtime, err := sandboxInjectionSpecToAPI(SandboxInjectionSpec{HTTP: spec})
	require.NoError(t, err)
	data, err = json.Marshal(runtime)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"if_body":{"model":"demo"}`)
	assert.Contains(t, string(data), `"body":{"api_key":"secret"}`)
}

func TestSnapshotRequiresIdempotencyKey(t *testing.T) {
	c := &Client{}
	_, err := c.CreateSnapshot(context.Background(), "sb-1", CreateSnapshotParams{})
	require.ErrorContains(t, err, "IdempotencyKey")
}

func TestSnapshotResumeAfterUnknownOutcome(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, "capture-unknown", r.Header.Get("Idempotency-Key"))
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "demo", body["name"])
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			w.WriteHeader(408)
			_, _ = w.Write([]byte(`{"code":408,"message":"still processing"}`))
			return
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"snapshotID":"original-snapshot","names":["demo"]}`))
	}))
	defer server.Close()
	retries := 0
	c, err := NewClient(&Config{Endpoint: server.URL, RetryMax: &retries})
	require.NoError(t, err)
	name := "demo"
	params := CreateSnapshotParams{Name: &name, IdempotencyKey: "capture-unknown"}
	snapshot, err := c.CreateSnapshot(context.Background(), "sb-1", params)
	require.Nil(t, snapshot)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 408, apiErr.StatusCode)
	snapshot, err = c.CreateSnapshot(context.Background(), "sb-1", params)
	require.NoError(t, err)
	assert.Equal(t, "original-snapshot", snapshot.SnapshotID)
	assert.Equal(t, 2, calls)
}

func TestIsSnapshotRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "408", err: &APIError{StatusCode: http.StatusRequestTimeout}, want: true},
		{name: "502", err: &APIError{StatusCode: http.StatusBadGateway}, want: true},
		{name: "504", err: &APIError{StatusCode: http.StatusGatewayTimeout}, want: true},
		{name: "api_shutdown 503", err: &APIError{StatusCode: http.StatusServiceUnavailable, Code: "api_shutdown"}, want: true},
		{name: "other 503", err: &APIError{StatusCode: http.StatusServiceUnavailable, Code: "unavailable"}},
		{name: "503 without error code", err: &APIError{StatusCode: http.StatusServiceUnavailable}},
		{name: "500", err: &APIError{StatusCode: http.StatusInternalServerError}},
		{name: "400", err: &APIError{StatusCode: http.StatusBadRequest}},
		{name: "409", err: &APIError{StatusCode: http.StatusConflict}},
		{name: "network", err: errors.New("connection reset by peer"), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isSnapshotRetryable(tt.err))
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  time.Duration
		ok    bool
	}{
		{name: "missing"},
		{name: "zero", value: "0", ok: true},
		{name: "seconds", value: "7", want: 7 * time.Second, ok: true},
		{name: "negative", value: "-1"},
		{name: "HTTP date", value: "Wed, 21 Oct 2015 07:28:00 GMT"},
		{name: "overflow", value: strings.Repeat("9", 100)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseRetryAfter(tt.value)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSnapshotRetryAfterFromAPIError(t *testing.T) {
	header := make(http.Header)
	header.Set("Retry-After", "12")
	err := newAPIError(&http.Response{StatusCode: http.StatusRequestTimeout, Header: header}, nil)
	delay, ok := snapshotRetryAfter(err)
	assert.True(t, ok)
	assert.Equal(t, 12*time.Second, delay)

	_, ok = snapshotRetryAfter(errors.New("connection reset"))
	assert.False(t, ok)
}

func TestRetryCallWithPolicyRetryAfterOverridesBackoff(t *testing.T) {
	retryMax := 1
	c, err := NewClient(&Config{RetryMax: &retryMax, RetryBackoff: func(int) time.Duration { return 0 }})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	calls := 0
	err = c.retryCallWithPolicy(ctx, func() error {
		calls++
		return errors.New("retry")
	}, func(error) bool {
		return true
	}, func(error) (time.Duration, bool) {
		return time.Hour, true
	})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, 1, calls)
}
