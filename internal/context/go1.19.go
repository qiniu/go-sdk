//go:build !1.20
// +build !1.20

package context

import (
	"context"
	"sync"
)

type (
	Context               = context.Context
	CancelCauseFunc       func(cause error)
	cancelCauseErrorKey   struct{}
	cancelCauseErrorValue struct {
		mutex sync.Mutex
		err   error
	}
)

func Cause(c Context) error {
	if v := c.Value(cancelCauseErrorKey{}); v != nil {
		if val, ok := v.(*cancelCauseErrorValue); ok {
			val.mutex.Lock()
			defer val.mutex.Unlock()
			return val.err
		}
	}
	return c.Err()
}

func WithCancelCause(parent Context) (ctx Context, cancel CancelCauseFunc) {
	errWrapper := new(cancelCauseErrorValue)
	newCtx, cancelFunc := context.WithCancel(context.WithValue(parent, cancelCauseErrorKey{}, errWrapper))
	return newCtx, func(cause error) {
		errWrapper.mutex.Lock()
		defer errWrapper.mutex.Unlock()
		if errWrapper.err == nil {
			errWrapper.err = cause
		}
		cancelFunc()
	}
}

func Background() Context {
	return context.Background()
}
