//go:build unit
// +build unit

package clientv2

import (
	"net/http"
	"testing"
	"time"

	clientV1 "github.com/qiniu/go-sdk/v7/client"
	"github.com/qiniu/go-sdk/v7/internal/hostprovider"
)

func TestHostsAlwaysRetryInterceptor(t *testing.T) {
	clientV1.DebugMode = true
	defer func() {
		clientV1.DebugMode = false
	}()

	hostA := "aaa.aa.com"
	hostB := "bbb.bb.com"
	hRetryMax := 2
	hRetryInterceptor := NewHostsRetryInterceptor(HostsRetryConfig{
		RetryConfig: RetryConfig{
			RetryMax: hRetryMax,
			RetryInterval: func(_ *RetryInfo) time.Duration {
				return time.Second
			},
			ShouldRetry: func(req *http.Request, resp *http.Response, err error) bool {
				return true
			},
		},
		ShouldFreezeHost:   nil,
		HostFreezeDuration: 0,
		HostProvider:       hostprovider.NewWithHosts([]string{hostA, hostB}),
	})

	retryMax := 1
	sRetryInterceptor := NewSimpleRetryInterceptor(SimpleRetryConfig{
		RetryMax: retryMax,
		RetryInterval: func(_ *RetryInfo) time.Duration {
			return time.Second
		},
		ShouldRetry: func(req *http.Request, resp *http.Response, err error) bool {
			return true
		},
	})

	doCount := 0
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

	c := NewClient(&testClient{}, interceptor, hRetryInterceptor, sRetryInterceptor)

	start := time.Now()

	resp, _ := Do(c, RequestParams{
		Context: nil,
		Method:  RequestMethodGet,
		Url:     "https://" + hostA + "/path/123",
		Header:  nil,
		GetBody: nil,
	})
	duration := float32(time.Now().UnixNano()-start.UnixNano()) / 1e9

	if (retryMax+1)*2 != doCount {
		t.Fatalf("retry count is not error:%d", doCount)
	}

	if duration > float32(doCount-1)+0.3 || duration < float32(doCount-1)-0.3 {
		t.Fatalf("retry interval may be error:%f", duration)
	}

	value := resp.Header.Get(headerKey)
	if value != " -> request -> Do -> response" {
		t.Fatalf("retry flow error")
	}

	if resp.Request.Host != hostB {
		t.Fatalf("retry host set error")
	}
}

func TestHostsNotRetryInterceptor(t *testing.T) {
	clientV1.DebugMode = true
	defer func() {
		clientV1.DebugMode = false
	}()

	hostA := "aaa.aa.com"
	hostB := "bbb.bb.com"
	hRetryMax := 2
	hRetryInterceptor := NewHostsRetryInterceptor(HostsRetryConfig{
		RetryConfig: RetryConfig{
			RetryMax: hRetryMax,
			RetryInterval: func(_ *RetryInfo) time.Duration {
				return time.Second
			},
			//ShouldRetry: func(req *http.Request, resp *http.Response, err error) bool {
			//	return true
			//},
		},
		ShouldFreezeHost:   nil,
		HostFreezeDuration: 0,
		HostProvider:       hostprovider.NewWithHosts([]string{hostA, hostB}),
	})

	retryMax := 1
	sRetryInterceptor := NewSimpleRetryInterceptor(SimpleRetryConfig{
		RetryMax: retryMax,
		RetryInterval: func(_ *RetryInfo) time.Duration {
			return time.Second
		},
		//ShouldRetry: func(req *http.Request, resp *http.Response, err error) bool {
		//	return true
		//},
	})

	doCount := 0
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

	c := NewClient(&testClient{statusCode: 400}, interceptor, hRetryInterceptor, sRetryInterceptor)

	start := time.Now()

	resp, _ := Do(c, RequestParams{
		Context: nil,
		Method:  RequestMethodGet,
		Url:     "https://" + hostA + "/path/123",
		Header:  nil,
		GetBody: nil,
	})
	duration := float32(time.Now().UnixNano()-start.UnixNano()) / 1e9

	if duration > float32(doCount-1)+0.1 || duration < float32(doCount-1)-0.1 {
		t.Fatalf("retry interval may be error")
	}

	if 1 != doCount {
		t.Fatalf("retry count is not error:%d", doCount)
	}

	value := resp.Header.Get(headerKey)
	if value != " -> request -> Do -> response" {
		t.Fatalf("retry flow error")
	}

	if resp.Request.Host != hostA {
		t.Fatalf("retry host set error")
	}
}

func TestHostsRetryInterceptorByRequest(t *testing.T) {
	clientV1.DebugMode = true
	defer func() {
		clientV1.DebugMode = false
	}()

	hostA := "aaa.aa.com"
	hostB := "www.qiniu.com"
	hRetryMax := 30
	hRetryInterceptor := NewHostsRetryInterceptor(HostsRetryConfig{
		RetryConfig: RetryConfig{
			RetryMax: hRetryMax,
			RetryInterval: func(_ *RetryInfo) time.Duration {
				return time.Second
			},
		},
		ShouldFreezeHost:   nil,
		HostFreezeDuration: 0,
		HostProvider:       hostprovider.NewWithHosts([]string{hostA, hostB}),
	})

	retryMax := 1
	sRetryInterceptor := NewSimpleRetryInterceptor(SimpleRetryConfig{
		RetryMax: retryMax,
		RetryInterval: func(_ *RetryInfo) time.Duration {
			return time.Second
		},
	})

	doCount := 0
	interceptor := NewSimpleInterceptor(func(req *http.Request, handler Handler) (*http.Response, error) {
		doCount += 1

		value := req.Header.Get(headerKey)
		value += " -> request"
		req.Header.Set(headerKey, value)

		resp, err := handler(req)
		if err != nil || resp == nil {
			return nil, err
		}

		value = resp.Header.Get(headerKey)
		value += " -> response"
		resp.Header.Set(headerKey, value)
		return resp, err
	})

	c := NewClient(nil, interceptor, hRetryInterceptor, sRetryInterceptor)
	resp, err := Do(c, RequestParams{
		Context: nil,
		Method:  RequestMethodGet,
		Url:     "https://" + hostA,
		Header:  nil,
		GetBody: nil,
	})

	if err != nil {
		t.Fatalf("request should success:%v", err)
	}

	if (retryMax+1)+1 != doCount {
		t.Fatalf("retry count is not error:%d", doCount)
	}

	value := resp.Header.Get(headerKey)
	if value != " -> response" {
		t.Fatalf("retry flow error")
	}

	if resp.Request.Host != hostB {
		t.Fatalf("retry host set error")
	}
}
