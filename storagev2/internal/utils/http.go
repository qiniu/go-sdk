package utils

import (
	"io"
	"net/http"
	"strconv"
)

func HttpHeadAddContentLength(header http.Header, data io.ReadSeekCloser) error {
	_, err := data.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	contentLength, err := data.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	header.Set("Content-Length", strconv.FormatInt(contentLength, 10))

	return nil
}
