//go:build unit
// +build unit

package clientv2

import (
	"context"
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
		AfterResponse: func(resp *http.Response, options *retrier.RetrierOptions, err error) {
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
		AfterResponse: func(resp *http.Response, options *retrier.RetrierOptions, err error) {
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
	duration := time.Since(start)

	// 不重试，只执行一次，不等待
	if duration > 500*time.Millisecond {
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

func TestSimpleRetryInterceptorWithNoopResolver(t *testing.T) {
	doCount := 0
	resolveCount := 0
	rInterceptor := NewSimpleRetryInterceptor(SimpleRetryConfig{
		RetryMax: 0,
		Resolver: resolver.NewResolver(func(ctx context.Context, host string) ([]net.IP, error) {
			return nil, nil
		}),
		BeforeResolve: func(req *http.Request) {
			resolveCount += 1
		},
		AfterResolve: func(req *http.Request, ips []net.IP) {
			resolveCount += 1
			if len(ips) != 0 {
				t.Fatalf("expected empty ips, got %v", ips)
			}
		},
		ResolveError: func(req *http.Request, err error) {
			t.Fatal("unexpected resolve error", err)
		},
		Chooser: chooser.NewDirectChooser(),
	})

	interceptor := NewSimpleInterceptor(func(req *http.Request, handler Handler) (*http.Response, error) {
		doCount += 1
		resp, err := handler(req)
		return resp, err
	})

	c := NewClient(&testClient{statusCode: 200}, rInterceptor, interceptor)

	resp, err := Do(c, RequestParams{
		Context: nil,
		Method:  "",
		Url:     "https://aaa.com",
		Header:  nil,
		GetBody: nil,
	})
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	if doCount != 1 {
		t.Fatalf("expected 1 request, got %d", doCount)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	// 首次 resolve 调 BeforeResolve + AfterResolve，
	// ensureChoose 发现无可用 IP 后再次 bypassCache resolve 调 BeforeResolve + AfterResolve
	if resolveCount != 4 {
		t.Fatalf("expected 4 resolve callbacks, got %d", resolveCount)
	}
}

func TestRetryInterceptorWithBackoff(t *testing.T) {
	retryMax := 5
	doCount := 0
	callbackedCount := 0
	rInterceptor := NewSimpleRetryInterceptor(SimpleRetryConfig{
		RetryMax: retryMax,
		Backoff:  backoff.NewExponentialBackoff(1*time.Second, 2),
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
			if duration != 1*time.Second*time.Duration(math.Pow(2, float64(options.Attempts))) {
				t.Fatalf("unexpected duration:%v", duration)
			}
		},
		AfterBackoff: func(req *http.Request, options *retrier.RetrierOptions, duration time.Duration) {
			callbackedCount += 1
			if options.Attempts != (doCount - 1) {
				t.Fatalf("unexpected attempts:%d", options.Attempts)
			}
			if duration != 1*time.Second*time.Duration(math.Pow(2, float64(options.Attempts))) {
				t.Fatalf("unexpected duration:%v", duration)
			}
		},
		BeforeRequest: func(req *http.Request, options *retrier.RetrierOptions) {
			callbackedCount += 1
			if options.Attempts != doCount {
				t.Fatal("unexpected attempts", options.Attempts)
			}
		},
		AfterResponse: func(resp *http.Response, options *retrier.RetrierOptions, err error) {
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
	duration := time.Since(start)

	if d := duration - 31*time.Second; d >= 900*time.Millisecond || d <= -900*time.Millisecond {
		t.Fatalf("retry interval may be error:%v", duration)
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
