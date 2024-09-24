//go:build unit
// +build unit

package region

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sync/atomic"
	"testing"
)

func TestBucketRegionsQuery(t *testing.T) {
	const accessKey = "fakeaccesskey"
	const bucketName = "fakeBucketName"
	var callCount uint64
	mux := http.NewServeMux()
	mux.HandleFunc("/v4/query", func(w http.ResponseWriter, r *http.Request) {
		if gotAk := r.URL.Query().Get("ak"); gotAk != accessKey {
			t.Fatalf("Unexpected ak: %s", gotAk)
		}
		if gotBucketName := r.URL.Query().Get("bucket"); gotBucketName != bucketName {
			t.Fatalf("Unexpected bucket: %s", gotBucketName)
		}
		w.Header().Add("X-ReqId", "fakereqid")
		if _, err := io.WriteString(w, mockUcQueryResponseBody()); err != nil {
			t.Fatal(err)
		}
		atomic.AddUint64(&callCount, 1)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	cacheFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(cacheFile.Name())
	defer cacheFile.Close()

	query, err := NewBucketRegionsQuery(Endpoints{Preferred: []string{server.URL}}, &BucketRegionsQueryOptions{
		PersistentFilePath: cacheFile.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		regions, err := query.Query(accessKey, bucketName).GetRegions(context.Background())
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

func mockUcQueryResponseBody() string {
	return `
	{
		"hosts": [
			{
				"region": "z0",
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
				"region": "z1",
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
