package conf

import (
	"github.com/qiniu/go-sdk/v7/internal/env"
)

const Version = "7.24.0"

const (
	CONTENT_TYPE_JSON      = "application/json"
	CONTENT_TYPE_FORM      = "application/x-www-form-urlencoded"
	CONTENT_TYPE_OCTET     = "application/octet-stream"
	CONTENT_TYPE_MULTIPART = "multipart/form-data"
)

func IsDisableQiniuTimestampSignature() bool {
	isDisabled, _ := env.DisableQiniuTimestampSignatureFromEnvironment()
	return isDisabled
}
