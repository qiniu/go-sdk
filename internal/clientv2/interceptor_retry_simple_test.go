//go:build unit
// +build unit

package clientv2

import (
	"net/http"
	"testing"
	"time"
)

func TestSimpleAlwaysRetryInterceptor(t *testing.T) {

	retryMax := 1
	rInterceptor := NewSimpleRetryInterceptor(RetryOptions{
		RetryMax: retryMax,
		RetryInterval: func() time.Duration {
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

	c := NewClient(&testClient{}, rInterceptor, interceptor)

	start := time.Now()
	resp, _ := Do(c, RequestParams{
		Context:     nil,
		Method:      "",
		Url:         "https://aaa.com",
		Header:      nil,
		BodyCreator: nil,
	})
	duration := float32(time.Now().Unix() - start.Unix())

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
}

func TestSimpleNotRetryInterceptor(t *testing.T) {

	retryMax := 1
	rInterceptor := NewSimpleRetryInterceptor(RetryOptions{
		RetryMax: retryMax,
		RetryInterval: func() time.Duration {
			return time.Second
		},
		// 默认状态码是 400，400 不重试
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

	c := NewClient(&testClient{statusCode: 400}, rInterceptor, interceptor)

	start := time.Now()
	resp, _ := Do(c, RequestParams{
		Context:     nil,
		Method:      "",
		Url:         "https://aaa.com",
		Header:      nil,
		BodyCreator: nil,
	})
	duration := float32(time.Now().Unix() - start.Unix())

	// 不重试，只执行一次，不等待
	if duration != 0 {
		t.Fatalf("retry interval may be error")
	}

	if doCount != 1 {
		t.Fatalf("retry count is not 1")
	}

	value := resp.Header.Get(headerKey)
	if value != " -> request -> Do -> response" {
		t.Fatalf("retry flow error")
	}
}
