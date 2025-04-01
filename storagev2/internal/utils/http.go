package utils

import (
	"io"
	"net/http"
	"strconv"

	innerio "github.com/qiniu/go-sdk/v7/internal/io"
)

func HttpHeadAddContentLength(header http.Header, data innerio.ReadSeekCloser) error {
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
