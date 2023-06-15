package clientv2

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

const (
	RequestMethodGet    = "GET"
	RequestMethodPut    = "PUT"
	RequestMethodPost   = "POST"
	RequestMethodHead   = "HEAD"
	RequestMethodDelete = "DELETE"
)

type RequestBodyCreator func(options *RequestOptions) (io.Reader, error)

func JsonRequestBodyCreator(object interface{}) RequestBodyCreator {
	body := object
	return func(o *RequestOptions) (io.Reader, error) {
		reqBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		o.Header.Add("Content-Type", "application/json")
		return bytes.NewReader(reqBody), nil
	}
}

func FormRequestBodyCreator(info map[string][]string) RequestBodyCreator {
	body := FormString(info)
	return func(o *RequestOptions) (io.Reader, error) {
		o.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		return bytes.NewBufferString(body), nil
	}
}

func FormString(info map[string][]string) string {
	if len(info) == 0 {
		return ""
	}
	return url.Values(info).Encode()
}

type RequestOptions struct {
	Context     context.Context
	Method      string
	Url         string
	Header      http.Header
	BodyCreator RequestBodyCreator
}

func (o *RequestOptions) init() {
	if o.Context == nil {
		o.Context = context.Background()
	}

	if len(o.Method) == 0 {
		o.Method = RequestMethodGet
	}

	if o.Header == nil {
		o.Header = http.Header{}
	}

	if o.BodyCreator == nil {
		o.BodyCreator = func(options *RequestOptions) (io.Reader, error) {
			return nil, nil
		}
	}
}

func NewRequest(options RequestOptions) (*http.Request, error) {
	options.init()
	body, cErr := options.BodyCreator(&options)
	if cErr != nil {
		return nil, cErr
	}

	req, err := http.NewRequest(options.Method, options.Url, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(options.Context)
	req.Header = options.Header
	return req, nil
}
