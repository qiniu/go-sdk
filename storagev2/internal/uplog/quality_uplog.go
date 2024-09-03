package uplog

import (
	"context"
	"encoding/json"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/qiniu/go-sdk/v7/client"
	"github.com/qiniu/go-sdk/v7/conf"
)

type qualityUplog struct {
	LogType           LogType   `json:"log_type,omitempty"`
	Result            LogResult `json:"result,omitempty"`
	UpType            UpType    `json:"up_type,omitempty"`
	TotalElapsedTime  uint64    `json:"total_elapsed_time,omitempty"`
	RequestsCount     uint64    `json:"requests_count,omitempty"`
	RegionsCount      uint64    `json:"regions_count,omitempty"`
	BytesSent         uint64    `json:"bytes_sent,omitempty"`
	BytesReceived     uint64    `json:"bytes_received,omitempty"`
	FileSize          uint64    `json:"file_size,omitempty"`
	APIType           APIType   `json:"api_type,omitempty"`
	UpTime            int64     `json:"up_time,omitempty"`
	TargetBucket      string    `json:"target_bucket,omitempty"`
	TargetKey         string    `json:"target_key,omitempty"`
	OSName            string    `json:"os_name,omitempty"`
	OSVersion         string    `json:"os_version,omitempty"`
	OSArch            string    `json:"os_arch,omitempty"`
	SDKName           string    `json:"sdk_name,omitempty"`
	SDKVersion        string    `json:"sdk_version,omitempty"`
	HTTPClient        string    `json:"http_client,omitempty"`
	HTTPClientVersion string    `json:"http_client_version,omitempty"`
	ErrorType         ErrorType `json:"error_type,omitempty"`
	ErrorDescription  string    `json:"error_description,omitempty"`
	PerceptiveSpeed   uint64    `json:"perceptive_speed,omitempty"`
}

func WithQuality(ctx context.Context, upType UpType, fileSize uint64, targetBucket, targetKey, upToken string, handle func(ctx context.Context, switchRegion func()) error) error {
	if !IsUplogEnabled() {
		return handle(ctx, func() {})
	}
	uplog := qualityUplog{
		LogType:           LogTypeQuality,
		UpType:            upType,
		FileSize:          fileSize,
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
	bytesSentTotal, ctx := withBytesSentTotalContext(ctx, false)
	bytesReceivedTotal, ctx := withBytesReceivedTotalContext(ctx, false)
	requestsCount, ctx := withRequestsCountContext(ctx, false)
	beginAt := time.Now()
	err := handle(ctx, func() {
		atomic.AddUint64(&uplog.RegionsCount, 1)
	})
	uplog.TotalElapsedTime = getElapsedTime(beginAt)
	if err != nil {
		uplog.ErrorType = detectErrorType(err)
		uplog.ErrorDescription = truncate(err.Error(), maxFieldValueLength)
	}
	uplog.Result = detectLogResult(err)
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

func detectLogResult(err error) LogResult {
	if err != nil {
		if clientErr, ok := err.(*client.ErrorInfo); ok {
			if clientErr.Code/100 == 4 || clientErr.Code == 573 || clientErr.Code == 579 || clientErr.Code == 608 || clientErr.Code == 612 ||
				clientErr.Code == 614 || clientErr.Code == 630 || clientErr.Code == 631 || clientErr.Code == 701 {
				return LogResultBadRequest
			} else if clientErr.Code/100 != 2 {
				return LogResultResponseError
			}
		}
		switch detectErrorType(err) {
		case ErrorTypeTimeout:
			return LogResultTimeout
		case ErrorTypeUnknownHost:
			return LogResultUnknownHost
		case ErrorTypeMaliciousResponse:
			return LogResultMaliciousResponse
		case ErrorTypeCannotConnectToHost:
			return LogResultCannotConnectToHost
		case ErrorTypeUserCanceled:
			return LogResultUserCanceled
		case ErrorTypeProtocolError:
			return LogResultProtocolError
		case ErrorTypeSSLError:
			return LogResultSSLError
		case ErrorTypeTransmissionError:
			return LogResultTransmissionError
		case ErrorTypeLocalIoError:
			return LogResultLocalIoError
		case ErrorTypeUnexpectedSyscallError:
			return LogResultUnexpectedSyscallError
		default:
			return LogResultUnknownError
		}
	}
	return LogResultOK
}
