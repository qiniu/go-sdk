package clientv2

import (
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
	RetryMax      int
	RetryInterval func() time.Duration
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

func (r *simpleRetryInterceptor) Priority() InterceptorPriority {
	return InterceptorPriorityRetrySimple
}

func (r *simpleRetryInterceptor) Intercept(req *http.Request, handler Handler) (resp *http.Response, err error) {
	r.options.Init()

	// 不重试
	if r.options.RetryMax == 0 {
		return handler(req)
	}

	// 可能会被重试多次
	for i := 0; ; i++ {
		// Clone 防止后面 Handler 处理对 req 有污染
		reqBefore := req.Clone(req.Context())
		resp, err = handler(req)

		if !r.options.ShouldRetry(reqBefore, resp, err) {
			return resp, err
		}
		req = reqBefore

		if i >= r.options.RetryMax {
			break
		}

		retryInterval := r.options.RetryInterval()
		if retryInterval <= time.Millisecond {
			continue
		}
		time.Sleep(retryInterval)
	}
	return resp, err
}

func isSimpleRetryable(req *http.Request, resp *http.Response, err error) bool {
	return isRequestSimpleRetryable(req) && isResponseSimpleRetryable(resp) && isErrorSimpleRetryable(err)
}

func isRequestSimpleRetryable(req *http.Request) bool {
	if req == nil {
		return false
	}

	if req.Body == nil {
		return true
	}

	seeker, ok := req.Body.(io.Seeker)
	if !ok {
		return false
	}

	_, err := seeker.Seek(0, io.SeekStart)
	return err == nil
}

func isResponseSimpleRetryable(resp *http.Response) bool {
	if resp == nil {
		return true
	}

	statusCode := resp.StatusCode
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

func isErrorSimpleRetryable(err error) bool {
	return err == nil || isNetworkError(err)
}

func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	switch t := err.(type) {
	case *net.OpError:
		return isNetworkErrorWithOpError(t)
	case *url.Error:
		return isNetworkError(t.Err)
	case net.Error:
		return t.Timeout()
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
			case syscall.ECONNREFUSED:
				return true
			case syscall.ETIMEDOUT:
				return true
			}
		}
	}

	return false
}
