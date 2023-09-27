package client

import (
	"net/http"
	"time"

	"github.com/qiniu/go-sdk/v7/conf"
)

const (
	RequestHeaderKeyXQiniuDate = "X-Qiniu-Date"
)

func addDefaultHeader(headers http.Header) error {
	return addHttpHeaderXQiniuDate(headers)
}

func addHttpHeaderXQiniuDate(headers http.Header) error {
	if conf.IsDisableQiniuTimestampSignature() {
		return nil
	}

	timeString := time.Now().UTC().Format("20060102T150405Z")
	headers.Set(RequestHeaderKeyXQiniuDate, timeString)
	return nil
}

func AddHttpHeaderRange(header http.Header, contentRange string) {
	if len(contentRange) == 0 {
		return
	}

	header.Set("Range", contentRange)
}
