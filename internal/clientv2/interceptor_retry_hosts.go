package clientv2

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/qiniu/go-sdk/v7/internal/hostprovider"
	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
)

type HostsRetryConfig struct {
	RetryConfig        RetryConfig               // 主备域名重试参数
	HostFreezeDuration time.Duration             // 主备域名冻结时间（默认：600s），当一个域名请求失败被冻结的时间，最小 time.Millisecond
	HostProvider       hostprovider.HostProvider // 备用域名获取方法
	ShouldFreezeHost   func(req *http.Request, resp *http.Response, err error) bool
}

func (c *HostsRetryConfig) init() {
	if c.RetryConfig.ShouldRetry == nil {
		c.RetryConfig.ShouldRetry = func(req *http.Request, resp *http.Response, err error) bool {
			return isHostRetryable(req, resp, err)
		}
	}
	if c.RetryConfig.RetryMax < 0 {
		c.RetryConfig.RetryMax = 1
	}

	c.RetryConfig.init()

	if c.HostFreezeDuration < time.Millisecond {
		c.HostFreezeDuration = 600 * time.Second
	}

	if c.ShouldFreezeHost == nil {
		c.ShouldFreezeHost = func(req *http.Request, resp *http.Response, err error) bool {
			return true
		}
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
	if interceptor.options.RetryConfig.RetryMax <= 0 {
		return handler(req)
	}

	for i := 0; ; i++ {
		// Clone 防止后面 Handler 处理对 req 有污染
		reqBefore := cloneReq(req)
		resp, err = handler(req)

		if !interceptor.options.RetryConfig.ShouldRetry(reqBefore, resp, err) {
			return resp, err
		}

		// 尝试冻结域名
		oldHost := req.URL.Host
		if interceptor.options.ShouldFreezeHost(req, resp, err) {
			if fErr := interceptor.options.HostProvider.Freeze(oldHost, err, interceptor.options.HostFreezeDuration); fErr != nil {
				break
			}
		}

		if i >= interceptor.options.RetryConfig.RetryMax {
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

		if resp != nil && resp.Body != nil {
			_ = internal_io.SinkAll(resp.Body)
			resp.Body.Close()
		}

		retryInterval := interceptor.options.RetryConfig.RetryInterval()
		if retryInterval < time.Microsecond {
			continue
		}
		time.Sleep(retryInterval)
	}
	return resp, err
}

func isHostRetryable(req *http.Request, resp *http.Response, err error) bool {
	return isRequestRetryable(req) && (isResponseHostRetryable(resp) || IsErrorRetryable(err))
}

func isResponseHostRetryable(resp *http.Response) bool {
	if resp == nil {
		return false
	}

	statusCode := resp.StatusCode

	// 2xx
	if statusCode < 300 {
		return false
	}

	// 3xx
	if statusCode < 400 {
		return true
	}

	// 4xx
	if statusCode < 500 {
		return statusCode == 404
	}

	return isStatusCodeRetryable(statusCode)
}
