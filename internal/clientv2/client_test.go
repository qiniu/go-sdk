//go:build unit
// +build unit

package clientv2

import (
	"fmt"
	"net/http"
	"testing"
)

const headerKey = "request"

type testClient struct {
	statusCode int
}

func (t testClient) Do(req *http.Request) (*http.Response, error) {
	value := req.Header.Get(headerKey)
	value += " -> Do"
	req.Header.Set(headerKey, value)
	fmt.Printf("=== Client Do ===\n")
	return &http.Response{
		Request:    req,
		StatusCode: t.statusCode,
		Header:     req.Header,
	}, nil
}

func TestInterceptor(t *testing.T) {

	// 不配置拦截器
	c := NewClient(&testClient{})
	resp, _ := c.Do(&http.Request{
		Header: http.Header{},
	})

	v := resp.Header.Get(headerKey)
	if v != " -> Do" {
		t.Fatal()
	}

	// 配置多个拦截器
	interceptor01 := NewSimpleInterceptor(func(req *http.Request, handler Handler) (*http.Response, error) {
		value := req.Header.Get(headerKey)
		value += " -> request-01"
		req.Header.Set(headerKey, value)

		rep, err := handler(req)

		value = rep.Header.Get(headerKey)
		value += " -> response-01"
		rep.Header.Set(headerKey, value)
		return rep, err
	})

	interceptor02 := NewSimpleInterceptor(func(req *http.Request, handler Handler) (*http.Response, error) {
		value := req.Header.Get(headerKey)
		value += " -> request-02"
		req.Header.Set(headerKey, value)

		rep, err := handler(req)

		value = rep.Header.Get(headerKey)
		value += " -> response-02"
		rep.Header.Set(headerKey, value)
		return rep, err
	})

	interceptor03 := NewSimpleInterceptor(func(req *http.Request, handler Handler) (*http.Response, error) {
		value := req.Header.Get(headerKey)
		value += " -> request-03"
		req.Header.Set(headerKey, value)

		rep, err := handler(req)

		value = rep.Header.Get(headerKey)
		value += " -> response-03"
		rep.Header.Set(headerKey, value)
		return rep, err
	})

	c = NewClient(&testClient{}, interceptor01, interceptor02, interceptor03)
	resp, _ = c.Do(&http.Request{
		Header: http.Header{},
	})

	v = resp.Header.Get(headerKey)
	if v != " -> request-01 -> request-02 -> request-03 -> Do -> response-03 -> response-02 -> response-01" {
		t.Fatalf("Unexpected header value: %s", v)
	}
}
