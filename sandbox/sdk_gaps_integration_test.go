//go:build integration

package sandbox

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationListCursor(t *testing.T) {
	client := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	runID := uuid.NewString()
	metadata := Metadata{"sdkListTest": runID}
	first := createTestSandbox(t, client, ctx, metadata)
	second := createTestSandbox(t, client, ctx, metadata)
	require.NotEqual(t, first.ID(), second.ID())
	query := "sdkListTest=" + runID
	limit := int32(1)
	params := &ListParams{Limit: &limit, Metadata: &query}
	page, err := client.List(ctx, params)
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.NotEmpty(t, page.NextToken, "two matching sandboxes require a second page")
	params.NextToken = &page.NextToken
	next, err := client.List(ctx, params)
	require.NoError(t, err)
	require.Len(t, next.Items, 1)
	assert.Empty(t, next.NextToken)
	assert.NotEqual(t, page.Items[0].SandboxID, next.Items[0].SandboxID)
	assert.ElementsMatch(t, []string{first.ID(), second.ID()}, []string{page.Items[0].SandboxID, next.Items[0].SandboxID})
}

func TestIntegrationPtyReconnectRawOutput(t *testing.T) {
	client := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	sb := createTestSandbox(t, client, ctx)
	original, err := sb.Pty().Create(ctx, PtySize{Rows: 24, Cols: 80})
	require.NoError(t, err)
	pid, err := original.WaitPID(ctx)
	require.NoError(t, err)
	original.Disconnect()
	_, err = original.WaitContext(ctx)
	require.NoError(t, err)

	var output []byte
	var mu sync.Mutex
	reconnected, err := sb.Pty().ConnectWithOptions(ctx, pid, &PtyConnectOptions{OnData: func(data []byte) {
		mu.Lock()
		output = append(output, data...)
		mu.Unlock()
	}})
	require.NoError(t, err)
	require.NoError(t, sb.Pty().SendInput(ctx, pid, []byte("printf '\\377'; exit\n")))
	result, err := reconnected.WaitContext(ctx)
	require.NoError(t, err)
	assert.Zero(t, result.ExitCode)
	assert.True(t, bytes.Contains([]byte(result.Stdout), []byte{0xff}), "result must preserve non-UTF-8 PTY bytes")
	mu.Lock()
	defer mu.Unlock()
	assert.True(t, bytes.Contains(output, []byte{0xff}), "callback must preserve non-UTF-8 PTY bytes")
}
