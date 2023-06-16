package clientv2

import (
	clientv1 "github.com/qiniu/go-sdk/v7/client"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"syscall"
	"time"
)

type RetryOptions struct {
	RetryMax      int                  // 最大重试次数
	RetryInterval func() time.Duration // 重试时间间隔
	ShouldRetry   func(req *http.Request, resp *http.Response, err error) bool
}

func DefaultOptions() RetryOptions {
	o := RetryOptions{}
	o.Init()
	return o
}

func (o *RetryOptions) Init() {
	if o == nil {
		return
	}

	if o.RetryMax < 0 {
		o.RetryMax = 0
	}

	if o.RetryInterval == nil {
		o.RetryInterval = func() time.Duration {
			return time.Duration(50+rand.Int()%50) * time.Millisecond
		}
	}

	if o.ShouldRetry == nil {
		o.ShouldRetry = func(req *http.Request, resp *http.Response, err error) bool {
			return isSimpleRetryable(req, resp, err)
		}
	}
}

type simpleRetryInterceptor struct {
	options RetryOptions
}

func NewSimpleRetryInterceptor(options RetryOptions) Interceptor {
	return &simpleRetryInterceptor{
		options: options,
	}
}

func (interceptor *simpleRetryInterceptor) Priority() InterceptorPriority {
	return InterceptorPriorityRetrySimple
}

func (interceptor *simpleRetryInterceptor) Intercept(req *http.Request, handler Handler) (resp *http.Response, err error) {
	if interceptor == nil || req == nil {
		return handler(req)
	}

	interceptor.options.Init()

	// 不重试
	if interceptor.options.RetryMax == 0 {
		return handler(req)
	}

	// 可能会被重试多次
	for i := 0; ; i++ {
		// Clone 防止后面 Handler 处理对 req 有污染
		reqBefore := cloneReq(req.Context(), req)
		resp, err = handler(req)

		if !interceptor.options.ShouldRetry(reqBefore, resp, err) {
			return resp, err
		}
		req = reqBefore

		if i >= interceptor.options.RetryMax {
			break
		}

		retryInterval := interceptor.options.RetryInterval()
		if retryInterval <= time.Millisecond {
			continue
		}
		time.Sleep(retryInterval)
	}
	return resp, err
}

func isSimpleRetryable(req *http.Request, resp *http.Response, err error) bool {
	return isRequestRetryable(req) && (isResponseRetryable(resp) || IsErrorRetryable(err))
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
		return false
	}
}

func isNetworkErrorWithOpError(err *net.OpError) bool {
	if err == nil {
		return false
	}

	switch t := err.Err.(type) {
	case *net.DNSError:
		return true
	case *os.SyscallError:
		if errno, ok := t.Err.(syscall.Errno); ok {
			switch errno {
			case syscall.ECONNABORTED:
				return true
			case syscall.ECONNRESET:
				return true
			case syscall.ECONNREFUSED:
				return true
			case syscall.ETIMEDOUT:
				return true
			}
		}
	}

	return false
}
