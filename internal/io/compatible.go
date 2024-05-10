package io

import (
	"errors"
	"io"
	"syscall"
)

func MakeReadSeekCloserFromReader(r io.Reader) ReadSeekCloser {
	return &readSeekCloserFromReader{r: r}
}

type readSeekCloserFromReader struct {
	r io.Reader
}

func (r *readSeekCloserFromReader) Read(p []byte) (int, error) {
	return r.r.Read(p)
}

func (r *readSeekCloserFromReader) Seek(offset int64, whence int) (int64, error) {
	if seeker, ok := r.r.(io.Seeker); ok {
		return seeker.Seek(offset, whence)
	}
	return 0, syscall.ESPIPE
}

func (r *readSeekCloserFromReader) Close() error {
	return nil
}

func MakeReadSeekCloserFromLimitedReader(r io.Reader, size int64) ReadSeekCloser {
	return &sizedReadSeekCloserFromReader{r: io.LimitedReader{R: r, N: size}, size: size}
}

type sizedReadSeekCloserFromReader struct {
	r    io.LimitedReader
	size int64
}

func (r *sizedReadSeekCloserFromReader) Read(p []byte) (int, error) {
	return r.r.Read(p)
}

func (r *sizedReadSeekCloserFromReader) Seek(offset int64, whence int) (int64, error) {
	if seeker, ok := r.r.R.(io.ReadSeeker); ok {
		newPos, err := seeker.Seek(offset, whence)
		if err != nil {
			return newPos, err
		}
		r.r.N = r.size - newPos
		return newPos, nil
	}
	return 0, errors.New("not support seek")
}

func (r *sizedReadSeekCloserFromReader) Close() error {
	return nil
}

func (r *sizedReadSeekCloserFromReader) DetectLength() (int64, error) {
	return r.size, nil
}
