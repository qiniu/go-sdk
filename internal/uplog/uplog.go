package uplog

import (
	"context"
	"net"
	"net/url"
	"os"
	"strings"
	"syscall"

	"github.com/qiniu/go-sdk/v7/storagev2/retrier"
)

type (
	LogType      string
	APIType      string
	ErrorType    string
	UpApiVersion uint8
	UpType       string
	LogResult    string
)

const (
	LogTypeRequest                  LogType      = "request"
	LogTypeBlock                    LogType      = "block"
	LogTypeQuality                  LogType      = "quality"
	LogTypeTransaction              LogType      = "transaction"
	APITypeKodo                     APIType      = "kodo"
	ErrorTypeUnknownError           ErrorType    = "unknown_error"
	ErrorTypeTimeout                ErrorType    = "timeout"
	ErrorTypeUnknownHost            ErrorType    = "unknown_host"
	ErrorTypeMaliciousResponse      ErrorType    = "malicious_response"
	ErrorTypeCannotConnectToHost    ErrorType    = "cannot_connect_to_host"
	ErrorTypeSSLError               ErrorType    = "ssl_error"
	ErrorTypeTransmissionError      ErrorType    = "transmission_error"
	ErrorTypeProtocolError          ErrorType    = "protocol_error"
	ErrorTypeResponseError          ErrorType    = "response_error"
	ErrorTypeUserCanceled           ErrorType    = "user_canceled"
	ErrorTypeLocalIoError           ErrorType    = "local_io_error"
	ErrorTypeUnexpectedSyscallError ErrorType    = "unexpected_syscall_error"
	UpApiVersionV1                  UpApiVersion = 1
	UpApiVersionV2                  UpApiVersion = 2
	UpTypeForm                      UpType       = "form"
	UpTypeResumableV1               UpType       = "resumable_v1"
	UpTypeResumableV2               UpType       = "resumable_v2"
	LogResultOK                     LogResult    = "ok"
	LogResultBadRequest             LogResult    = "bad_request"
	LogResultInvalidArgs            LogResult    = "invalid_args"
	LogResultUnknownError           LogResult    = "unknown_error"
	LogResultTimeout                LogResult    = "timeout"
	LogResultUnknownHost            LogResult    = "unknown_host"
	LogResultMaliciousResponse      LogResult    = "malicious_response"
	LogResultCannotConnectToHost    LogResult    = "cannot_connect_to_host"
	LogResultSSLError               LogResult    = "ssl_error"
	LogResultTransmissionError      LogResult    = "transmission_error"
	LogResultProtocolError          LogResult    = "protocol_error"
	LogResultResponseError          LogResult    = "response_error"
	LogResultUserCanceled           LogResult    = "user_canceled"
	LogResultLocalIoError           LogResult    = "local_io_error"
	LogResultUnexpectedSyscallError LogResult    = "unexpected_syscall_error"
)

var (
	osVersion string
)

func getOsVersion() string {
	return osVersion
}

func detectErrorType(err error) ErrorType {
	tryToUnwrapUnderlyingError := func(err error) (error, bool) {
		switch err := err.(type) {
		case *os.PathError:
			return err.Err, true
		case *os.LinkError:
			return err.Err, true
		case *os.SyscallError:
			return err.Err, true
		case *url.Error:
			return err.Err, true
		case *net.OpError:
			return err.Err, true
		}
		return err, false
	}
	unwrapUnderlyingError := func(err error) error {
		ok := true
		for ok {
			err, ok = tryToUnwrapUnderlyingError(err)
		}
		return err
	}

	unwrapedErr := unwrapUnderlyingError(err)
	if unwrapedErr == retrier.ErrMaliciousResponse {
		return ErrorTypeMaliciousResponse
	} else if os.IsTimeout(unwrapedErr) {
		return ErrorTypeTimeout
	} else if dnsError, ok := unwrapedErr.(*net.DNSError); ok && isDnsNotFoundError(dnsError) {
		return ErrorTypeUnknownHost
	} else if os.IsNotExist(unwrapedErr) || os.IsPermission(unwrapedErr) {
		return ErrorTypeLocalIoError
	} else if syscallError, ok := unwrapedErr.(*os.SyscallError); ok {
		switch syscallError.Err {
		case syscall.ECONNREFUSED, syscall.ECONNABORTED, syscall.ECONNRESET:
			return ErrorTypeCannotConnectToHost
		default:
			return ErrorTypeUnexpectedSyscallError
		}
	} else if errno, ok := unwrapedErr.(syscall.Errno); ok {
		switch errno {
		case syscall.ECONNREFUSED, syscall.ECONNABORTED, syscall.ECONNRESET:
			return ErrorTypeCannotConnectToHost
		default:
			return ErrorTypeUnexpectedSyscallError
		}
	} else if unwrapedErr == context.Canceled {
		return ErrorTypeUserCanceled
	} else {
		desc := unwrapedErr.Error()
		if strings.HasPrefix(desc, "tls: ") ||
			strings.HasPrefix(desc, "x509: ") {
			return ErrorTypeSSLError
		} else if strings.Contains(desc, "use of closed network connection") ||
			strings.Contains(desc, "unexpected EOF reading trailer") ||
			strings.Contains(desc, "transport connection broken") ||
			strings.Contains(desc, "server closed idle connection") {
			return ErrorTypeTransmissionError
		}
		return ErrorTypeUnknownError
	}
}

func getHttpClientName() string {
	httpClientName := "QiniuGo"
	if testRuntime {
		httpClientName = "QiniuGo_Debug"
	}
	return httpClientName
}

type (
	bytesSentContextKey struct {
		blockLevel bool
	}
	bytesSentTotal struct {
		N uint64
	}
)

func withBytesSentTotalContext(ctx context.Context, blockLevel bool) (*uint64, context.Context) {
	brt := new(bytesSentTotal)
	return &brt.N, context.WithValue(ctx, bytesSentContextKey{blockLevel}, brt)
}

func getBytesSentTotalFromContext(ctx context.Context, blockLevel bool) *uint64 {
	if bytesSentTotal, ok := ctx.Value(bytesSentContextKey{blockLevel}).(*bytesSentTotal); ok {
		return &bytesSentTotal.N
	}
	return nil
}

type (
	bytesReceivedContextKey struct {
		blockLevel bool
	}
	bytesReceivedTotal struct {
		N uint64
	}
)

func withBytesReceivedTotalContext(ctx context.Context, blockLevel bool) (*uint64, context.Context) {
	brt := new(bytesReceivedTotal)
	return &brt.N, context.WithValue(ctx, bytesReceivedContextKey{blockLevel}, brt)
}

func getBytesReceivedTotalFromContext(ctx context.Context, blockLevel bool) *uint64 {
	if bytesReceivedTotal, ok := ctx.Value(bytesReceivedContextKey{blockLevel}).(*bytesReceivedTotal); ok {
		return &bytesReceivedTotal.N
	}
	return nil
}

type (
	requestsCountContextKey struct {
		blockLevel bool
	}
	requestsCount struct {
		N uint64
	}
)

func withRequestsCountContext(ctx context.Context, blockLevel bool) (*uint64, context.Context) {
	brt := new(requestsCount)
	return &brt.N, context.WithValue(ctx, requestsCountContextKey{blockLevel}, brt)
}

func getRequestsCountFromContext(ctx context.Context, blockLevel bool) *uint64 {
	if requestsCount, ok := ctx.Value(requestsCountContextKey{blockLevel}).(*requestsCount); ok {
		return &requestsCount.N
	}
	return nil
}
