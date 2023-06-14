//go:build unit
// +build unit

package clientv2

import (
	clientV1 "github.com/qiniu/go-sdk/v7/client"
	"github.com/qiniu/go-sdk/v7/internal/hostprovider"
	"net/http"
	"os"
	"testing"
	"time"
)

var (
	testAK     = os.Getenv("accessKey")
	testBucket = os.Getenv("QINIU_TEST_BUCKET")
)

func TestHostsAlwaysRetryInterceptor(t *testing.T) {
	clientV1.DebugMode = true

	hostA := "aaa.aa.com"
	hostB := "bbb.bb.com"
	hRetryMax := 2
	hRetryInterceptor := NewHostsRetryInterceptor(HostsRetryOptions{
		RetryOptions: RetryOptions{
			RetryMax: hRetryMax,
			RetryInterval: func() time.Duration {
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
	sRetryInterceptor := NewSimpleRetryInterceptor(RetryOptions{
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

	c := NewClient(&testClient{}, interceptor, hRetryInterceptor, sRetryInterceptor)

	start := time.Now()

	resp, _ := Do(c, RequestOptions{
		Context:     nil,
		Method:      RequestMethodGet,
		Url:         "https://" + hostA + "/path/123",
		Header:      nil,
		BodyCreator: nil,
	})
	duration := float32(time.Now().Unix() - start.Unix())

	if duration > float32(doCount-1)+0.1 || duration < float32(doCount-1)-0.1 {
		t.Fatalf("retry interval may be error")
	}

	if (retryMax+1)*2 != doCount {
		t.Fatalf("retry count is not error:%d", doCount)
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

	hostA := "aaa.aa.com"
	hostB := "bbb.bb.com"
	hRetryMax := 2
	hRetryInterceptor := NewHostsRetryInterceptor(HostsRetryOptions{
		RetryOptions: RetryOptions{
			RetryMax: hRetryMax,
			RetryInterval: func() time.Duration {
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
	sRetryInterceptor := NewSimpleRetryInterceptor(RetryOptions{
		RetryMax: retryMax,
		RetryInterval: func() time.Duration {
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

	resp, _ := Do(c, RequestOptions{
		Context:     nil,
		Method:      RequestMethodGet,
		Url:         "https://" + hostA + "/path/123",
		Header:      nil,
		BodyCreator: nil,
	})
	duration := float32(time.Now().Unix() - start.Unix())

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

func TestHostsRetryInterceptorByUcQuery(t *testing.T) {
	clientV1.DebugMode = true

	hostA := "aaa.aa.com"
	hostB := "uc.qbox.me"
	hRetryMax := 30
	hRetryInterceptor := NewHostsRetryInterceptor(HostsRetryOptions{
		RetryOptions: RetryOptions{
			RetryMax: hRetryMax,
			RetryInterval: func() time.Duration {
				return time.Second
			},
		},
		ShouldFreezeHost:   nil,
		HostFreezeDuration: 0,
		HostProvider:       hostprovider.NewWithHosts([]string{hostA, hostB}),
	})

	retryMax := 1
	sRetryInterceptor := NewSimpleRetryInterceptor(RetryOptions{
		RetryMax: retryMax,
		RetryInterval: func() time.Duration {
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

	start := time.Now()

	resp, err := Do(c, RequestOptions{
		Context:     nil,
		Method:      RequestMethodGet,
		Url:         "https://" + hostA + "/v4/query?ak=" + testAK + "&bucket=" + testBucket,
		Header:      nil,
		BodyCreator: nil,
	})

	if err != nil {
		t.Fatalf("request should success:%v", err)
	}

	duration := float32(time.Now().Unix() - start.Unix())

	if duration > float32(doCount-1)+0.2 || duration < float32(doCount-1)-0.2 {
		t.Fatalf("retry interval may be error")
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
