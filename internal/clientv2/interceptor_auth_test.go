//go:build unit
// +build unit

package clientv2

import (
	"net/http"
	"strings"
	"testing"

	"github.com/qiniu/go-sdk/v7/auth"
	clientV1 "github.com/qiniu/go-sdk/v7/client"
)

func TestAuthInterceptor(t *testing.T) {
	clientV1.DebugMode = true
	defer func() {
		clientV1.DebugMode = false
	}()

	interceptor := NewAuthInterceptor(AuthConfig{
		Credentials: auth.New("ak", "sk"),
		TokenType:   auth.TokenQiniu,
		BeforeSign: func(req *http.Request) {
			if authorization := req.Header.Get("Authorization"); authorization != "" {
				t.Fatal("Authorization header should be empty")
			}
		},
		AfterSign: func(req *http.Request) {
			if authorization := req.Header.Get("Authorization"); authorization == "" {
				t.Fatal("Authorization header should not be empty")
			} else if !strings.HasPrefix(authorization, "Qiniu ak:") {
				t.Fatal("Unexpected Authorization header")
			}
		},
	})
	c := NewClient(&testClient{statusCode: http.StatusOK}, interceptor)
	resp, err := Do(c, RequestParams{
		Context: nil,
		Method:  RequestMethodGet,
		Url:     "https://test.qiniu.com/path/123",
		Header:  nil,
		GetBody: nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatal("status code not 200")
	}
}
