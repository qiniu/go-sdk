package clientv2

import (
	"context"
	"io"
	"math/rand"
	"net"
	"net/http"
	"time"

	clientv1 "github.com/qiniu/go-sdk/v7/client"
	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
	"github.com/qiniu/go-sdk/v7/storagev2/backoff"
	"github.com/qiniu/go-sdk/v7/storagev2/chooser"
	"github.com/qiniu/go-sdk/v7/storagev2/resolver"
	"github.com/qiniu/go-sdk/v7/storagev2/retrier"
)

type (
	bufferResponseContextKey struct{}

	SimpleRetryConfig struct {
		RetryMax      int                  // 最大重试次数
		RetryInterval func() time.Duration // 重试时间间隔 v1
		Backoff       backoff.Backoff      // 重试时间间隔 v2，优先级高于 RetryInterval
		ShouldRetry   func(req *http.Request, resp *http.Response, err error) bool
		Resolver      resolver.Resolver // 主备域名解析器
		Chooser       chooser.Chooser   // IP 选择器
		Retrier       retrier.Retrier   // 重试器

		BeforeResolve func(*http.Request)                                         // 域名解析前回调函数
		AfterResolve  func(*http.Request, []net.IP)                               // 域名解析后回调函数
		ResolveError  func(*http.Request, error)                                  // 域名解析错误回调函数
		BeforeBackoff func(*http.Request, *retrier.RetrierOptions, time.Duration) // 退避前回调函数
		AfterBackoff  func(*http.Request, *retrier.RetrierOptions, time.Duration) // 退避后回调函数
		BeforeRequest func(*http.Request, *retrier.RetrierOptions)                // 请求前回调函数
		AfterResponse func(*http.Response, *retrier.RetrierOptions, error)        // 请求后回调函数
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
}

func (c *SimpleRetryConfig) getRetryInterval(ctx context.Context, attempts int) time.Duration {
	if bf := c.Backoff; bf != nil {
		return bf.Time(ctx, &backoff.BackoffOptions{Attempts: attempts})
	}
	if ri := c.RetryInterval; ri != nil {
		return ri()
	}
	return defaultRetryInterval()
}

var errorRetrier = retrier.NewErrorRetrier()

func (c *SimpleRetryConfig) getRetryDecision(req *http.Request, resp *http.Response, err error, attempts int) retrier.RetryDecision {
	if c.ShouldRetry != nil {
		if c.ShouldRetry(req, resp, err) {
			return retrier.RetryRequest
		} else {
			return retrier.DontRetry
		}
	} else {
		r := errorRetrier
		if c.Retrier != nil {
			r = c.Retrier
		}
		return r.Retry(resp, err, &retrier.RetrierOptions{Attempts: attempts})
	}
}

func NewSimpleRetryInterceptor(config SimpleRetryConfig) Interceptor {
	return &simpleRetryInterceptor{config: config}
}

func (interceptor *simpleRetryInterceptor) Priority() InterceptorPriority {
	return InterceptorPriorityRetrySimple
}

func (interceptor *simpleRetryInterceptor) Intercept(req *http.Request, handler Handler) (resp *http.Response, err error) {
	var chosenIPs []net.IP

	if interceptor == nil || req == nil {
		return interceptor.callHandler(req, &retrier.RetrierOptions{Attempts: 0}, handler)
	}

	interceptor.config.init()

	hostname := req.URL.Hostname()
	resolvedIPs := interceptor.resolve(req, hostname)

	// 可能会被重试多次
	for i := 0; ; i++ {
		req, chosenIPs = interceptor.choose(req, resolvedIPs, hostname)
		// Clone 防止后面 Handler 处理对 req 有污染
		reqBefore := cloneReq(req)
		resp, err = interceptor.callHandler(req, &retrier.RetrierOptions{Attempts: i}, handler)

		retryDecision := interceptor.config.getRetryDecision(reqBefore, resp, err, i)
		if retryDecision == retrier.DontRetry {
			interceptor.feedbackGood(req, hostname, chosenIPs)
			return resp, err
		}
		interceptor.feedbackBad(req, hostname, chosenIPs)

		req = reqBefore

		if retryDecision == retrier.TryNextHost || i >= interceptor.config.RetryMax {
			break
		}

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

		interceptor.backoff(req, i)
	}
	return resp, err
}

func (interceptor *simpleRetryInterceptor) callHandler(req *http.Request, options *retrier.RetrierOptions, handler Handler) (resp *http.Response, err error) {
	if interceptor.config.BeforeRequest != nil {
		interceptor.config.BeforeRequest(req, options)
	}
	resp, err = handler(req)
	if interceptor.config.AfterResponse != nil {
		interceptor.config.AfterResponse(resp, options, err)
	}
	return
}

func (interceptor *simpleRetryInterceptor) resolve(req *http.Request, hostname string) []net.IP {
	var (
		ips []net.IP
		err error
	)
	if resolver := interceptor.config.Resolver; resolver != nil {
		if interceptor.config.BeforeResolve != nil {
			interceptor.config.BeforeResolve(req)
		}
		if ips, err = resolver.Resolve(req.Context(), hostname); err == nil {
			if interceptor.config.AfterResolve != nil {
				interceptor.config.AfterResolve(req, ips)
			}
		} else if err != nil && interceptor.config.ResolveError != nil {
			interceptor.config.ResolveError(req, err)
		}
	}
	return ips
}

func (interceptor *simpleRetryInterceptor) choose(req *http.Request, ips []net.IP, hostname string) (*http.Request, []net.IP) {
	if len(ips) > 0 {
		if cs := interceptor.config.Chooser; cs != nil {
			ips = cs.Choose(req.Context(), ips, &chooser.ChooseOptions{Domain: hostname})
		}
		req = req.WithContext(clientv1.WithResolvedIPs(req.Context(), hostname, ips))
	}
	return req, ips
}

func (interceptor *simpleRetryInterceptor) feedbackGood(req *http.Request, hostname string, ips []net.IP) {
	if len(ips) > 0 {
		if cs := interceptor.config.Chooser; cs != nil {
			cs.FeedbackGood(req.Context(), ips, &chooser.FeedbackOptions{Domain: hostname})
		}
	}
}

func (interceptor *simpleRetryInterceptor) feedbackBad(req *http.Request, hostname string, ips []net.IP) {
	if len(ips) > 0 {
		if cs := interceptor.config.Chooser; cs != nil {
			cs.FeedbackBad(req.Context(), ips, &chooser.FeedbackOptions{Domain: hostname})
		}
	}
}

func (interceptor *simpleRetryInterceptor) backoff(req *http.Request, attempts int) {
	retryInterval := interceptor.config.getRetryInterval(req.Context(), attempts)
	if interceptor.config.BeforeBackoff != nil {
		interceptor.config.BeforeBackoff(req, &retrier.RetrierOptions{Attempts: attempts}, retryInterval)
	}
	if retryInterval >= time.Microsecond {
		time.Sleep(retryInterval)
	}
	if interceptor.config.AfterBackoff != nil {
		interceptor.config.AfterBackoff(req, &retrier.RetrierOptions{Attempts: attempts}, retryInterval)
	}
}

func bufferResponse(resp *http.Response) error {
	buffer, err := internal_io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body = internal_io.NewBytesNopCloser(buffer)
	return nil
}

func defaultRetryInterval() time.Duration {
	return time.Duration(50+rand.Int()%50) * time.Millisecond
}
