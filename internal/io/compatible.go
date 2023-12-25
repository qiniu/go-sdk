package io

import (
	"errors"
	"io"
)

func MakeReadSeekCloserFromReader(r io.Reader) ReadSeekCloser {
	return readSeekCloserFromReader{r: r}
}

type readSeekCloserFromReader struct {
	r io.Reader
}

func (r readSeekCloserFromReader) Read(p []byte) (int, error) {
	return r.r.Read(p)
}

func (r readSeekCloserFromReader) Seek(offset int64, whence int) (int64, error) {
	if seeker, ok := r.r.(io.Seeker); ok {
		return seeker.Seek(offset, whence)
	} else {
		return 0, errors.New("not support seek")
	}
}

func (r readSeekCloserFromReader) Close() error {
	if closer, ok := r.r.(io.Closer); ok {
		return closer.Close()
	} else {
		return nil
	}
}
