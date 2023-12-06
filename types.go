package api

import (
	"bytes"
	"io"
	"net/http"

	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
)

// BytesFromRequest 读取 http.Request.Body 的内容到 slice 中
func BytesFromRequest(r *http.Request) ([]byte, error) {
	if bytesNopCloser, ok := r.Body.(*internal_io.BytesNopCloser); ok {
		return bytesNopCloser.Bytes(), nil
	}
	buf := bytes.NewBuffer(make([]byte, 0, int(r.ContentLength)+1024))
	_, err := io.Copy(buf, r.Body)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// SeekerLen 通过 io.Seeker 获取数据大小
func SeekerLen(s io.Seeker) (int64, error) {

	curOffset, err := s.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}

	endOffset, err := s.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}

	_, err = s.Seek(curOffset, io.SeekStart)
	if err != nil {
		return 0, err
	}

	return endOffset - curOffset, nil
}
