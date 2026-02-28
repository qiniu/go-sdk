//go:build unit
// +build unit

package region

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
)

func TestAllRegionsProvider(t *testing.T) {
	const accessKey = "fakeaccesskey"
	const secretKey = "fakesecretkey"
	var callCount uint64
	mux := http.NewServeMux()
	mux.HandleFunc("/regions", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Qiniu "+accessKey) {
			t.Fatalf("unexpected authorization")
		}
		w.Header().Add("X-ReqId", "fakereqid")
		if _, err := io.WriteString(w, mockUcRegionsResponseBody()); err != nil {
			t.Fatal(err)
		}
		atomic.AddUint64(&callCount, 1)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	cacheFile, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(cacheFile.Name())
	defer cacheFile.Close()

	provider, err := NewAllRegionsProvider(credentials.NewCredentials(accessKey, secretKey), Endpoints{Preferred: []string{server.URL}}, &AllRegionsProviderOptions{
		PersistentFilePath: cacheFile.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		regions, err := provider.GetRegions(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if regionsCount := len(regions); regionsCount != 2 {
			t.Fatalf("Unexpected regions count: %d", regionsCount)
		}
		if regionId := regions[0].RegionID; regionId != "z0" {
			t.Fatalf("Unexpected regionId: %s", regionId)
		}
		if preferredUp := regions[0].Up.Preferred; !reflect.DeepEqual(preferredUp, []string{"upload.qiniup.com", "up.qiniup.com"}) {
			t.Fatalf("Unexpected preferred up domains: %v", preferredUp)
		}
		if alternativeUp := regions[0].Up.Alternative; !reflect.DeepEqual(alternativeUp, []string{"upload.qbox.me", "up.qbox.me"}) {
			t.Fatalf("Unexpected alternative up domains: %v", alternativeUp)
		}
		if regionId := regions[1].RegionID; regionId != "z1" {
			t.Fatalf("Unexpected regionId: %s", regionId)
		}
		if preferredUp := regions[1].Up.Preferred; !reflect.DeepEqual(preferredUp, []string{"upload-z1.qiniup.com", "up-z1.qiniup.com"}) {
			t.Fatalf("Unexpected preferred up domains: %v", preferredUp)
		}
		if alternativeUp := regions[1].Up.Alternative; !reflect.DeepEqual(alternativeUp, []string{"upload-z1.qbox.me", "up-z1.qbox.me"}) {
			t.Fatalf("Unexpected alternative up domains: %v", alternativeUp)
		}
	}
	if cc := atomic.LoadUint64(&callCount); cc != 1 {
		t.Fatalf("Unexpected call count: %d", cc)
	}
}

func mockUcRegionsResponseBody() string {
	return `
	{
		"regions": [
			{
				"id": "z0",
				"ttl": 86400,
				"io": {
					"domains": ["iovip.qbox.me"]
				},
				"up": {
					"domains": ["upload.qiniup.com", "up.qiniup.com"],
					"old": ["upload.qbox.me", "up.qbox.me"]
				},
				"uc": {
					"domains": ["uc.qbox.me"]
				},
				"rs": {
					"domains": ["rs-z0.qbox.me"]
				},
				"rsf": {
					"domains": ["rsf-z0.qbox.me"]
				},
				"api": {
					"domains": ["api.qiniu.com"]
				}
			},
			{
				"id": "z1",
				"ttl": 86400,
				"io": {
					"domains": ["iovip-z1.qbox.me"]
				},
				"up": {
					"domains": ["upload-z1.qiniup.com", "up-z1.qiniup.com"],
					"old": ["upload-z1.qbox.me", "up-z1.qbox.me"]
				},
				"uc": {
					"domains": ["uc.qbox.me"]
				},
				"rs": {
					"domains": ["rs-z1.qbox.me"]
				},
				"rsf": {
					"domains": ["rsf-z1.qbox.me"]
				},
				"api": {
					"domains": ["api-z1.qiniu.com"]
				}
			}
		]
	}
	`
}
