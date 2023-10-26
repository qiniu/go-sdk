//go:build 1.20
// +build 1.20

package context

import (
	"context"
)

type (
	Context         = context.Context
	CancelCauseFunc = context.CancelCauseFunc
)

func Cause(c Context) error {
	return Cause(c)
}

func WithCancelCause(parent Context) (ctx Context, cancel CancelCauseFunc) {
	return context.WithCancelCause(parent)
}

func Background() Context {
	return context.Background()
}
