//go:build unit
// +build unit

package context_test

import (
	offical_context "context"
	"io"
	"testing"

	"github.com/qiniu/go-sdk/v7/internal/context"
)

func TestCause(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())

	select {
	case <-ctx.Done():
		t.Fatalf("Expect ctx.Done() is not done")
	default:
	}

	if err := ctx.Err(); err != nil {
		t.Fatalf("Expect ctx.Err() to return nil, but %s", err)
	}

	cancel(io.EOF)

	select {
	case <-ctx.Done():
	default:
		t.Fatalf("Expect ctx.Done() is done")
	}

	if err := ctx.Err(); err != offical_context.Canceled {
		t.Fatalf("Expect ctx.Err() to return Canceled, but %s", err)
	}

	if c := context.Cause(ctx); c != io.EOF {
		t.Fatalf("Expect context.Cause(ctx) to return io.EOF, but %T", c)
	}
}
