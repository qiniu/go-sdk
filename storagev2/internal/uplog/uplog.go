package uplog

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"

	"github.com/matishsiao/goInfo"
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
	ErrorTypeCannotConnectToHost    ErrorType    = "cannot_connect_to_host"
	ErrorTypeSSLError               ErrorType    = "ssl_error"
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
	LogResultCannotConnectToHost    LogResult    = "cannot_connect_to_host"
	LogResultSSLError               LogResult    = "ssl_error"
	LogResultProtocolError          LogResult    = "protocol_error"
	LogResultResponseError          LogResult    = "response_error"
	LogResultUserCanceled           LogResult    = "user_canceled"
	LogResultLocalIoError           LogResult    = "local_io_error"
	LogResultUnexpectedSyscallError LogResult    = "unexpected_syscall_error"
)

var osVersion string

func getOsVersion() (string, error) {
	if osVersion == "" {
		osInfo, err := goInfo.GetInfo()
		if err != nil {
			return "", err
		}
		osVersion = osInfo.Core
	}
	return osVersion, nil
}

func detectErrorType(err error) ErrorType {
	var (
		dnsError           *net.DNSError
		urlError           *url.Error
		tlsVerifyCertError *tls.CertificateVerificationError
		syscallError       syscall.Errno
	)
	if os.IsTimeout(err) {
		return ErrorTypeTimeout
	} else if errors.As(err, &dnsError) && dnsError.IsNotFound {
		return ErrorTypeUnknownHost
	} else if os.IsNotExist(err) || os.IsPermission(err) {
		return ErrorTypeLocalIoError
	} else if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET) {
		return ErrorTypeCannotConnectToHost
	} else if errors.As(err, &syscallError) {
		return ErrorTypeUnexpectedSyscallError
	} else if errors.Is(err, context.Canceled) {
		return ErrorTypeUserCanceled
	} else if errors.Is(err, http.ErrSchemeMismatch) {
		return ErrorTypeProtocolError
	} else if errors.As(err, &tlsVerifyCertError) {
		return ErrorTypeSSLError
	} else if errors.As(err, &urlError) &&
		(strings.HasPrefix(urlError.Err.Error(), "tls: ") ||
			strings.HasPrefix(urlError.Err.Error(), "x509: ")) {
		return ErrorTypeSSLError
	}
	return ErrorTypeUnknownError
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
