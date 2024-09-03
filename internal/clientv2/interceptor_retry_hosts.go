package clientv2

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/qiniu/go-sdk/v7/internal/hostprovider"
	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
	"github.com/qiniu/go-sdk/v7/storagev2/retrier"
)

type HostsRetryConfig struct {
	RetryMax           int // 最大重试次数
	ShouldRetry        func(req *http.Request, resp *http.Response, err error) bool
	Retrier            retrier.Retrier           // 重试器
	HostFreezeDuration time.Duration             // 主备域名冻结时间（默认：600s），当一个域名请求失败被冻结的时间，最小 time.Millisecond
	HostProvider       hostprovider.HostProvider // 备用域名获取方法
	ShouldFreezeHost   func(req *http.Request, resp *http.Response, err error) bool
}

func (c *HostsRetryConfig) init() {
	if c.RetryMax < 0 {
		c.RetryMax = 1
	}

	if c.HostFreezeDuration < time.Millisecond {
		c.HostFreezeDuration = 600 * time.Second
	}

	if c.ShouldFreezeHost == nil {
		c.ShouldFreezeHost = func(req *http.Request, resp *http.Response, err error) bool {
			return true
		}
	}
}

func (c *HostsRetryConfig) getRetryDecision(req *http.Request, resp *http.Response, err error, attempts int) retrier.RetryDecision {
	if c.ShouldRetry != nil {
		if c.ShouldRetry(req, resp, err) {
			return retrier.RetryRequest
		} else {
			return retrier.DontRetry
		}
	} else if c.Retrier != nil {
		return c.Retrier.Retry(resp, err, &retrier.RetrierOptions{Attempts: attempts})
	} else {
		return errorRetrier.Retry(resp, err, &retrier.RetrierOptions{Attempts: attempts})
	}
}

type hostsRetryInterceptor struct {
	options HostsRetryConfig
}

func NewHostsRetryInterceptor(options HostsRetryConfig) Interceptor {
	return &hostsRetryInterceptor{
		options: options,
	}
}

func (interceptor *hostsRetryInterceptor) Priority() InterceptorPriority {
	return InterceptorPriorityRetryHosts
}

func (interceptor *hostsRetryInterceptor) Intercept(req *http.Request, handler Handler) (resp *http.Response, err error) {
	if interceptor == nil || req == nil {
		return handler(req)
	}

	interceptor.options.init()

	// 不重试
	if interceptor.options.RetryMax <= 0 {
		return handler(req)
	}

	for i := 0; ; i++ {
		// Clone 防止后面 Handler 处理对 req 有污染
		reqBefore := cloneReq(req)

		resp, err = handler(req)

		retryDecision := interceptor.options.getRetryDecision(reqBefore, resp, err, i)
		if retryDecision == retrier.DontRetry {
			return resp, err
		}

		// 尝试冻结域名
		oldHost := req.URL.Host
		if interceptor.options.ShouldFreezeHost(req, resp, err) {
			if fErr := interceptor.options.HostProvider.Freeze(oldHost, err, interceptor.options.HostFreezeDuration); fErr != nil {
				break
			}
		}

		if i >= interceptor.options.RetryMax {
			break
		}

		// 尝试更换域名
		newHost, pErr := interceptor.options.HostProvider.Provider()
		if pErr != nil {
			break
		}
		if index := strings.Index(newHost, "://"); index >= 0 {
			newHost = newHost[(index + len("://")):]
		}
		if index := strings.Index(newHost, "/"); index >= 0 {
			newHost = newHost[:index]
		}
		if len(newHost) == 0 {
			break
		}

		if newHost != oldHost {
			urlString := req.URL.String()
			urlString = strings.Replace(urlString, oldHost, newHost, 1)
			u, ppErr := url.Parse(urlString)
			if ppErr != nil {
				break
			}

			reqBefore.Host = u.Host
			reqBefore.URL = u
		}

		req = reqBefore

		if req.Body != nil && req.GetBody != nil {
			if closer, ok := req.Body.(io.Closer); ok {
				closer.Close()
			}
			if req.Body, err = req.GetBody(); err != nil {
				return
			}
		}

		if resp != nil && resp.Body != nil {
			_ = internal_io.SinkAll(resp.Body)
			resp.Body.Close()
		}
	}
	return resp, err
}
