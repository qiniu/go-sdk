package clientv2

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/qiniu/go-sdk/v7/conf"
	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
)

const (
	RequestMethodGet    = http.MethodGet
	RequestMethodPut    = http.MethodPut
	RequestMethodPost   = http.MethodPost
	RequestMethodHead   = http.MethodHead
	RequestMethodDelete = http.MethodDelete
)

type GetRequestBody func(options *RequestParams) (io.ReadCloser, error)

func GetJsonRequestBody(object interface{}) (GetRequestBody, error) {
	reqBody, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}
	return func(o *RequestParams) (io.ReadCloser, error) {
		o.Header.Set("Content-Type", conf.CONTENT_TYPE_JSON)
		o.Header.Set("Content-Length", strconv.Itoa(len(reqBody)))
		return internal_io.NewReadSeekableNopCloser(bytes.NewReader(reqBody)), nil
	}, nil
}

func GetFormRequestBody(info map[string][]string) GetRequestBody {
	body := formStringInfo(info)
	return func(o *RequestParams) (io.ReadCloser, error) {
		o.Header.Set("Content-Type", conf.CONTENT_TYPE_FORM)
		o.Header.Set("Content-Length", strconv.Itoa(len(body)))
		return internal_io.NewReadSeekableNopCloser(strings.NewReader(body)), nil
	}
}

func formStringInfo(info map[string][]string) string {
	if len(info) == 0 {
		return ""
	}
	return url.Values(info).Encode()
}

type RequestParams struct {
	Context           context.Context
	Method            string
	Url               string
	Header            http.Header
	GetBody           GetRequestBody
	BufferResponse    bool
	OnRequestProgress RequestBodyProgress
}

func (o *RequestParams) init() {
	if o.Context == nil {
		o.Context = context.Background()
	}

	if len(o.Method) == 0 {
		o.Method = RequestMethodGet
	}

	if o.Header == nil {
		o.Header = http.Header{}
	}

	if o.GetBody == nil {
		o.GetBody = func(options *RequestParams) (io.ReadCloser, error) {
			return nil, nil
		}
	}
}

func NewRequest(options RequestParams) (req *http.Request, err error) {
	var (
		bodyWrapper   *requestBodyWrapperWithProgress = nil
		contentLength uint64
	)

	options.init()

	body, err := options.GetBody(&options)
	if err != nil {
		return nil, err
	}
	if options.OnRequestProgress != nil && body != nil {
		if contentLengthHeaderValue := options.Header.Get("Content-Length"); contentLengthHeaderValue != "" {
			contentLength, _ = strconv.ParseUint(contentLengthHeaderValue, 10, 64)
		}
		bodyWrapper = &requestBodyWrapperWithProgress{body: body, expectedSize: contentLength, callback: options.OnRequestProgress}
	}
	if bodyWrapper != nil {
		req, err = http.NewRequest(options.Method, options.Url, bodyWrapper)
	} else {
		req, err = http.NewRequest(options.Method, options.Url, body)
	}
	if err != nil {
		return
	}
	if options.Context != nil {
		req = req.WithContext(options.Context)
	}
	if options.BufferResponse {
		req = req.WithContext(context.WithValue(options.Context, bufferResponseContextKey{}, struct{}{}))
	}
	req.Header = options.Header
	if options.GetBody != nil && body != nil && body != http.NoBody {
		req.GetBody = func() (io.ReadCloser, error) {
			reqBody, err := options.GetBody(&options)
			if err != nil {
				return nil, err
			}
			if bodyWrapper != nil {
				return &requestBodyWrapperWithProgress{
					body:         reqBody,
					expectedSize: contentLength,
					callback:     options.OnRequestProgress,
				}, nil
			} else {
				return reqBody, nil
			}
		}
	}
	return
}

type (
	RequestBodyProgress            func(uint64, uint64)
	requestBodyWrapperWithProgress struct {
		body                       io.ReadCloser
		haveReadSize, expectedSize uint64
		callback                   RequestBodyProgress
	}
)

func (wrapper *requestBodyWrapperWithProgress) Read(p []byte) (n int, err error) {
	n, err = wrapper.body.Read(p)
	if callback := wrapper.callback; callback != nil && n > 0 {
		wrapper.haveReadSize += uint64(n)
		callback(wrapper.haveReadSize, wrapper.expectedSize)
	}
	return
}

func (wrapper *requestBodyWrapperWithProgress) Close() error {
	return wrapper.body.Close()
}
