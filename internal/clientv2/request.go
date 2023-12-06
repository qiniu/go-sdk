package clientv2

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
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
		return internal_io.NewReadSeekableNopCloser(bytes.NewReader(reqBody)), nil
	}, nil
}

func GetFormRequestBody(info map[string][]string) GetRequestBody {
	body := formStringInfo(info)
	return func(o *RequestParams) (io.ReadCloser, error) {
		o.Header.Set("Content-Type", conf.CONTENT_TYPE_FORM)
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
	Context        context.Context
	Method         string
	Url            string
	Header         http.Header
	GetBody        GetRequestBody
	BufferResponse bool
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
	options.init()

	body, err := options.GetBody(&options)
	if err != nil {
		return nil, err
	}
	req, err = http.NewRequest(options.Method, options.Url, body)
	if err != nil {
		return
	}
	if options.Context != nil {
		req = req.WithContext(options.Context)
	}
	if options.BufferResponse {
		req = req.WithContext(context.WithValue(options.Context, contextKeyBufferResponse{}, struct{}{}))
	}
	req.Header = options.Header
	if options.GetBody != nil && body != nil && body != http.NoBody {
		req.GetBody = func() (io.ReadCloser, error) {
			return options.GetBody(&options)
		}
	}
	return
}
