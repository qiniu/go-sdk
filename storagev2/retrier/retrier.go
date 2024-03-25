package retrier

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"

	clientv1 "github.com/qiniu/go-sdk/v7/client"
	"github.com/qiniu/go-sdk/v7/storagev2/backoff"
)

type (
	// RetryDecision 重试决策
	RetryDecision int
	// RetrierOptions 重试器选项
	RetrierOptions backoff.BackoffOptions

	// Retrier 重试器接口
	Retrier interface {
		// Retry 判断是否重试，如何重试
		Retry(*http.Response, error, *RetrierOptions) RetryDecision
	}

	neverRetrier      struct{}
	errorRetrier      struct{}
	customizedRetrier struct {
		retryFn func(*http.Response, error, *RetrierOptions) RetryDecision
	}
)

const (
	// 不再重试
	DontRetry RetryDecision = iota

	// 重试下一个域名
	TryNextHost

	// 重试当前域名
	RetryRequest
)

// NewRetrier 创建自定义重试器
func NewRetrier(fn func(*http.Response, error, *RetrierOptions) RetryDecision) Retrier {
	return customizedRetrier{retryFn: fn}
}

func (retrier customizedRetrier) Retry(response *http.Response, err error, options *RetrierOptions) RetryDecision {
	return retrier.retryFn(response, err, options)
}

// NewNeverRetrier 创建从不重试的重试器
func NewNeverRetrier() Retrier {
	return neverRetrier{}
}

func (neverRetrier) Retry(*http.Response, error, *RetrierOptions) RetryDecision {
	return DontRetry
}

// NewErrorRetrier 创建错误重试器，为七牛默认的错误重试器
func NewErrorRetrier() Retrier {
	return errorRetrier{}
}

func (errorRetrier) Retry(response *http.Response, err error, _ *RetrierOptions) RetryDecision {
	if isResponseRetryable(response) || IsErrorRetryable(err) {
		return RetryRequest
	} else {
		return DontRetry
	}
}

func isResponseRetryable(resp *http.Response) bool {
	if resp == nil {
		return false
	}
	return isStatusCodeRetryable(resp.StatusCode)
}

func isStatusCodeRetryable(statusCode int) bool {
	if statusCode < 500 {
		return false
	}

	if statusCode == 501 || statusCode == 509 || statusCode == 573 || statusCode == 579 ||
		statusCode == 608 || statusCode == 612 || statusCode == 614 || statusCode == 616 || statusCode == 618 ||
		statusCode == 630 || statusCode == 631 || statusCode == 632 || statusCode == 640 || statusCode == 701 {
		return false
	}

	return true
}

func IsErrorRetryable(err error) bool {
	if err == nil {
		return false
	}

	switch getRetryDecisionForError(err) {
	case RetryRequest, TryNextHost:
		return true
	default:
		return false
	}
}

func getRetryDecisionForError(err error) RetryDecision {
	if err == nil {
		return DontRetry
	}

	var dnsError *net.DNSError
	if os.IsTimeout(err) || errors.Is(err, syscall.ETIMEDOUT) {
		return RetryRequest
	} else if errors.As(err, &dnsError) && dnsError.IsNotFound {
		return TryNextHost
	} else if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNABORTED) || errors.Is(err, syscall.ECONNRESET) {
		return TryNextHost
	} else if errors.Is(err, context.Canceled) {
		return DontRetry
	} else if errors.Is(err, http.ErrSchemeMismatch) {
		return DontRetry
	}

	switch t := err.(type) {
	case *url.Error:
		desc := err.Error()
		if strings.Contains(desc, "use of closed network connection") ||
			strings.Contains(desc, "unexpected EOF reading trailer") ||
			strings.Contains(desc, "transport connection broken") ||
			strings.Contains(desc, "server closed idle connection") {
			return RetryRequest
		} else {
			return DontRetry
		}
	case *clientv1.ErrorInfo:
		if isStatusCodeRetryable(t.Code) {
			return RetryRequest
		} else {
			return DontRetry
		}
	default:
		return RetryRequest
	}
}
