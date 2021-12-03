package pili

import (
	"net/http"

	"github.com/qiniu/go-sdk/v7/client"
)

var (
	ErrInvalidArgs             = &client.ErrorInfo{Code: http.StatusBadRequest, Err: "invalid args"}
	ErrInvalidRule             = &client.ErrorInfo{Code: http.StatusBadRequest, Err: "invalid rule"}
	ErrUnsupportedSecurityType = &client.ErrorInfo{Code: http.StatusBadRequest, Err: "unsupported security type"}
)

func ErrInfo(code int, err string) *client.ErrorInfo {
	return &client.ErrorInfo{
		Code: code,
		Err:  err,
	}
}
