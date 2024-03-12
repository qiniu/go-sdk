//go:build unit
// +build unit

package http_client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	clientv1 "github.com/qiniu/go-sdk/v7/client"
	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/region"
)

func TestHttpClient(t *testing.T) {
	type Req struct {
		id  int
		url *url.URL
	}
	var reqs = make([]Req, 0, 3)
	mux_1 := http.NewServeMux()
	mux_1.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		reqs = append(reqs, Req{id: 1, url: r.URL})
		if auth := r.Header.Get("Authorization"); !strings.HasPrefix(auth, "Qiniu TestAk:") {
			t.Fatalf("Unexpected authorization: %s", auth)
		}
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "test error")
	})
	server_1 := httptest.NewServer(mux_1)
	defer server_1.Close()

	mux_2 := http.NewServeMux()
	mux_2.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		reqs = append(reqs, Req{id: 2, url: r.URL})
		if auth := r.Header.Get("Authorization"); !strings.HasPrefix(auth, "Qiniu TestAk:") {
			t.Fatalf("Unexpected authorization: %s", auth)
		}
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "test error")
	})
	server_2 := httptest.NewServer(mux_2)
	defer server_2.Close()

	mux_3 := http.NewServeMux()
	mux_3.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		reqs = append(reqs, Req{id: 3, url: r.URL})
		if auth := r.Header.Get("Authorization"); !strings.HasPrefix(auth, "Qiniu TestAk:") {
			t.Fatalf("Unexpected authorization: %s", auth)
		}
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "test error")
	})
	server_3 := httptest.NewServer(mux_3)
	defer server_3.Close()

	httpClient := NewClient(&Options{
		Regions: &region.Region{
			Api: region.Endpoints{
				Preferred:   []string{server_1.URL, server_2.URL},
				Alternative: []string{server_3.URL},
			},
		},
	})
	_, err := httpClient.Do(context.Background(), &Request{
		ServiceNames: []region.ServiceName{region.ServiceApi},
		Method:       http.MethodGet,
		Path:         "/test",
		RawQuery:     "fakeRawQuery",
		Query: url.Values{
			"x-query-1": {"x-value-1"},
			"x-query-2": {"x-value-2"},
		},
		Header: http.Header{
			"x-qiniu-1": {"x-value-1"},
			"x-qiniu-2": {"x-value-2"},
		},
		Credentials: credentials.NewCredentials("TestAk", "TestSk"),
	})
	if err == nil {
		t.Fatalf("Expected error")
	}
	if clientErr, ok := err.(*clientv1.ErrorInfo); ok {
		if clientErr.Code != http.StatusInternalServerError {
			t.Fatalf("Unexpected status code: %d", clientErr.Code)
		}
	}
	if len(reqs) != 3 {
		t.Fatalf("Unexpected reqs: %#v", reqs)
	}
	for i, req := range reqs {
		if i+1 != req.id || req.url.String() != "/test?fakeRawQuery&x-query-1=x-value-1&x-query-2=x-value-2" {
			t.Fatalf("Unexpected req: %#v", req)
		}
	}
}

func TestHttpClientJson(t *testing.T) {
	mux_1 := http.NewServeMux()
	mux_1.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); !strings.HasPrefix(auth, "Qiniu TestAk:") {
			t.Fatalf("Unexpected authorization: %s", auth)
		}
		io.WriteString(w, "{\"Test\":\"AccessKey\"}")
	})
	server_1 := httptest.NewServer(mux_1)
	defer server_1.Close()

	httpClient := NewClient(&Options{
		Regions: &region.Region{
			Api: region.Endpoints{
				Preferred: []string{server_1.URL},
			},
		},
	})

	var body struct {
		Test string `json:"Test"`
	}

	err := httpClient.DoAndAcceptJSON(context.Background(), &Request{
		ServiceNames: []region.ServiceName{region.ServiceApi},
		Method:       http.MethodGet,
		Path:         "/test",
		RawQuery:     "fakeRawQuery",
		Query: url.Values{
			"x-query-1": {"x-value-1"},
			"x-query-2": {"x-value-2"},
		},
		Header: http.Header{
			"x-qiniu-1": {"x-value-1"},
			"x-qiniu-2": {"x-value-2"},
		},
		Credentials: credentials.NewCredentials("TestAk", "TestSk"),
	}, &body)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if body.Test != "AccessKey" {
		t.Fatalf("Unexpected body: %#v", body)
	}
}
