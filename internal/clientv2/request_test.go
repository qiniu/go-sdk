//go:build unit
// +build unit

package clientv2

import (
	"net/http"
	"testing"
)

func TestGetJsonRequestBody(t *testing.T) {
	runTestCase := func(t *testing.T, getBody GetRequestBody) {
		params := RequestParams{Header: make(http.Header)}
		readCloser, err := getBody(&params)
		if err != nil {
			t.Fatal(err)
		}
		defer readCloser.Close()

		buf := make([]byte, 1024)
		n, err := readCloser.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		if string(buf[:n]) != `{"v":"value","v2":10}` {
			t.Fatal("invalid body")
		} else if params.Header.Get("Content-Type") != "application/json" {
			t.Fatal("invalid header")
		} else if params.Header.Get("Content-Length") != "21" {
			t.Fatal("invalid header")
		} else if err = readCloser.Close(); err != nil {
			t.Fatal(err)
		}
	}
	type S struct {
		V  string `json:"v"`
		V2 int    `json:"v2"`
	}

	getBody, err := GetJsonRequestBody(S{V: "value", V2: 10})
	if err != nil {
		t.Fatal(err)
	}
	runTestCase(t, getBody)
	runTestCase(t, getBody)
}

func TestGetFormRequestBody(t *testing.T) {
	runTestCase := func(t *testing.T, getBody GetRequestBody) {
		params := RequestParams{Header: make(http.Header)}
		readCloser, err := getBody(&params)
		if err != nil {
			t.Fatal(err)
		}
		defer readCloser.Close()

		buf := make([]byte, 1024)
		n, err := readCloser.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		if string(buf[:n]) != `v=value&v2=1&v2=2&v2=3` {
			t.Fatal("invalid body")
		} else if params.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Fatal("invalid header")
		} else if params.Header.Get("Content-Length") != "22" {
			t.Fatal("invalid header")
		} else if err = readCloser.Close(); err != nil {
			t.Fatal(err)
		}
	}

	getBody := GetFormRequestBody(map[string][]string{"v": {"value"}, "v2": {"1", "2", "3"}})
	runTestCase(t, getBody)
	runTestCase(t, getBody)
}
