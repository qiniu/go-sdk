package clientv2

import (
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"

	clientv1 "github.com/qiniu/go-sdk/v7/client"
	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
	"github.com/qiniu/go-sdk/v7/storagev2/chooser"
	"github.com/qiniu/go-sdk/v7/storagev2/resolver"
)

type contextKeyBufferResponse struct{}

type RetryConfig struct {
	RetryMax      int                  // 最大重试次数
	RetryInterval func() time.Duration // 重试时间间隔
	ShouldRetry   func(req *http.Request, resp *http.Response, err error) bool
}

func (c *RetryConfig) init() {
	if c == nil {
		return
	}

	if c.RetryMax < 0 {
		c.RetryMax = 0
	}

	if c.RetryInterval == nil {
		c.RetryInterval = func() time.Duration {
			return time.Duration(50+rand.Int()%50) * time.Millisecond
		}
	}

	if c.ShouldRetry == nil {
		c.ShouldRetry = isSimpleRetryable
	}
}

type (
	SimpleRetryConfig struct {
		RetryMax      int                  // 最大重试次数
		RetryInterval func() time.Duration // 重试时间间隔
		ShouldRetry   func(req *http.Request, resp *http.Response, err error) bool
		Resolver      resolver.Resolver // 主备域名解析器
		Chooser       chooser.Chooser   // 主备域名选择器
	}

	simpleRetryInterceptor struct {
		config SimpleRetryConfig
	}
)

func (c *SimpleRetryConfig) init() {
	if c == nil {
		return
	}

	if c.RetryMax < 0 {
		c.RetryMax = 0
	}

	if c.RetryInterval == nil {
		c.RetryInterval = func() time.Duration {
			return time.Duration(50+rand.Int()%50) * time.Millisecond
		}
	}

	if c.ShouldRetry == nil {
		c.ShouldRetry = isSimpleRetryable
	}
}

func NewSimpleRetryInterceptor(config SimpleRetryConfig) Interceptor {
	return &simpleRetryInterceptor{config: config}
}

func (interceptor *simpleRetryInterceptor) Priority() InterceptorPriority {
	return InterceptorPriorityRetrySimple
}

func (interceptor *simpleRetryInterceptor) Intercept(req *http.Request, handler Handler) (resp *http.Response, err error) {
	if interceptor == nil || req == nil {
		return handler(req)
	}
	toBufferResponse := req.Context().Value(contextKeyBufferResponse{}) != nil

	interceptor.config.init()

	var ips []net.IP
	hostname := req.URL.Hostname()

	if resolver := interceptor.config.Resolver; resolver != nil {
		if ips, err = resolver.Resolve(req.Context(), hostname); err == nil && len(ips) > 0 {
			if cs := interceptor.config.Chooser; cs != nil {
				ips = cs.Choose(req.Context(), &chooser.ChooseOptions{IPs: ips, Domain: hostname})
			}
			if len(ips) > 0 {
				req = req.WithContext(clientv1.WithResolvedIPs(req.Context(), hostname, ips))
			}
		}
	}

	// 可能会被重试多次
	for i := 0; ; i++ {
		// Clone 防止后面 Handler 处理对 req 有污染
		reqBefore := cloneReq(req)
		resp, err = handler(req)

		if err == nil {
			if toBufferResponse {
				err = bufferResponse(resp)
			}
		}

		if !interceptor.config.ShouldRetry(reqBefore, resp, err) {
			if len(ips) > 0 {
				if cs := interceptor.config.Chooser; cs != nil {
					cs.FeedbackGood(req.Context(), &chooser.FeedbackOptions{IPs: ips, Domain: hostname})
				}
			}
			return resp, err
		}
		if len(ips) > 0 {
			if cs := interceptor.config.Chooser; cs != nil {
				cs.FeedbackBad(req.Context(), &chooser.FeedbackOptions{IPs: ips, Domain: hostname})
			}
		}

		req = reqBefore

		if i >= interceptor.config.RetryMax {
			break
		}

		if resp != nil && resp.Body != nil {
			_ = internal_io.SinkAll(resp.Body)
			resp.Body.Close()
		}

		retryInterval := interceptor.config.RetryInterval()
		if retryInterval < time.Microsecond {
			continue
		}
		time.Sleep(retryInterval)
	}
	return resp, err
}

func bufferResponse(resp *http.Response) error {
	buffer, err := internal_io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body = internal_io.NewBytesNopCloser(buffer)
	return nil
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
		if err == io.EOF {
			return true
		}
		return false
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
	case *net.OpError:
		return isNetworkErrorWithOpError(t)
	default:
		desc := err.Err.Error()
		if strings.Contains(desc, "use of closed network connection") ||
			strings.Contains(desc, "unexpected EOF reading trailer") ||
			strings.Contains(desc, "transport connection broken") ||
			strings.Contains(desc, "server closed idle connection") {
			return true
		}
	}

	return false
}
