package io

import (
	"bytes"
	"io"
	"strings"
)

func ReadAll(r io.Reader) ([]byte, error) {
	switch b := r.(type) {
	case *BytesNopCloser:
		_, err := b.Seek(0, io.SeekEnd)
		return b.Bytes(), err
	default:
		return io.ReadAll(r)
	}
}

func SinkAll(r io.Reader) (err error) {
	switch b := r.(type) {
	case *BytesNopCloser:
		_, err = b.Seek(0, io.SeekEnd)
	case *bytes.Buffer:
		b.Truncate(0)
	case *bytes.Reader:
		_, err = b.Seek(0, io.SeekEnd)
	case *strings.Reader:
		_, err = b.Seek(0, io.SeekEnd)
	default:
		_, err = io.Copy(io.Discard, r)
	}
	return
}
