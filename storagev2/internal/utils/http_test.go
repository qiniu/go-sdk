package utils

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"

	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
)

// 模拟 io.ReadSeekCloser 以模拟错误
type errorSeeker struct {
	io.ReadSeeker
	err error
}

func (e *errorSeeker) Close() error {
	return nil
}

func (e *errorSeeker) Seek(offset int64, whence int) (int64, error) {
	if e.err != nil {
		return 0, e.err
	}
	return e.ReadSeeker.Seek(offset, whence)
}

func TestHttpHeadAddContentLength_Success(t *testing.T) {
	data := bytes.NewReader([]byte("test"))
	dataS := internal_io.MakeReadSeekCloserFromLimitedReader(data, data.Size())

	header := http.Header{}
	err := HttpHeadAddContentLength(header, dataS)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expectedLength := "4"
	actualLength := header.Get("Content-Length")
	if actualLength != expectedLength {
		t.Errorf("Expected Content-Length to be %s, got %s", expectedLength, actualLength)
	}
}

func TestHttpHeadAddContentLength_SeekStartError(t *testing.T) {
	header := http.Header{}
	data := &errorSeeker{
		ReadSeeker: bytes.NewReader([]byte("test")),
		err:        errors.New("seek start error"),
	}

	err := HttpHeadAddContentLength(header, data)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestHttpHeadAddContentLength_SeekEndError(t *testing.T) {
	header := http.Header{}
	data := &errorSeeker{
		ReadSeeker: bytes.NewReader([]byte("test")),
		err:        errors.New("seek end error"),
	}

	err := HttpHeadAddContentLength(header, data)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}
