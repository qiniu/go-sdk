//go:build !1.16
// +build !1.16

package io

import (
	"io"
)

type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}
