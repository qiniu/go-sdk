//go:build unit
// +build unit

package clientv2

import (
	"math"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/backoff"
	"github.com/qiniu/go-sdk/v7/storagev2/chooser"
	"github.com/qiniu/go-sdk/v7/storagev2/resolver"
	"github.com/qiniu/go-sdk/v7/storagev2/retrier"
)

func TestSimpleAlwaysRetryInterceptor(t *testing.T) {

	retryMax := 1
	doCount := 0
	callbackedCount := 0
	rInterceptor := NewSimpleRetryInterceptor(SimpleRetryConfig{
		RetryMax: retryMax,
		RetryInterval: func() time.Duration {
			return time.Second
		},
		ShouldRetry: func(req *http.Request, resp *http.Response, err error) bool {
			return true
		},
		Resolver: resolver.NewDefaultResolver(),
		BeforeResolve: func(req *http.Request) {
			callbackedCount += 1
		},
		AfterResolve: func(req *http.Request, ips []net.IP) {
			callbackedCount += 1
			if len(ips) == 0 {
				t.Fatal("unexpected ips", ips)
			}
		},
		ResolveError: func(req *http.Request, err error) {
			t.Fatal("unexpected error", err)
		},
		Chooser: chooser.NewDirectChooser(),
		BeforeBackoff: func(req *http.Request, options *retrier.RetrierOptions, duration time.Duration) {
			callbackedCount += 1
			if options.Attempts != 0 {
				t.Fatalf("unexpected attempts:%d", options.Attempts)
			}
			if duration != time.Second {
				t.Fatalf("unexpected duration:%v", duration)
			}
		},
		AfterBackoff: func(req *http.Request, options *retrier.RetrierOptions, duration time.Duration) {
			callbackedCount += 1
			if options.Attempts != 0 {
				t.Fatalf("unexpected attempts:%d", options.Attempts)
			}
			if duration != time.Second {
				t.Fatalf("unexpected duration:%v", duration)
			}
		},
		BeforeRequest: func(req *http.Request, options *retrier.RetrierOptions) {
			callbackedCount += 1
			if options.Attempts != doCount {
				t.Fatal("unexpected attempts", options.Attempts)
			}
		},
		AfterResponse: func(req *http.Request, resp *http.Response, options *retrier.RetrierOptions, err error) {
			callbackedCount += 1
			if options.Attempts != (doCount - 1) {
				t.Fatal("unexpected attempts", options.Attempts)
			}
			if err != nil {
				t.Fatal("unexpected error", err)
			}
		},
	})

	interceptor := NewSimpleInterceptor(func(req *http.Request, handler Handler) (*http.Response, error) {
		doCount += 1

		value := req.Header.Get(headerKey)
		value += " -> request"
		req.Header.Set(headerKey, value)

		resp, err := handler(req)

		value = resp.Header.Get(headerKey)
		value += " -> response"
		resp.Header.Set(headerKey, value)
		return resp, err
	})

	c := NewClient(&testClient{}, rInterceptor, interceptor)

	start := time.Now()
	resp, _ := Do(c, RequestParams{
		Context: nil,
		Method:  "",
		Url:     "https://aaa.com",
		Header:  nil,
		GetBody: nil,
	})
	duration := float32(time.Now().UnixNano()-start.UnixNano()) / 1e9

	if duration > float32(doCount-1)+0.3 || duration < float32(doCount-1)-0.3 {
		t.Fatalf("retry interval may be error:%f", duration)
	}

	if (retryMax + 1) != doCount {
		t.Fatalf("retry count is not 2")
	}

	value := resp.Header.Get(headerKey)
	if value != " -> request -> Do -> response" {
		t.Fatalf("retry flow error")
	}
	if callbackedCount != 8 {
		t.Fatalf("unexpected callbackedCount: %d", callbackedCount)
	}
}

func TestSimpleNotRetryInterceptor(t *testing.T) {

	retryMax := 1
	doCount := 0
	callbackedCount := 0
	rInterceptor := NewSimpleRetryInterceptor(SimpleRetryConfig{
		RetryMax: retryMax,
		RetryInterval: func() time.Duration {
			return time.Second
		},
		// 默认状态码是 400，400 不重试
		//ShouldRetry: func(req *http.Request, resp *http.Response, err error) bool {
		//	return true
		//},
		Resolver: resolver.NewDefaultResolver(),
		BeforeResolve: func(req *http.Request) {
			callbackedCount += 1
		},
		AfterResolve: func(req *http.Request, ips []net.IP) {
			callbackedCount += 1
			if len(ips) == 0 {
				t.Fatal("unexpected ips", ips)
			}
		},
		ResolveError: func(req *http.Request, err error) {
			t.Fatal("unexpected error", err)
		},
		Chooser: chooser.NewDirectChooser(),
		BeforeBackoff: func(req *http.Request, options *retrier.RetrierOptions, duration time.Duration) {
			callbackedCount += 1
			if options.Attempts != 0 {
				t.Fatalf("unexpected attempts:%d", options.Attempts)
			}
			if duration != time.Second {
				t.Fatalf("unexpected duration:%v", duration)
			}
		},
		AfterBackoff: func(req *http.Request, options *retrier.RetrierOptions, duration time.Duration) {
			callbackedCount += 1
			if options.Attempts != 0 {
				t.Fatalf("unexpected attempts:%d", options.Attempts)
			}
			if duration != time.Second {
				t.Fatalf("unexpected duration:%v", duration)
			}
		},
		BeforeRequest: func(req *http.Request, options *retrier.RetrierOptions) {
			callbackedCount += 1
			if options.Attempts != doCount {
				t.Fatal("unexpected attempts", options.Attempts)
			}
		},
		AfterResponse: func(req *http.Request, resp *http.Response, options *retrier.RetrierOptions, err error) {
			callbackedCount += 1
			if options.Attempts != (doCount - 1) {
				t.Fatal("unexpected attempts", options.Attempts)
			}
			if err != nil {
				t.Fatal("unexpected error", err)
			}
		},
	})

	interceptor := NewSimpleInterceptor(func(req *http.Request, handler Handler) (*http.Response, error) {
		doCount += 1

		value := req.Header.Get(headerKey)
		value += " -> request"
		req.Header.Set(headerKey, value)

		resp, err := handler(req)

		value = resp.Header.Get(headerKey)
		value += " -> response"
		resp.Header.Set(headerKey, value)
		return resp, err
	})

	c := NewClient(&testClient{statusCode: 400}, rInterceptor, interceptor)

	start := time.Now()
	resp, _ := Do(c, RequestParams{
		Context: nil,
		Method:  "",
		Url:     "https://aaa.com",
		Header:  nil,
		GetBody: nil,
	})
	duration := float32(time.Now().UnixNano()-start.UnixNano()) / 1e9

	// 不重试，只执行一次，不等待
	if duration > 0.3 {
		t.Fatalf("retry interval may be error")
	}

	if doCount != 1 {
		t.Fatalf("retry count is not 1")
	}

	value := resp.Header.Get(headerKey)
	if value != " -> request -> Do -> response" {
		t.Fatalf("retry flow error")
	}
	if callbackedCount != 4 {
		t.Fatalf("unexpected callbackedCount: %d", callbackedCount)
	}
}

func TestRetryInterceptorWithBackoff(t *testing.T) {
	retryMax := 5
	doCount := 0
	callbackedCount := 0
	rInterceptor := NewSimpleRetryInterceptor(SimpleRetryConfig{
		RetryMax: retryMax,
		Backoff:  backoff.NewExponentialBackoff(100*time.Millisecond, 2),
		ShouldRetry: func(req *http.Request, resp *http.Response, err error) bool {
			return true
		},
		Resolver: resolver.NewDefaultResolver(),
		BeforeResolve: func(req *http.Request) {
			callbackedCount += 1
		},
		AfterResolve: func(req *http.Request, ips []net.IP) {
			callbackedCount += 1
			if len(ips) == 0 {
				t.Fatal("unexpected ips", ips)
			}
		},
		ResolveError: func(req *http.Request, err error) {
			t.Fatal("unexpected error", err)
		},
		Chooser: chooser.NewDirectChooser(),
		BeforeBackoff: func(req *http.Request, options *retrier.RetrierOptions, duration time.Duration) {
			callbackedCount += 1
			if options.Attempts != (doCount - 1) {
				t.Fatalf("unexpected attempts:%d", options.Attempts)
			}
			if duration != 100*time.Millisecond*time.Duration(math.Pow(2, float64(options.Attempts))) {
				t.Fatalf("unexpected duration:%v", duration)
			}
		},
		AfterBackoff: func(req *http.Request, options *retrier.RetrierOptions, duration time.Duration) {
			callbackedCount += 1
			if options.Attempts != (doCount - 1) {
				t.Fatalf("unexpected attempts:%d", options.Attempts)
			}
			if duration != 100*time.Millisecond*time.Duration(math.Pow(2, float64(options.Attempts))) {
				t.Fatalf("unexpected duration:%v", duration)
			}
		},
		BeforeRequest: func(req *http.Request, options *retrier.RetrierOptions) {
			callbackedCount += 1
			if options.Attempts != doCount {
				t.Fatal("unexpected attempts", options.Attempts)
			}
		},
		AfterResponse: func(req *http.Request, resp *http.Response, options *retrier.RetrierOptions, err error) {
			callbackedCount += 1
			if options.Attempts != (doCount - 1) {
				t.Fatal("unexpected attempts", options.Attempts)
			}
			if err != nil {
				t.Fatal("unexpected error", err)
			}
		},
	})

	interceptor := NewSimpleInterceptor(func(req *http.Request, handler Handler) (*http.Response, error) {
		doCount += 1

		value := req.Header.Get(headerKey)
		value += " -> request"
		req.Header.Set(headerKey, value)

		resp, err := handler(req)

		value = resp.Header.Get(headerKey)
		value += " -> response"
		resp.Header.Set(headerKey, value)
		return resp, err
	})

	c := NewClient(&testClient{}, rInterceptor, interceptor)

	start := time.Now()
	resp, _ := Do(c, RequestParams{
		Context: nil,
		Method:  "",
		Url:     "https://aaa.com",
		Header:  nil,
		GetBody: nil,
	})
	duration := float32(time.Now().UnixNano()-start.UnixNano()) / float32(time.Millisecond)

	if duration > 3100+50 || duration < 3100-50 {
		t.Fatalf("retry interval may be error:%f", duration)
	}

	if (retryMax + 1) != doCount {
		t.Fatalf("retry count is not 2")
	}

	value := resp.Header.Get(headerKey)
	if value != " -> request -> Do -> response" {
		t.Fatalf("retry flow error")
	}
	if callbackedCount != 24 {
		t.Fatalf("unexpected callbackedCount: %d", callbackedCount)
	}
}
