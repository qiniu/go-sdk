//go:build unit
// +build unit

package objects_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/apis/stat_object"
	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/objects"
	"github.com/qiniu/go-sdk/v7/storagev2/region"
)

func TestObjectStat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if r.URL.RequestURI() == "/stat/"+base64.URLEncoding.EncodeToString([]byte("bucket1:testobject"))+"?needparts=true" {
				rw.Header().Set("Content-Type", "application/json")
				jsonData, err := json.Marshal(&stat_object.Response{
					Size:                        4 * 1024 * 1024,
					Hash:                        "testhash1",
					MimeType:                    "application/json",
					Type:                        0,
					PutTime:                     time.Now().UnixNano() / 100,
					RestoringStatus:             0,
					Status:                      0,
					TransitionToIaTime:          time.Now().Add(24 * time.Hour).Unix(),
					TransitionToArchiveIrTime:   time.Now().Add(2 * 24 * time.Hour).Unix(),
					TransitionToArchiveTime:     time.Now().Add(3 * 24 * time.Hour).Unix(),
					TransitionToDeepArchiveTime: time.Now().Add(4 * 24 * time.Hour).Unix(),
					ExpirationTime:              time.Now().Add(5 * 24 * time.Hour).Unix(),
					Metadata:                    map[string]string{"x-qn-meta-a": "b"},
					Parts:                       []int64{4 * 1024 * 1024},
				})
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
			Regions:     &region.Region{Rs: region.Endpoints{Preferred: []string{server.URL}}},
		},
	})
	bucket := objectsManager.Bucket("bucket1")
	if obj, err := bucket.Object("testobject").Stat().NeedParts(true).Call(context.Background()); err != nil {
		t.Fatal(err)
	} else {
		if obj.Size != 4*1024*1024 {
			t.Fatalf("unexpected fsize")
		}
		if obj.ETag != "testhash1" {
			t.Fatalf("unexpected etag")
		}
		if obj.MimeType != "application/json" {
			t.Fatalf("unexpected mimeType")
		}
		if obj.UploadedAt.Unix()-time.Now().Unix() >= 10 {
			t.Fatalf("unexpected putTime")
		}
		if obj.TransitionToIA.Unix()-time.Now().Add(24*time.Hour).Unix() >= 10 {
			t.Fatalf("unexpected transitionToIA")
		}
		if obj.TransitionToArchiveIR.Unix()-time.Now().Add(2*24*time.Hour).Unix() >= 10 {
			t.Fatalf("unexpected transitionToArchiveIR")
		}
		if obj.TransitionToArchive.Unix()-time.Now().Add(3*24*time.Hour).Unix() >= 10 {
			t.Fatalf("unexpected transitionToArchive")
		}
		if obj.TransitionToDeepArchive.Unix()-time.Now().Add(4*24*time.Hour).Unix() >= 10 {
			t.Fatalf("unexpected transitionToDeepArchive")
		}
		if obj.ExpireAt.Unix()-time.Now().Add(5*24*time.Hour).Unix() >= 10 {
			t.Fatalf("unexpected expiration")
		}
		if len(obj.Metadata) != 1 || obj.Metadata["x-qn-meta-a"] != "b" {
			t.Fatalf("unexpected metadata")
		}
		if len(obj.Parts) != 1 || obj.Parts[0] != 4*1024*1024 {
			t.Fatalf("unexpected parts")
		}
	}
}

func TestObjectMoveTo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			if r.URL.RequestURI() != "/move/"+base64.URLEncoding.EncodeToString([]byte("bucket1:testobject"))+"/"+base64.URLEncoding.EncodeToString([]byte("bucket2:testobject")) {
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
			Regions:     &region.Region{Rs: region.Endpoints{Preferred: []string{server.URL}}},
		},
	})
	bucket := objectsManager.Bucket("bucket1")
	if err := bucket.Object("testobject").MoveTo("bucket2", "testobject").Call(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestObjectCopyTo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			if r.URL.RequestURI() != "/copy/"+base64.URLEncoding.EncodeToString([]byte("bucket1:testobject"))+"/"+base64.URLEncoding.EncodeToString([]byte("bucket2:testobject")) {
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
			Regions:     &region.Region{Rs: region.Endpoints{Preferred: []string{server.URL}}},
		},
	})
	bucket := objectsManager.Bucket("bucket1")
	if err := bucket.Object("testobject").CopyTo("bucket2", "testobject").Call(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestObjectDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			if r.URL.RequestURI() != "/delete/"+base64.URLEncoding.EncodeToString([]byte("bucket1:testobject")) {
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
			Regions:     &region.Region{Rs: region.Endpoints{Preferred: []string{server.URL}}},
		},
	})
	bucket := objectsManager.Bucket("bucket1")
	if err := bucket.Object("testobject").Delete().Call(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestObjectRestore(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			if r.URL.RequestURI() != "/restoreAr/"+base64.URLEncoding.EncodeToString([]byte("bucket1:testobject"))+"/freezeAfterDays/7" {
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
			Regions:     &region.Region{Rs: region.Endpoints{Preferred: []string{server.URL}}},
		},
	})
	bucket := objectsManager.Bucket("bucket1")
	if err := bucket.Object("testobject").Restore(7).Call(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestObjectSetStorageClass(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			if r.URL.RequestURI() != "/chtype/"+base64.URLEncoding.EncodeToString([]byte("bucket1:testobject"))+"/type/4" {
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
			Regions:     &region.Region{Rs: region.Endpoints{Preferred: []string{server.URL}}},
		},
	})
	bucket := objectsManager.Bucket("bucket1")
	if err := bucket.Object("testobject").SetStorageClass(objects.ArchiveIRStorageClass).Call(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestObjectSetStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			if r.URL.RequestURI() != "/chstatus/"+base64.URLEncoding.EncodeToString([]byte("bucket1:testobject"))+"/status/1" {
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
			Regions:     &region.Region{Rs: region.Endpoints{Preferred: []string{server.URL}}},
		},
	})
	bucket := objectsManager.Bucket("bucket1")
	if err := bucket.Object("testobject").SetStatus(objects.DisabledStatus).Call(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestObjectSetMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			if r.URL.RequestURI() != "/chgm/"+base64.URLEncoding.EncodeToString([]byte("bucket1:testobject"))+
				"/mime/"+base64.URLEncoding.EncodeToString([]byte("application/json"))+
				"/cond/"+base64.URLEncoding.EncodeToString([]byte("fsize=1&hash=testhash"))+
				"/x-qn-meta-a/"+base64.URLEncoding.EncodeToString([]byte("b")) {
				t.Fatalf("unexpected path: %s", r.URL.RequestURI())
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
			Regions:     &region.Region{Rs: region.Endpoints{Preferred: []string{server.URL}}},
		},
	})
	bucket := objectsManager.Bucket("bucket1")
	if err := bucket.Object("testobject").SetMetadata("application/json").
		Metadata(map[string]string{"a": "b"}).
		Conditions(map[string]string{"fsize": "1", "hash": "testhash"}).
		Call(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestObjectSetLifeCycle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			if r.URL.RequestURI() != "/lifecycle/"+base64.URLEncoding.EncodeToString([]byte("bucket1:testobject"))+
				"/toIAAfterDays/1/toArchiveAfterDays/3/toDeepArchiveAfterDays/4/toArchiveIRAfterDays/2/deleteAfterDays/5" {
				t.Fatalf("unexpected path: %s", r.URL.RequestURI())
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
			Regions:     &region.Region{Rs: region.Endpoints{Preferred: []string{server.URL}}},
		},
	})
	bucket := objectsManager.Bucket("bucket1")
	if err := bucket.Object("testobject").
		SetLifeCycle().
		ToIAAfterDays(1).
		ToArchiveIRAfterDays(2).
		ToArchiveAfterDays(3).
		ToDeepArchiveAfterDays(4).
		DeleteAfterDays(5).
		Call(context.Background()); err != nil {
		t.Fatal(err)
	}
}
