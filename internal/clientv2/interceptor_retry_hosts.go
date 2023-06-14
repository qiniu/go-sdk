package clientv2

import (
	"github.com/qiniu/go-sdk/v7/internal/hostprovider"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HostsRetryOptions struct {
	RetryOptions     RetryOptions
	ShouldFreezeHost func(req *http.Request, resp *http.Response, err error) bool

	// 主备域名冻结时间（默认：600s），当一个域名请求失败（单个域名会被重试 TryTimes 次），会被冻结一段时间，使用备用域名进行重试，在冻结时间内，域名不能被使用，当一个操作中所有域名竣备冻结操作不在进行重试，返回最后一次操作的错误。
	HostFreezeDuration time.Duration
	HostProvider       hostprovider.HostProvider
}

func (o *HostsRetryOptions) init() {
	o.RetryOptions.Init()
	if o.RetryOptions.RetryMax <= 0 {
		o.RetryOptions.RetryMax = 2
	}

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

func (r *hostsRetryInterceptor) Priority() InterceptorPriority {
	return InterceptorPriorityRetryHosts
}

func (r *hostsRetryInterceptor) Intercept(req *http.Request, handler Handler) (resp *http.Response, err error) {
	r.options.init()

	// 不重试
	if r.options.RetryOptions.RetryMax == 0 {
		return handler(req)
	}

	for i := 0; ; i++ {
		// Clone 防止后面 Handler 处理对 req 有污染
		reqBefore := req.Clone(req.Context())
		resp, err = handler(req)

		if !r.options.RetryOptions.ShouldRetry(reqBefore, resp, err) {
			return resp, err
		}

		// 尝试冻结域名
		oldHost := req.URL.Host
		if r.options.ShouldFreezeHost(req, resp, err) {
			if fErr := r.options.HostProvider.Freeze(oldHost, err, r.options.HostFreezeDuration); fErr != nil {
				break
			}
		}

		if i >= r.options.RetryOptions.RetryMax {
			break
		}

		// 尝试更换域名
		newHost, pErr := r.options.HostProvider.Provider()
		if pErr != nil {
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

		retryInterval := r.options.RetryOptions.RetryInterval()
		if retryInterval <= time.Millisecond {
			continue
		}
		time.Sleep(retryInterval)
	}
	return resp, err
}

func isHostRetryable(req *http.Request, resp *http.Response, err error) bool {
	return isRequestSimpleRetryable(req) && isResponseSimpleRetryable(resp) && isErrorSimpleRetryable(err)
}
