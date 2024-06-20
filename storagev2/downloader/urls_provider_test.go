//go:build unit
// +build unit

package downloader_test

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync/atomic"
	"testing"

	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/downloader"
	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/region"
)

func TestStaticDomainBasedURLsProvider(t *testing.T) {
	generator := downloader.NewStaticDomainBasedURLsProvider([]string{
		"http://testa.com/",
		"https://b.testb.com/",
		"testc.com",
	})

	urls, err := generator.GetURLs(context.Background(), "/!@#$%^&*()?", &downloader.GenerateOptions{
		Command: "test1|test2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) != 3 {
		t.Fatalf("unexpected urls count")
	}
	if getURLProviderString(t, urls[0]) != "http://testa.com//%21@%23$%25%5E&%2A%28%29%3F?test1|test2" {
		t.Fatalf("unexpected generated url")
	}
	if getURLProviderString(t, urls[1]) != "https://b.testb.com//%21@%23$%25%5E&%2A%28%29%3F?test1|test2" {
		t.Fatalf("unexpected generated url")
	}
	if getURLProviderString(t, urls[2]) != "https://testc.com//%21@%23$%25%5E&%2A%28%29%3F?test1|test2" {
		t.Fatalf("unexpected generated url")
	}
}

func TestDefaultSrcURLsProvider(t *testing.T) {
	const accessKey = "fakeaccesskey"
	const secretKey = "fakesecretkey"
	const bucketName = "fakeBucketName"
	mux := http.NewServeMux()
	mux.HandleFunc("/v4/query", func(w http.ResponseWriter, r *http.Request) {
		if gotAk := r.URL.Query().Get("ak"); gotAk != accessKey {
			t.Fatalf("Unexpected ak: %s", gotAk)
		}
		if gotBucketName := r.URL.Query().Get("bucket"); gotBucketName != bucketName {
			t.Fatalf("Unexpected bucket: %s", gotBucketName)
		}
		if _, err := io.WriteString(w, `
{
	"hosts": [
		{
			"region": "z0",
			"ttl": 86400,
			"io_src": {
				"domains": ["fakebucket.cn-east-1.qiniucs.com"]
			}
		}
	]
}
		`); err != nil {
			t.Fatal(err)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	cacheFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(cacheFile.Name())
	defer cacheFile.Close()

	urlsProvider := downloader.NewDefaultSrcURLsProvider(
		accessKey,
		&downloader.DefaultSrcURLsProviderOptions{
			BucketRegionsQueryOptions: region.BucketRegionsQueryOptions{PersistentFilePath: cacheFile.Name()},
			BucketHosts:               region.Endpoints{Preferred: []string{server.URL}},
		},
	)
	urls, err := urlsProvider.GetURLs(context.Background(), "/!@#$%^&*()?", &downloader.GenerateOptions{
		BucketName: bucketName,
		Command:    "test1|test2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) != 1 {
		t.Fatalf("unexpected urls count")
	}
	if getURLProviderString(t, urls[0]) != "https://fakebucket.cn-east-1.qiniucs.com//%21@%23$%25%5E&%2A%28%29%3F?test1|test2" {
		t.Fatalf("unexpected generated url")
	}
}

func TestDomainsQueryURLsProvider(t *testing.T) {
	const accessKey = "fakeaccesskey"
	const secretKey = "fakesecretkey"
	const bucketName = "fakeBucketName"
	var callCount uint64
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/domains", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&callCount, 1)
		if r.URL.String() != "/v2/domains?tbl="+bucketName {
			t.Fatalf("unexpected request url")
		}
		bytes, err := json.Marshal([]string{"domain1.com", "domain2.com"})
		if err != nil {
			t.Fatal(err)
		}
		w.Write(bytes)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	cacheFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(cacheFile.Name())
	defer cacheFile.Close()

	generator, err := downloader.NewDomainsQueryURLsProvider(&downloader.DomainsQueryURLsProviderOptions{
		Options: http_client.Options{
			Regions:     &region.Region{Bucket: region.Endpoints{Preferred: []string{server.URL}}},
			Credentials: credentials.NewCredentials(accessKey, secretKey),
		},
		PersistentFilePath: cacheFile.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		urls, err := generator.GetURLs(context.Background(), "/!@#$%^&*()?", &downloader.GenerateOptions{
			BucketName: bucketName,
			Command:    "test1|test2",
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(urls) != 2 {
			t.Fatalf("unexpected urls")
		}
		if getURLProviderString(t, urls[0]) != "https://domain1.com//%21@%23$%25%5E&%2A%28%29%3F?test1|test2" {
			t.Fatalf("unexpected urls")
		}
		if getURLProviderString(t, urls[1]) != "https://domain2.com//%21@%23$%25%5E&%2A%28%29%3F?test1|test2" {
			t.Fatalf("unexpected urls")
		}
	}
	if atomic.LoadUint64(&callCount) != 1 {
		t.Fatalf("unexpected call count")
	}
}

func getURLProviderString(t *testing.T, urlProvider downloader.URLProvider) string {
	var u url.URL
	if err := urlProvider.GetURL(&u); err != nil {
		t.Fatal(err)
	}
	return u.String()
}
