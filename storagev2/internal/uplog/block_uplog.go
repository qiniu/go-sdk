package uplog

import (
	"context"
	"encoding/json"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/qiniu/go-sdk/v7/conf"
)

type blockUplog struct {
	LogType           LogType      `json:"log_type,omitempty"`
	UpApiVersion      UpApiVersion `json:"up_api_version,omitempty"`
	TotalElapsedTime  uint64       `json:"total_elapsed_time,omitempty"`
	RequestsCount     uint64       `json:"requests_count,omitempty"`
	BytesSent         uint64       `json:"bytes_sent,omitempty"`
	BytesReceived     uint64       `json:"bytes_received,omitempty"`
	RecoveredFrom     uint64       `json:"recovered_from,omitempty"`
	FileSize          uint64       `json:"file_size,omitempty"`
	APIType           APIType      `json:"api_type,omitempty"`
	UpTime            int64        `json:"up_time,omitempty"`
	TargetBucket      string       `json:"target_bucket,omitempty"`
	TargetKey         string       `json:"target_key,omitempty"`
	OSName            string       `json:"os_name,omitempty"`
	OSVersion         string       `json:"os_version,omitempty"`
	OSArch            string       `json:"os_arch,omitempty"`
	SDKName           string       `json:"sdk_name,omitempty"`
	SDKVersion        string       `json:"sdk_version,omitempty"`
	HTTPClient        string       `json:"http_client,omitempty"`
	HTTPClientVersion string       `json:"http_client_version,omitempty"`
	ErrorType         ErrorType    `json:"error_type,omitempty"`
	ErrorDescription  string       `json:"error_description,omitempty"`
	PerceptiveSpeed   uint64       `json:"perceptive_speed,omitempty"`
}

func WithBlock(ctx context.Context, upApiVersion UpApiVersion, fileSize, recoveredFrom uint64, targetBucket, targetKey, upToken string, handle func(context.Context) error) error {
	if !IsUplogEnabled() {
		return handle(ctx)
	}

	uplog := blockUplog{
		LogType:           LogTypeBlock,
		UpApiVersion:      upApiVersion,
		FileSize:          fileSize,
		RecoveredFrom:     recoveredFrom,
		TargetBucket:      truncate(targetBucket, maxFieldValueLength),
		TargetKey:         truncate(targetKey, maxFieldValueLength),
		APIType:           APITypeKodo,
		OSName:            truncate(runtime.GOOS, maxFieldValueLength),
		OSVersion:         truncate(getOsVersion(), maxFieldValueLength),
		OSArch:            truncate(runtime.GOARCH, maxFieldValueLength),
		SDKName:           "go",
		SDKVersion:        truncate(conf.Version, maxFieldValueLength),
		HTTPClient:        truncate(getHttpClientName(), maxFieldValueLength),
		HTTPClientVersion: truncate(conf.Version, maxFieldValueLength),
	}
	bytesSentTotal, ctx := withBytesSentTotalContext(ctx, true)
	bytesReceivedTotal, ctx := withBytesReceivedTotalContext(ctx, true)
	requestsCount, ctx := withRequestsCountContext(ctx, true)
	beginAt := time.Now()
	err := handle(ctx)
	uplog.TotalElapsedTime = getElapsedTime(beginAt)
	if err != nil {
		uplog.ErrorType = detectErrorType(err)
		uplog.ErrorDescription = truncate(err.Error(), maxFieldValueLength)
	}
	uplog.BytesSent = atomic.LoadUint64(bytesSentTotal)
	uplog.BytesReceived = atomic.LoadUint64(bytesReceivedTotal)
	uplog.RequestsCount = atomic.LoadUint64(requestsCount)
	if uplog.TotalElapsedTime > 0 {
		if uplog.BytesSent > uplog.BytesReceived {
			uplog.PerceptiveSpeed = uplog.BytesSent * 1000 / uplog.TotalElapsedTime
		} else {
			uplog.PerceptiveSpeed = uplog.BytesReceived * 1000 / uplog.TotalElapsedTime
		}
	}
	if uplogBytes, jsonError := json.Marshal(uplog); jsonError == nil {
		uplogChan <- uplogSerializedEntry{serializedUplog: uplogBytes, getUpToken: func() (string, error) {
			return upToken, nil
		}}
	}
	return err
}
