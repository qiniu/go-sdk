package clientv2

import (
	clientV1 "github.com/qiniu/go-sdk/v7/client"
	"net/http"
	"sort"
)

type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

type Handler func(req *http.Request) (*http.Response, error)

type client struct {
	coreClient   Client
	interceptors []Interceptor
}

func NewClient(cli Client, interceptors ...Interceptor) Client {
	if cli == nil {
		cli = http.DefaultClient
	}

	var is Interceptors = interceptors
	is = append(is, newDefaultHeaderInterceptor())
	is = append(is, newDebugInterceptor())
	sort.Sort(is)

	return &client{
		coreClient:   cli,
		interceptors: is,
	}
}

func (c *client) Do(req *http.Request) (*http.Response, error) {
	handler := func(req *http.Request) (*http.Response, error) {
		return c.coreClient.Do(req)
	}

	interceptors := c.interceptors

	// 反转
	for i, j := 0, len(interceptors)-1; i < j; i, j = i+1, j-1 {
		interceptors[i], interceptors[j] = interceptors[j], interceptors[i]
	}

	for _, interceptor := range interceptors {
		h := handler
		i := interceptor
		handler = func(r *http.Request) (*http.Response, error) {
			return i.Intercept(r, h)
		}
	}

	resp, err := handler(req)
	if err != nil {
		return resp, err
	}

	if resp == nil {
		return nil, &clientV1.ErrorInfo{
			Code: -999,
			Err:  "unknown error, no response",
		}
	}

	if resp.StatusCode/100 != 2 {
		return resp, clientV1.ResponseError(resp)
	}

	return resp, nil
}

func Do(c Client, options RequestOptions) (*http.Response, error) {
	req, err := NewRequest(options)
	if err != nil {
		return nil, err
	}

	return c.Do(req)
}

func DoAndParseJsonResponse(c Client, options RequestOptions, ret interface{}) (*http.Response, error) {
	resp, err := Do(c, options)
	if err != nil {
		return resp, err
	}

	if ret == nil || resp.ContentLength == 0 {
		return resp, nil
	}

	if dErr := clientV1.DecodeJsonFromReader(resp.Body, ret); dErr != nil {
		return resp, err
	}

	return resp, nil
}
