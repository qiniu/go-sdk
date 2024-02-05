package retrier

import (
	"io"
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
		Retry(*http.Request, *http.Response, error, *RetrierOptions) RetryDecision
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

type neverRetrier struct{}

func NewNeverRetrier() Retrier {
	return neverRetrier{}
}

func (neverRetrier) Retry(*http.Request, *http.Response, error, *RetrierOptions) RetryDecision {
	return DontRetry
}

type errorRetrier struct{}

// NewErrorRetrier 创建错误重试器，为七牛默认的错误重试器
func NewErrorRetrier() Retrier {
	return errorRetrier{}
}

func (errorRetrier) Retry(request *http.Request, response *http.Response, err error, _ *RetrierOptions) RetryDecision {
	if isRequestRetryable(request) && (isResponseRetryable(response) || IsErrorRetryable(err)) {
		return RetryRequest
	} else {
		return DontRetry
	}
}

func isRequestRetryable(req *http.Request) bool {
	if req == nil {
		return false
	}

	if req.Body == nil {
		return true
	}

	if req.GetBody != nil {
		b, err := req.GetBody()
		if err != nil || b == nil {
			return false
		}
		req.Body = b
		return true
	}

	seeker, ok := req.Body.(io.Seeker)
	if !ok {
		return false
	}

	_, err := seeker.Seek(0, io.SeekStart)
	return err == nil
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

	switch t := err.(type) {
	case *net.OpError:
		return isNetworkErrorWithOpError(t)
	case *url.Error:
		return IsErrorRetryable(t.Err)
	case net.Error:
		return t.Timeout()
	case *clientv1.ErrorInfo:
		return isStatusCodeRetryable(t.Code)
	default:
		return err == io.EOF
	}
}

func isNetworkErrorWithOpError(err *net.OpError) bool {
	if err == nil || err.Err == nil {
		return false
	}

	switch t := err.Err.(type) {
	case *net.DNSError:
		return true
	case *os.SyscallError:
		if errno, ok := t.Err.(syscall.Errno); ok {
			return errno == syscall.ECONNABORTED ||
				errno == syscall.ECONNRESET ||
				errno == syscall.ECONNREFUSED ||
				errno == syscall.ETIMEDOUT
		}
		return false
	case *net.OpError:
		return isNetworkErrorWithOpError(t)
	default:
		desc := err.Err.Error()
		return strings.Contains(desc, "use of closed network connection") ||
			strings.Contains(desc, "unexpected EOF reading trailer") ||
			strings.Contains(desc, "transport connection broken") ||
			strings.Contains(desc, "server closed idle connection")
	}
}
