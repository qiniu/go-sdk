package rpc

import (
	"strings"
	"errors"
	"io"
	"net/http"
	"net/url"
	"encoding/json"
	"github.com/qiniu/go-sdk/api"
)

// --------------------------------------------------------------------

type Client struct {
	*http.Client
}

// --------------------------------------------------------------------

func (r Client) PostWith(url1 string, bodyType string, body io.Reader, bodyLength int) (resp *http.Response, err error) {
	req, err := http.NewRequest("POST", url1, body)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", bodyType)
	req.ContentLength = int64(bodyLength)
	return r.Do(req)
}

func (r Client) PostWith64(url1 string, bodyType string, body io.Reader, bodyLength int64) (resp *http.Response, err error) {
	req, err := http.NewRequest("POST", url1, body)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", bodyType)
	req.ContentLength = bodyLength
	return r.Do(req)
}

func (r Client) PostWithForm(url1 string, data map[string][]string) (resp *http.Response, err error) {
	msg := url.Values(data).Encode()
	return r.PostWith(url1, "application/x-www-form-urlencoded", strings.NewReader(msg), len(msg))
}

// --------------------------------------------------------------------

type ErrorRet struct {
	Error string "error"
}

func callRet(ret interface{}, resp *http.Response) (code int, err error) {
	defer resp.Body.Close()
	code = resp.StatusCode
	if code/100 == 2 {
		if ret != nil && resp.ContentLength != 0 {
			err = json.NewDecoder(resp.Body).Decode(ret)
			if err != nil {
				code = api.UnexceptedResponse
			}
		}
	} else {
		if resp.ContentLength != 0 {
			if ct, ok := resp.Header["Content-Type"]; ok && ct[0] == "application/json" {
				var ret1 ErrorRet
				json.NewDecoder(resp.Body).Decode(&ret1)
				if ret1.Error != "" {
					err = errors.New(ret1.Error)
					return
				}
			}
		}
		err = api.Errno(code)
	}
	return
}

func (r Client) CallWithForm(ret interface{}, url1 string, param map[string][]string) (code int, err error) {
	resp, err := r.PostWithForm(url1, param)
	if err != nil {
		return api.InternalError, err
	}
	return callRet(ret, resp)
}

func (r Client) CallWith(ret interface{}, url1 string, bodyType string, body io.Reader, bodyLength int) (code int, err error) {

	resp, err := r.PostWith(url1, bodyType, body, bodyLength)
	if err != nil {
		return api.InternalError, err
	}
	return callRet(ret, resp)
}

func (r Client) CallWith64(ret interface{}, url1 string, bodyType string, body io.Reader, bodyLength int64) (code int, err error) {

	resp, err := r.PostWith64(url1, bodyType, body, bodyLength)
	if err != nil {
		return api.InternalError, err
	}
	return callRet(ret, resp)
}

func (r Client) Call(ret interface{}, url1 string) (code int, err error) {
	resp, err := r.PostWith(url1, "application/x-www-form-urlencoded", nil, 0)
	if err != nil {
		return api.InternalError, err
	}
	return callRet(ret, resp)
}

// --------------------------------------------------------------------

