//go:build unit
// +build unit

package client

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultDialer(t *testing.T) {
	var responseBody struct {
		Status string `json:"status"`
	}
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}))
	server := httptest.NewServer(mux)
	defer server.Close()

	port := server.Listener.Addr().(*net.TCPAddr).Port

	ctx := WithResolvedIPs(context.Background(), "www.qiniu.com", []net.IP{net.IPv4(127, 0, 0, 1)})
	err := DefaultClient.Call(ctx, &responseBody, http.MethodGet, fmt.Sprintf("http://www.qiniu.com:%d/", port), nil)
	if err != nil {
		t.Fatal(err)
	}
	if responseBody.Status != "ok" {
		t.Fatal("unexpected response")
	}
}
