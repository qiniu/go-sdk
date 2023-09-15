package clientv2

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	RequestMethodGet    = http.MethodGet
	RequestMethodPut    = http.MethodPut
	RequestMethodPost   = http.MethodPost
	RequestMethodHead   = http.MethodHead
	RequestMethodDelete = http.MethodDelete
)

type nopCloser struct {
	r io.ReadSeeker
}

func (nc nopCloser) Read(p []byte) (n int, err error) {
	return nc.r.Read(p)
}

func (nc nopCloser) Seek(offset int64, whence int) (int64, error) {
	return nc.r.Seek(offset, whence)
}

func (nc nopCloser) Close() error {
	return nil
}

type GetRequestBody func(options *RequestParams) io.ReadCloser

func GetJsonRequestBody(object interface{}) (GetRequestBody, error) {
	reqBody, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}
	return func(o *RequestParams) io.ReadCloser {
		o.Header.Add("Content-Type", "application/json")
		return nopCloser{r: bytes.NewReader(reqBody)}
	}, nil
}

func GetFormRequestBody(info map[string][]string) GetRequestBody {
	body := FormStringInfo(info)
	return func(o *RequestParams) io.ReadCloser {
		o.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		return nopCloser{r: strings.NewReader(body)}
	}
}

func FormStringInfo(info map[string][]string) string {
	if len(info) == 0 {
		return ""
	}
	return url.Values(info).Encode()
}

type RequestParams struct {
	Context context.Context
	Method  string
	Url     string
	Header  http.Header
	GetBody GetRequestBody
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
		o.GetBody = func(options *RequestParams) io.ReadCloser {
			return nil
		}
	}
}

func NewRequest(options RequestParams) (*http.Request, error) {
	options.init()

	body := options.GetBody(&options)
	req, err := http.NewRequest(options.Method, options.Url, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(options.Context)
	req.Header = options.Header
	if body != nil {
		req.GetBody = func() (io.ReadCloser, error) {
			return options.GetBody(&options), nil
		}
	}
	return req, nil
}
