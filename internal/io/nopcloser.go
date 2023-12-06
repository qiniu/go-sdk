package io

import (
	"bytes"
	"io"
)

type ReadSeekableNopCloser struct {
	r io.ReadSeeker
}

func NewReadSeekableNopCloser(r io.ReadSeeker) ReadSeekableNopCloser {
	return ReadSeekableNopCloser{r: r}
}

func (nc ReadSeekableNopCloser) Read(p []byte) (int, error) {
	return nc.r.Read(p)
}

func (nc ReadSeekableNopCloser) Seek(offset int64, whence int) (int64, error) {
	return nc.r.Seek(offset, whence)
}

func (nc ReadSeekableNopCloser) Close() error {
	return nil
}

type BytesNopCloser struct {
	r *bytes.Reader
	b []byte
}

func NewBytesNopCloser(b []byte) *BytesNopCloser {
	return &BytesNopCloser{r: bytes.NewReader(b), b: b}
}

func (nc *BytesNopCloser) Read(p []byte) (int, error) {
	return nc.r.Read(p)
}

func (nc *BytesNopCloser) ReadAt(b []byte, off int64) (int, error) {
	return nc.r.ReadAt(b, off)
}

func (nc *BytesNopCloser) Seek(offset int64, whence int) (int64, error) {
	return nc.r.Seek(offset, whence)
}

func (nc *BytesNopCloser) Size() int64 {
	return nc.r.Size()
}

func (nc *BytesNopCloser) Close() error {
	return nil
}

func (nc *BytesNopCloser) Bytes() []byte {
	return nc.b
}
