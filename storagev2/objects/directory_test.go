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

func TestDirectoryListEntriesWithoutRecurse(t *testing.T) {
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
				if query.Get("prefix") != "" {
					t.Fatalf("unexpected prefix")
				}
				if query.Get("delimiter") != "/" {
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
						Marker:         "testmarker1",
						CommonPrefixes: []string{"test1/", "test2/"},
						Items: []get_objects.ListedObjectEntry{{
							Key:      "file1",
							PutTime:  time.Now().UnixNano() / 100,
							Hash:     "testhash1",
							Size:     4 * 1024 * 1024,
							MimeType: "application/json",
						}},
					})
				case 1:
					if query.Get("marker") != "testmarker1" {
						t.Fatalf("unexpected marker")
					}
					jsonData, err = json.Marshal(&get_objects.Response{
						CommonPrefixes: []string{"test3/", "test4/"},
						Items: []get_objects.ListedObjectEntry{{
							Key:      "file2",
							PutTime:  time.Now().UnixNano() / 100,
							Hash:     "testhash1",
							Size:     4 * 1024 * 1024,
							MimeType: "application/json",
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
	directory := objectsManager.Bucket("bucket1").Directory("", "")
	listed := make(map[string]*objects.ObjectDetails)
	err := directory.ListEntries(context.Background(), nil, func(de *objects.Entry) error {
		if de.DirectoryName != "" {
			listed[de.DirectoryName] = nil
		} else {
			listed[de.Object.Name] = de.Object
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 6 {
		t.Fatalf("unexpected listed length")
	}
	if obj, ok := listed["test1/"]; !ok || obj != nil {
		t.Fatalf("unexpected directory list1")
	}
	if obj, ok := listed["test2/"]; !ok || obj != nil {
		t.Fatalf("unexpected directory list2")
	}
	if obj, ok := listed["test3/"]; !ok || obj != nil {
		t.Fatalf("unexpected directory list3")
	}
	if obj, ok := listed["test4/"]; !ok || obj != nil {
		t.Fatalf("unexpected directory list4")
	}
	if obj, ok := listed["file1"]; !ok || obj == nil {
		t.Fatalf("unexpected object file1")
	}
	if obj, ok := listed["file2"]; !ok || obj == nil {
		t.Fatalf("unexpected object file2")
	}
}

func TestDirectoryListEntriesWithRecurse(t *testing.T) {
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
				if query.Get("delimiter") != "/" {
					t.Fatalf("unexpected delimiter")
				}
				if query.Get("limit") != "" {
					t.Fatalf("unexpected limit")
				}
				switch counted {
				case 0:
					if query.Get("prefix") != "" {
						t.Fatalf("unexpected prefix")
					}
					if query.Get("marker") != "" {
						t.Fatalf("unexpected marker")
					}
					jsonData, err = json.Marshal(&get_objects.Response{
						Marker:         "testmarker1",
						CommonPrefixes: []string{"test1/"},
						Items: []get_objects.ListedObjectEntry{{
							Key:      "file1",
							PutTime:  time.Now().UnixNano() / 100,
							Hash:     "testhash1",
							Size:     4 * 1024 * 1024,
							MimeType: "application/json",
						}},
					})
				case 1:
					if query.Get("prefix") != "" {
						t.Fatalf("unexpected prefix")
					}
					if query.Get("marker") != "testmarker1" {
						t.Fatalf("unexpected marker")
					}
					jsonData, err = json.Marshal(&get_objects.Response{
						CommonPrefixes: []string{"test2/"},
						Items: []get_objects.ListedObjectEntry{{
							Key:      "file2",
							PutTime:  time.Now().UnixNano() / 100,
							Hash:     "testhash1",
							Size:     4 * 1024 * 1024,
							MimeType: "application/json",
						}},
					})
				case 2:
					if query.Get("prefix") != "test1/" {
						t.Fatalf("unexpected prefix")
					}
					if query.Get("marker") != "" {
						t.Fatalf("unexpected marker")
					}
					jsonData, err = json.Marshal(&get_objects.Response{
						Items: []get_objects.ListedObjectEntry{{
							Key:      "test1/file1",
							PutTime:  time.Now().UnixNano() / 100,
							Hash:     "testhash1",
							Size:     4 * 1024 * 1024,
							MimeType: "application/json",
						}},
					})
				case 3:
					if query.Get("prefix") != "test2/" {
						t.Fatalf("unexpected prefix")
					}
					if query.Get("marker") != "" {
						t.Fatalf("unexpected marker")
					}
					jsonData, err = json.Marshal(&get_objects.Response{
						Items: []get_objects.ListedObjectEntry{{
							Key:      "test2/file2",
							PutTime:  time.Now().UnixNano() / 100,
							Hash:     "testhash1",
							Size:     4 * 1024 * 1024,
							MimeType: "application/json",
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
	directory := objectsManager.Bucket("bucket1").Directory("", "")
	listed := make(map[string]*objects.ObjectDetails)
	err := directory.ListEntries(context.Background(), &objects.ListEntriesOptions{
		Recursive: true,
	}, func(de *objects.Entry) error {
		if de.DirectoryName != "" {
			listed[de.DirectoryName] = nil
		} else {
			listed[de.Object.Name] = de.Object
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 6 {
		t.Fatalf("unexpected listed length")
	}
	if obj, ok := listed["test1/"]; !ok || obj != nil {
		t.Fatalf("unexpected directory list1")
	}
	if obj, ok := listed["test2/"]; !ok || obj != nil {
		t.Fatalf("unexpected directory list2")
	}
	if obj, ok := listed["file1"]; !ok || obj == nil {
		t.Fatalf("unexpected object file1")
	}
	if obj, ok := listed["file2"]; !ok || obj == nil {
		t.Fatalf("unexpected object file2")
	}
	if obj, ok := listed["test1/file1"]; !ok || obj == nil {
		t.Fatalf("unexpected object file1")
	}
	if obj, ok := listed["test2/file2"]; !ok || obj == nil {
		t.Fatalf("unexpected object file2")
	}
}

func TestDirectoryListEntriesWithSkipDir(t *testing.T) {
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
				if query.Get("delimiter") != "/" {
					t.Fatalf("unexpected delimiter")
				}
				if query.Get("limit") != "" {
					t.Fatalf("unexpected limit")
				}
				switch counted {
				case 0:
					if query.Get("prefix") != "" {
						t.Fatalf("unexpected prefix")
					}
					if query.Get("marker") != "" {
						t.Fatalf("unexpected marker")
					}
					jsonData, err = json.Marshal(&get_objects.Response{
						Marker:         "testmarker1",
						CommonPrefixes: []string{"test1/"},
						Items: []get_objects.ListedObjectEntry{{
							Key:      "file1",
							PutTime:  time.Now().UnixNano() / 100,
							Hash:     "testhash1",
							Size:     4 * 1024 * 1024,
							MimeType: "application/json",
						}},
					})
				case 1:
					if query.Get("prefix") != "" {
						t.Fatalf("unexpected prefix")
					}
					if query.Get("marker") != "testmarker1" {
						t.Fatalf("unexpected marker")
					}
					jsonData, err = json.Marshal(&get_objects.Response{
						CommonPrefixes: []string{"test2/"},
						Items: []get_objects.ListedObjectEntry{{
							Key:      "file2",
							PutTime:  time.Now().UnixNano() / 100,
							Hash:     "testhash1",
							Size:     4 * 1024 * 1024,
							MimeType: "application/json",
						}},
					})
				case 2:
					if query.Get("prefix") != "test1/" {
						t.Fatalf("unexpected prefix")
					}
					if query.Get("marker") != "" {
						t.Fatalf("unexpected marker")
					}
					jsonData, err = json.Marshal(&get_objects.Response{
						Items: []get_objects.ListedObjectEntry{{
							Key:      "test1/file1",
							PutTime:  time.Now().UnixNano() / 100,
							Hash:     "testhash1",
							Size:     4 * 1024 * 1024,
							MimeType: "application/json",
						}, {
							Key:      "test1/file2",
							PutTime:  time.Now().UnixNano() / 100,
							Hash:     "testhash1",
							Size:     4 * 1024 * 1024,
							MimeType: "application/json",
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
	directory := objectsManager.Bucket("bucket1").Directory("", "")
	listed := make(map[string]*objects.ObjectDetails)
	err := directory.ListEntries(context.Background(), &objects.ListEntriesOptions{
		Recursive: true,
	}, func(de *objects.Entry) error {
		if de.DirectoryName != "" {
			listed[de.DirectoryName] = nil
			if de.DirectoryName == "test2/" {
				return objects.SkipDir
			}
		} else {
			listed[de.Object.Name] = de.Object
			if de.Object.Name == "test1/file1" {
				return objects.SkipDir
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 5 {
		t.Fatalf("unexpected listed length")
	}
	if obj, ok := listed["test1/"]; !ok || obj != nil {
		t.Fatalf("unexpected directory list1")
	}
	if obj, ok := listed["test2/"]; !ok || obj != nil {
		t.Fatalf("unexpected directory list2")
	}
	if obj, ok := listed["file1"]; !ok || obj == nil {
		t.Fatalf("unexpected object file1")
	}
	if obj, ok := listed["file2"]; !ok || obj == nil {
		t.Fatalf("unexpected object file2")
	}
	if obj, ok := listed["test1/file1"]; !ok || obj == nil {
		t.Fatalf("unexpected object file1")
	}
}
