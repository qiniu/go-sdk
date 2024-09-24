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
	if isResponseRetryable(response) {
		return RetryRequest
	} else if err == nil {
		return DontRetry
	} else {
		return getRetryDecisionForError(err)
	}
}

func isResponseRetryable(resp *http.Response) bool {
	if resp == nil {
		return false
	}
	return IsStatusCodeRetryable(resp.StatusCode)
}

func IsStatusCodeRetryable(statusCode int) bool {
	if statusCode < 500 {
		return false
	}

	if statusCode == 501 || statusCode == 509 || statusCode == 579 ||
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

var ErrMaliciousResponse = errors.New("malicious response")

func getRetryDecisionForError(err error) RetryDecision {
	if err == nil {
		return DontRetry
	}

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
	if unwrapedErr == context.DeadlineExceeded {
		return DontRetry
	} else if unwrapedErr == ErrMaliciousResponse {
		return RetryRequest
	} else if os.IsTimeout(unwrapedErr) {
		return RetryRequest
	} else if dnsError, ok := unwrapedErr.(*net.DNSError); ok && isDnsNotFoundError(dnsError) {
		return TryNextHost
	} else if syscallError, ok := unwrapedErr.(*os.SyscallError); ok {
		switch syscallError.Err {
		case syscall.ECONNREFUSED, syscall.ECONNABORTED, syscall.ECONNRESET:
			return TryNextHost
		default:
			return DontRetry
		}
	} else if errno, ok := unwrapedErr.(syscall.Errno); ok {
		switch errno {
		case syscall.ECONNREFUSED, syscall.ECONNABORTED, syscall.ECONNRESET:
			return TryNextHost
		default:
			return DontRetry
		}
	} else if unwrapedErr == context.Canceled {
		return DontRetry
	} else if clientErr, ok := unwrapedErr.(*clientv1.ErrorInfo); ok {
		if clientErr.Code == http.StatusBadRequest && strings.Contains(unwrapedErr.Error(), "transfer acceleration is not configured on this bucket") {
			return TryNextHost
		} else if IsStatusCodeRetryable(clientErr.Code) {
			return RetryRequest
		} else {
			return DontRetry
		}
	}
	desc := unwrapedErr.Error()
	if strings.Contains(desc, "use of closed network connection") ||
		strings.Contains(desc, "unexpected EOF reading trailer") ||
		strings.Contains(desc, "transport connection broken") ||
		strings.Contains(desc, "server closed idle connection") {
		return RetryRequest
	}
	return DontRetry
}
