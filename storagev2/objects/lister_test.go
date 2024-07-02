//go:build unit
// +build unit

package objects_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/apis/get_objects"
	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/objects"
	"github.com/qiniu/go-sdk/v7/storagev2/region"
)

func TestObjectLister(t *testing.T) {
	counted := 0
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if r.URL.Path == "/list" {
				rw.Header().Set("Content-Type", "application/json")
				var (
					jsonData []byte
					err      error
					query    = r.URL.Query()
				)
				if query.Get("bucket") != "bucket1" {
					t.Fatalf("unexpected bucket")
				}
				if query.Get("prefix") != "test/" {
					t.Fatalf("unexpected prefix")
				}
				if query.Get("delimiter") != "" {
					t.Fatalf("unexpected delimiter")
				}
				if query.Get("limit") != "" {
					t.Fatalf("unexpected limit")
				}
				switch counted {
				case 0:
					if query.Get("marker") != "" {
						t.Fatalf("unexpected marker")
					}
					jsonData, err = json.Marshal(&get_objects.Response{
						Marker: "testmarker1",
						Items: []get_objects.ListedObjectEntry{{
							Key:      "test/1",
							PutTime:  time.Now().UnixNano() / 100,
							Hash:     "testhash1",
							Size:     4 * 1024 * 1024,
							MimeType: "application/json",
							Parts:    []int64{4 * 1024 * 1024},
						}, {
							Key:      "test/2",
							PutTime:  time.Now().UnixNano() / 100,
							Hash:     "testhash2",
							Size:     4 * 1024 * 1024,
							MimeType: "application/json",
							Parts:    []int64{4 * 1024 * 1024},
						}, {
							Key:      "test/3",
							PutTime:  time.Now().UnixNano() / 100,
							Hash:     "testhash3",
							Size:     4 * 1024 * 1024,
							MimeType: "application/json",
							Parts:    []int64{4 * 1024 * 1024},
						}},
					})
				case 1:
					if query.Get("marker") != "testmarker1" {
						t.Fatalf("unexpected marker")
					}
					jsonData, err = json.Marshal(&get_objects.Response{
						Items: []get_objects.ListedObjectEntry{{
							Key:      "test/4",
							PutTime:  time.Now().UnixNano() / 100,
							Hash:     "testhash4",
							Size:     4 * 1024 * 1024,
							MimeType: "application/json",
							Parts:    []int64{4 * 1024 * 1024},
						}, {
							Key:      "test/5",
							PutTime:  time.Now().UnixNano() / 100,
							Hash:     "testhash5",
							Size:     4 * 1024 * 1024,
							MimeType: "application/json",
							Parts:    []int64{4 * 1024 * 1024},
						}},
					})
				default:
					t.Fatalf("unexpected request")
				}
				counted += 1
				if err != nil {
					t.Fatal(err)
				}
				rw.Write(jsonData)
			} else {
				t.Fatalf("unexpected path")
			}
			if !strings.HasPrefix(r.Header.Get("Authorization"), "Qiniu testak:") {
				t.Fatalf("unexpected authorization")
			}
		default:
			t.Fatalf("unexpected method: %s", r.Method)
		}
	}))
	defer server.Close()

	objectsManager := objects.NewObjectsManager(&objects.ObjectsManagerOptions{
		Options: http_client.Options{
			Credentials: credentials.NewCredentials("testak", "testsk"),
			Regions:     &region.Region{Rsf: region.Endpoints{Preferred: []string{server.URL}}},
		},
	})
	bucket := objectsManager.Bucket("bucket1")
	lister := bucket.List(context.Background(), &objects.ListObjectsOptions{
		Prefix:    "test/",
		NeedParts: true,
	})
	defer lister.Close()

	var objectDetails objects.ObjectDetails
	if !lister.Next(&objectDetails) {
		t.Fatalf("unexpected eof: %s", lister.Error())
	}
	if objectDetails.Name != "test/1" {
		t.Fatalf("unexpected object name")
	}
	if objectDetails.Size != 4*1024*1024 {
		t.Fatalf("unexpected size")
	}
	if objectDetails.UploadedAt.Unix()-time.Now().Unix() >= 10 {
		t.Fatalf("unexpected putTime")
	}

	if !lister.Next(&objectDetails) {
		t.Fatalf("unexpected eof: %s", lister.Error())
	}
	if objectDetails.Name != "test/2" {
		t.Fatalf("unexpected object name")
	}
	if objectDetails.Size != 4*1024*1024 {
		t.Fatalf("unexpected size")
	}
	if objectDetails.UploadedAt.Unix()-time.Now().Unix() >= 10 {
		t.Fatalf("unexpected putTime")
	}

	if !lister.Next(&objectDetails) {
		t.Fatalf("unexpected eof: %s", lister.Error())
	}
	if objectDetails.Name != "test/3" {
		t.Fatalf("unexpected object name")
	}
	if objectDetails.Size != 4*1024*1024 {
		t.Fatalf("unexpected size")
	}
	if objectDetails.UploadedAt.Unix()-time.Now().Unix() >= 10 {
		t.Fatalf("unexpected putTime")
	}

	if !lister.Next(&objectDetails) {
		t.Fatalf("unexpected eof: %s", lister.Error())
	}
	if objectDetails.Name != "test/4" {
		t.Fatalf("unexpected object name")
	}
	if objectDetails.Size != 4*1024*1024 {
		t.Fatalf("unexpected size")
	}
	if objectDetails.UploadedAt.Unix()-time.Now().Unix() >= 10 {
		t.Fatalf("unexpected putTime")
	}

	if !lister.Next(&objectDetails) {
		t.Fatalf("unexpected eof: %s", lister.Error())
	}
	if objectDetails.Name != "test/5" {
		t.Fatalf("unexpected object name")
	}
	if objectDetails.Size != 4*1024*1024 {
		t.Fatalf("unexpected size")
	}
	if objectDetails.UploadedAt.Unix()-time.Now().Unix() >= 10 {
		t.Fatalf("unexpected putTime")
	}

	if err := lister.Error(); err != nil {
		t.Fatal(err)
	}
}
