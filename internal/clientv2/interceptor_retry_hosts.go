package clientv2

import (
	"github.com/qiniu/go-sdk/v7/internal/hostprovider"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HostsRetryOptions struct {
	RetryOptions       RetryOptions              // 主备域名重试参数
	HostFreezeDuration time.Duration             // 主备域名冻结时间（默认：600s），当一个域名请求失败被冻结的时间
	HostProvider       hostprovider.HostProvider // 备用域名获取方法
	ShouldFreezeHost   func(req *http.Request, resp *http.Response, err error) bool
}

func (o *HostsRetryOptions) init() {
	if o.RetryOptions.ShouldRetry == nil {
		o.RetryOptions.ShouldRetry = func(req *http.Request, resp *http.Response, err error) bool {
			return isHostRetryable(req, resp, err)
		}
	}
	if o.RetryOptions.RetryMax <= 0 {
		o.RetryOptions.RetryMax = 1
	}

	o.RetryOptions.Init()

	if o.HostFreezeDuration <= time.Millisecond {
		o.HostFreezeDuration = 600 * time.Second
	}

	if o.ShouldFreezeHost == nil {
		o.ShouldFreezeHost = func(req *http.Request, resp *http.Response, err error) bool {
			return true
		}
	}
}

type hostsRetryInterceptor struct {
	options HostsRetryOptions
}

func NewHostsRetryInterceptor(options HostsRetryOptions) Interceptor {
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
	if interceptor.options.RetryOptions.RetryMax == 0 {
		return handler(req)
	}

	for i := 0; ; i++ {
		// Clone 防止后面 Handler 处理对 req 有污染
		reqBefore := cloneReq(req.Context(), req)
		resp, err = handler(req)

		if !interceptor.options.RetryOptions.ShouldRetry(reqBefore, resp, err) {
			return resp, err
		}

		// 尝试冻结域名
		oldHost := req.URL.Host
		if interceptor.options.ShouldFreezeHost(req, resp, err) {
			if fErr := interceptor.options.HostProvider.Freeze(oldHost, err, interceptor.options.HostFreezeDuration); fErr != nil {
				break
			}
		}

		if i >= interceptor.options.RetryOptions.RetryMax {
			break
		}

		// 尝试更换域名
		newHost, pErr := interceptor.options.HostProvider.Provider()
		if pErr != nil {
			break
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

		retryInterval := interceptor.options.RetryOptions.RetryInterval()
		if retryInterval <= time.Millisecond {
			continue
		}
		time.Sleep(retryInterval)
	}
	return resp, err
}

func isHostRetryable(req *http.Request, resp *http.Response, err error) bool {
	return isRequestRetryable(req) && (isResponseRetryable(resp) || IsErrorRetryable(err))
}
