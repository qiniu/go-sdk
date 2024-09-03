package clientv2

import (
	"net/http"
	"reflect"
	"sort"
	"sync"

	clientV1 "github.com/qiniu/go-sdk/v7/client"
	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
)

type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

type Handler func(req *http.Request) (*http.Response, error)

type client struct {
	coreClient        Client
	interceptors      interceptorList
	interceptorsMutex sync.RWMutex
}

func NewClient(cli Client, interceptors ...Interceptor) Client {
	if cli == nil {
		cli = NewClientWithClientV1(&clientV1.DefaultClient)
	}

	var is interceptorList = interceptors
	is = append(is, newDefaultHeaderInterceptor())
	is = append(is, newDebugInterceptor())

	if c, ok := cli.(*client); ok {
		c.addInterceptors(is)
		return c
	}

	sort.Sort(is)
	return &client{
		coreClient:   cli,
		interceptors: is,
	}
}

func (c *client) addInterceptors(is interceptorList) {
	c.interceptorsMutex.Lock()
	defer c.interceptorsMutex.Unlock()

	for _, i := range is {
		c.addInterceptor(i)
	}
	sort.Sort(c.interceptors)
}

func (c *client) addInterceptor(interceptor Interceptor) {
	for _, i := range c.interceptors {
		if reflect.TypeOf(i) == reflect.TypeOf(interceptor) {
			return
		}
	}
	c.interceptors = append(c.interceptors, interceptor)
}

func (c *client) mergeInterceptors(req *http.Request) interceptorList {
	c.interceptorsMutex.RLock()
	defer c.interceptorsMutex.RUnlock()

	intercetorsFromRequest := getIntercetorsFromRequest(req)
	newInterceptorList := make(interceptorList, 0, len(c.interceptors)+len(intercetorsFromRequest))
	if len(intercetorsFromRequest) == 0 {
		newInterceptorList = append(newInterceptorList, c.interceptors...)
	} else if len(c.interceptors) == 0 {
		newInterceptorList = intercetorsFromRequest
	} else {
		newInterceptorList = append(newInterceptorList, c.interceptors...)
		newInterceptorList = append(newInterceptorList, intercetorsFromRequest...)
		sort.Sort(newInterceptorList)
	}
	return newInterceptorList
}

func (c *client) Do(req *http.Request) (*http.Response, error) {
	handler := func(req *http.Request) (*http.Response, error) {
		return c.coreClient.Do(req)
	}
	newInterceptorList := c.mergeInterceptors(req)
	for _, interceptor := range newInterceptorList {
		h := handler
		i := interceptor
		handler = func(r *http.Request) (*http.Response, error) {
			return i.Intercept(r, h)
		}
	}

	return handleResponseAndError(handler(req))
}

func Do(c Client, options RequestParams) (*http.Response, error) {
	req, err := NewRequest(options)
	if err != nil {
		return nil, err
	}

	return handleResponseAndError(c.Do(req))
}

func handleResponseAndError(resp *http.Response, err error) (*http.Response, error) {
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

func DoAndDecodeJsonResponse(c Client, options RequestParams, ret interface{}) error {
	resp, err := Do(c, options)
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = internal_io.SinkAll(resp.Body)
			resp.Body.Close()
		}
	}()

	if err != nil {
		return err
	}

	if ret == nil || resp.ContentLength == 0 {
		return nil
	}

	if err = clientV1.DecodeJsonFromReader(resp.Body, ret); err != nil {
		return err
	}

	return nil
}

type clientV1Wrapper struct {
	c *clientV1.Client
}

func (c *clientV1Wrapper) Do(req *http.Request) (*http.Response, error) {
	return c.c.Do(req.Context(), req)
}

func NewClientWithClientV1(c *clientV1.Client) Client {
	if c == nil {
		c = &clientV1.DefaultClient
	}
	return &clientV1Wrapper{
		c: c,
	}
}
