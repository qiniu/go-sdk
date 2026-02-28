//go:build unit
// +build unit

package uploader

import (
	"bytes"
	"context"
	"crypto/md5"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/qiniu/go-sdk/v7/client"
	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/region"
)

func TestFormUploader(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "form-uploader-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	hasher := md5.New()
	if _, err = io.CopyN(tmpFile, io.TeeReader(r, hasher), 1024*1024); err != nil {
		t.Fatal(err)
	}
	if _, err = tmpFile.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	expectedMd5 := hasher.Sum(nil)

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if err := r.ParseMultipartForm(2 * 1024 * 1024); err != nil {
			t.Fatal(err)
		}
		if values := r.MultipartForm.Value["key"]; len(values) != 1 || values[0] != "testkey" {
			t.Fatalf("unexpected key")
		}
		if values := r.MultipartForm.Value["token"]; len(values) != 1 || !strings.HasPrefix(values[0], "testak:") {
			t.Fatalf("unexpected token")
		}
		if values := r.MultipartForm.Value["x-qn-meta-a"]; len(values) != 1 || values[0] != "b" {
			t.Fatalf("unexpected x-qn-meta-a")
		}
		if values := r.MultipartForm.Value["x-qn-meta-c"]; len(values) != 1 || values[0] != "d" {
			t.Fatalf("unexpected x-qn-meta-c")
		}
		if values := r.MultipartForm.Value["x:a"]; len(values) != 1 || values[0] != "b" {
			t.Fatalf("unexpected x:a")
		}
		if values := r.MultipartForm.Value["x:c"]; len(values) != 1 || values[0] != "d" {
			t.Fatalf("unexpected x:c")
		}
		if files := r.MultipartForm.File["file"]; len(files) != 1 || files[0].Filename != "testfilename" || files[0].Size != 1024*1024 {
			t.Fatalf("unexpected file")
		} else if contentType := files[0].Header.Get("Content-Type"); contentType != "application/json" {
			t.Fatalf("unexpected file content-type")
		} else if file, err := files[0].Open(); err != nil {
			t.Fatal(err)
		} else {
			defer file.Close()
			hasher := md5.New()
			if _, err = io.Copy(hasher, file); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(hasher.Sum(nil), expectedMd5) {
				t.Fatalf("unexpected file content")
			}
		}
		w.Header().Add("x-reqid", "fakereqid")
		w.Write([]byte(`{"ok":true}`))
	})
	server := httptest.NewServer(serveMux)
	defer server.Close()

	formUploader := NewFormUploader(&FormUploaderOptions{
		Options: http_client.Options{
			Regions:     &region.Region{Up: region.Endpoints{Preferred: []string{server.URL}}},
			Credentials: credentials.NewCredentials("testak", "testsk"),
		},
	})
	var (
		returnValue struct {
			Ok bool `json:"ok"`
		}
		key          = "testkey"
		lastUploaded uint64
	)
	if err = formUploader.UploadFile(context.Background(), tmpFile.Name(), &ObjectOptions{
		BucketName:  "testbucket",
		ObjectName:  &key,
		FileName:    "testfilename",
		ContentType: "application/json",
		Metadata:    map[string]string{"a": "b", "c": "d"},
		CustomVars:  map[string]string{"a": "b", "c": "d"},
		OnUploadingProgress: func(progress *UploadingProgress) {
			if progress.TotalSize != 1024*1024 {
				t.Fatalf("unexpected file size")
			} else if progress.Uploaded > progress.TotalSize {
				t.Fatalf("unexpected uploaded")
			} else if lu := atomic.SwapUint64(&lastUploaded, progress.Uploaded); lu > progress.Uploaded || lu > progress.TotalSize {
				t.Fatalf("unexpected uploaded")
			}
		},
	}, &returnValue); err != nil {
		t.Fatal(err)
	}
	if !returnValue.Ok {
		t.Fatalf("unexpected response value")
	}
}

func TestFormUploaderRetry(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "form-uploader-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	hasher := md5.New()
	if _, err = io.CopyN(tmpFile, io.TeeReader(r, hasher), 1024*1024); err != nil {
		t.Fatal(err)
	}
	if _, err = tmpFile.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	expectedMd5 := hasher.Sum(nil)

	var handlerCalled_1, handlerCalled_2, handlerCalled_3 uint64

	serveMux_1 := http.NewServeMux()
	serveMux_1.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&handlerCalled_1, 1)
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if err := r.ParseMultipartForm(2 * 1024 * 1024); err != nil {
			t.Fatal(err)
		}
		if values := r.MultipartForm.Value["key"]; len(values) != 1 || values[0] != "testkey" {
			t.Fatalf("unexpected key")
		}
		if values := r.MultipartForm.Value["token"]; len(values) != 1 || !strings.HasPrefix(values[0], "testak:") {
			t.Fatalf("unexpected token")
		}
		if values := r.MultipartForm.Value["x-qn-meta-a"]; len(values) != 1 || values[0] != "b" {
			t.Fatalf("unexpected x-qn-meta-a")
		}
		if values := r.MultipartForm.Value["x-qn-meta-c"]; len(values) != 1 || values[0] != "d" {
			t.Fatalf("unexpected x-qn-meta-c")
		}
		if values := r.MultipartForm.Value["x:a"]; len(values) != 1 || values[0] != "b" {
			t.Fatalf("unexpected x:a")
		}
		if values := r.MultipartForm.Value["x:c"]; len(values) != 1 || values[0] != "d" {
			t.Fatalf("unexpected x:c")
		}
		if files := r.MultipartForm.File["file"]; len(files) != 1 || files[0].Filename != "testfilename" || files[0].Size != 1024*1024 {
			t.Fatalf("unexpected file")
		} else if contentType := files[0].Header.Get("Content-Type"); contentType != "application/json" {
			t.Fatalf("unexpected file content-type")
		} else if file, err := files[0].Open(); err != nil {
			t.Fatal(err)
		} else {
			defer file.Close()
			hasher := md5.New()
			if _, err = io.Copy(hasher, file); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(hasher.Sum(nil), expectedMd5) {
				t.Fatalf("unexpected file content")
			}
		}
		w.Header().Add("x-reqid", "fakereqid")
		w.Write([]byte(`{"ok":true}`))
	})
	server_1 := httptest.NewServer(serveMux_1)
	defer server_1.Close()

	serveMux_2 := http.NewServeMux()
	serveMux_2.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&handlerCalled_2, 1)
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		w.Header().Add("x-reqid", "fakereqid")
		w.WriteHeader(599)
	})
	server_2 := httptest.NewServer(serveMux_2)
	defer server_2.Close()

	serveMux_3 := http.NewServeMux()
	serveMux_3.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&handlerCalled_3, 1)
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		w.Header().Add("x-reqid", "fakereqid")
		w.WriteHeader(504)
	})
	server_3 := httptest.NewServer(serveMux_3)
	defer server_3.Close()

	handlerCalled_4 := uint64(0)
	serveMux_4 := http.NewServeMux()
	serveMux_4.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&handlerCalled_4, 1)
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		w.Header().Add("x-reqid", "fakereqid")
		w.WriteHeader(612)
	})
	server_4 := httptest.NewServer(serveMux_4)
	defer server_4.Close()

	var (
		returnValue struct {
			Ok bool `json:"ok"`
		}
		key = "testkey"
	)

	formUploader := NewFormUploader(&FormUploaderOptions{
		Options: http_client.Options{
			Regions: regions{[]*region.Region{
				{Up: region.Endpoints{Preferred: []string{server_3.URL}}},
				{Up: region.Endpoints{Preferred: []string{server_2.URL}}},
				{Up: region.Endpoints{Preferred: []string{server_1.URL}}},
			}},
			Credentials: credentials.NewCredentials("testak", "testsk"),
		},
	})
	if err = formUploader.UploadFile(context.Background(), tmpFile.Name(), &ObjectOptions{
		BucketName:  "testbucket",
		ObjectName:  &key,
		FileName:    "testfilename",
		ContentType: "application/json",
		Metadata:    map[string]string{"a": "b", "c": "d"},
		CustomVars:  map[string]string{"a": "b", "c": "d"},
	}, &returnValue); err != nil {
		t.Fatal(err)
	}
	if !returnValue.Ok {
		t.Fatalf("unexpected response value")
	}
	if count := atomic.LoadUint64(&handlerCalled_1); count != 1 {
		t.Fatalf("unexpected handler call count: %d", count)
	}
	if count := atomic.LoadUint64(&handlerCalled_2); count != 4 {
		t.Fatalf("unexpected handler call count: %d", count)
	}
	if count := atomic.LoadUint64(&handlerCalled_3); count != 4 {
		t.Fatalf("unexpected handler call count: %d", count)
	}
	atomic.StoreUint64(&handlerCalled_1, 0)
	atomic.StoreUint64(&handlerCalled_2, 0)
	atomic.StoreUint64(&handlerCalled_3, 0)

	formUploader = NewFormUploader(&FormUploaderOptions{
		Options: http_client.Options{
			Regions: regions{[]*region.Region{
				{Up: region.Endpoints{Preferred: []string{server_3.URL}}},
				{Up: region.Endpoints{Preferred: []string{server_2.URL}}},
				{Up: region.Endpoints{Preferred: []string{server_4.URL}}},
			}},
			Credentials: credentials.NewCredentials("testak", "testsk"),
		},
	})
	if err = formUploader.UploadFile(context.Background(), tmpFile.Name(), &ObjectOptions{
		RegionsProvider: nil,
		BucketName:      "testbucket",
		ObjectName:      &key,
		FileName:        "testfilename",
		ContentType:     "application/json",
		Metadata:        map[string]string{"a": "b", "c": "d"},
		CustomVars:      map[string]string{"a": "b", "c": "d"},
	}, &returnValue); err != nil {
		if errInfo, ok := err.(*client.ErrorInfo); !ok || errInfo.Code != 612 {
			t.Fatal(err)
		}
	}
	if count := atomic.LoadUint64(&handlerCalled_4); count != 1 {
		t.Fatalf("unexpected handler call count: %d", count)
	}
	if count := atomic.LoadUint64(&handlerCalled_2); count != 4 {
		t.Fatalf("unexpected handler call count: %d", count)
	}
	if count := atomic.LoadUint64(&handlerCalled_3); count != 4 {
		t.Fatalf("unexpected handler call count: %d", count)
	}
}

func TestFormUploaderAccelaratedUploading(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "form-uploader-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	hasher := md5.New()
	if _, err = io.CopyN(tmpFile, io.TeeReader(r, hasher), 1024*1024); err != nil {
		t.Fatal(err)
	}
	if _, err = tmpFile.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	expectedMd5 := hasher.Sum(nil)

	var handlerCalled_1, handlerCalled_2 uint64

	serveMux_1 := http.NewServeMux()
	serveMux_1.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&handlerCalled_1, 1)
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if err := r.ParseMultipartForm(2 * 1024 * 1024); err != nil {
			t.Fatal(err)
		}
		if values := r.MultipartForm.Value["key"]; len(values) != 1 || values[0] != "testkey" {
			t.Fatalf("unexpected key")
		}
		if values := r.MultipartForm.Value["token"]; len(values) != 1 || !strings.HasPrefix(values[0], "testak:") {
			t.Fatalf("unexpected token")
		}
		if values := r.MultipartForm.Value["x-qn-meta-a"]; len(values) != 1 || values[0] != "b" {
			t.Fatalf("unexpected x-qn-meta-a")
		}
		if values := r.MultipartForm.Value["x-qn-meta-c"]; len(values) != 1 || values[0] != "d" {
			t.Fatalf("unexpected x-qn-meta-c")
		}
		if values := r.MultipartForm.Value["x:a"]; len(values) != 1 || values[0] != "b" {
			t.Fatalf("unexpected x:a")
		}
		if values := r.MultipartForm.Value["x:c"]; len(values) != 1 || values[0] != "d" {
			t.Fatalf("unexpected x:c")
		}
		if files := r.MultipartForm.File["file"]; len(files) != 1 || files[0].Filename != "testfilename" || files[0].Size != 1024*1024 {
			t.Fatalf("unexpected file")
		} else if contentType := files[0].Header.Get("Content-Type"); contentType != "application/json" {
			t.Fatalf("unexpected file content-type")
		} else if file, err := files[0].Open(); err != nil {
			t.Fatal(err)
		} else {
			defer file.Close()
			hasher := md5.New()
			if _, err = io.Copy(hasher, file); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(hasher.Sum(nil), expectedMd5) {
				t.Fatalf("unexpected file content")
			}
		}
		w.Header().Add("x-reqid", "fakereqid")
		w.Write([]byte(`{"ok":true}`))
	})
	server_1 := httptest.NewServer(serveMux_1)
	defer server_1.Close()

	serveMux_2 := http.NewServeMux()
	serveMux_2.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&handlerCalled_2, 1)
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if err := r.ParseMultipartForm(2 * 1024 * 1024); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Add("x-reqid", "fakereqid")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"transfer acceleration is not configured on this bucket"}`))
	})
	server_2 := httptest.NewServer(serveMux_2)
	defer server_2.Close()

	var (
		returnValue struct {
			Ok bool `json:"ok"`
		}
		key = "testkey"
	)

	formUploader := NewFormUploader(&FormUploaderOptions{
		Options: http_client.Options{
			Regions: regions{[]*region.Region{
				{Up: region.Endpoints{Accelerated: []string{server_2.URL}, Preferred: []string{server_1.URL}}},
			}},
			Credentials: credentials.NewCredentials("testak", "testsk"),
		},
	})
	if err = formUploader.UploadFile(context.Background(), tmpFile.Name(), &ObjectOptions{
		BucketName:  "testbucket",
		ObjectName:  &key,
		FileName:    "testfilename",
		ContentType: "application/json",
		Metadata:    map[string]string{"a": "b", "c": "d"},
		CustomVars:  map[string]string{"a": "b", "c": "d"},
	}, &returnValue); err != nil {
		t.Fatal(err)
	}
	if !returnValue.Ok {
		t.Fatalf("unexpected response value")
	}
	if count := atomic.LoadUint64(&handlerCalled_1); count != 1 {
		t.Fatalf("unexpected handler call count: %d", count)
	}
	if count := atomic.LoadUint64(&handlerCalled_2); count != 1 {
		t.Fatalf("unexpected handler call count: %d", count)
	}
}

type regions struct {
	regions []*region.Region
}

func (group regions) GetRegions(context.Context) ([]*region.Region, error) {
	return group.regions, nil
}
