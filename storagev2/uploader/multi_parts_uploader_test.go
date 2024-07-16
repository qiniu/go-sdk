//go:build unit
// +build unit

package uploader_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/mux"
	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
	"github.com/qiniu/go-sdk/v7/storagev2/apis"
	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/region"
	"github.com/qiniu/go-sdk/v7/storagev2/uploader"
	resumablerecorder "github.com/qiniu/go-sdk/v7/storagev2/uploader/resumable_recorder"
)

func TestMultiPartsUploader(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "multi-parts-uploader-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	if _, err = io.CopyN(tmpFile, r, 5*1024*1024); err != nil {
		t.Fatal(err)
	}
	if _, err = tmpFile.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}

	var server *httptest.Server
	serveMux := mux.NewRouter()
	serveMux.HandleFunc("/mkblk/4194304", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "UpToken testak:") {
			t.Fatalf("unexpected authorization")
		}
		actualBody, err := internal_io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		expectedBody, err := internal_io.ReadAll(io.NewSectionReader(tmpFile, 0, 4*1024*1024))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(actualBody, expectedBody) {
			t.Fatalf("unexpected body")
		}
		jsonBody, err := json.Marshal(&apis.ResumableUploadV1MakeBlockResponse{
			Ctx:       "testctx1",
			Checksum:  "testchecksum1",
			Crc32:     int64(crc32.ChecksumIEEE(actualBody)),
			Host:      server.URL,
			ExpiredAt: time.Now().Add(1 * time.Hour).Unix(),
		})
		if err != nil {
			t.Fatal(err)
		}
		w.Write(jsonBody)
	}).Methods(http.MethodPost)
	serveMux.HandleFunc("/mkblk/1048576", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "UpToken testak:") {
			t.Fatalf("unexpected authorization")
		}
		actualBody, err := internal_io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		expectedBody, err := internal_io.ReadAll(io.NewSectionReader(tmpFile, 4*1024*1024, 1*1024*1024))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(actualBody, expectedBody) {
			t.Fatalf("unexpected body")
		}
		jsonBody, err := json.Marshal(&apis.ResumableUploadV1MakeBlockResponse{
			Ctx:       "testctx2",
			Checksum:  "testchecksum2",
			Crc32:     int64(crc32.ChecksumIEEE(actualBody)),
			Host:      server.URL,
			ExpiredAt: time.Now().Add(1 * time.Hour).Unix(),
		})
		if err != nil {
			t.Fatal(err)
		}
		w.Write(jsonBody)
	}).Methods(http.MethodPost)
	serveMux.PathPrefix("/mkfile/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "UpToken testak:") {
			t.Fatalf("unexpected authorization")
		}
		components := strings.Split(strings.TrimPrefix(r.URL.Path, "/mkfile/"), "/")
		if components[0] != strconv.FormatInt(5*1024*1024, 10) {
			t.Fatalf("unexpected fileSize")
		}
		components = components[1:]
		for len(components) > 0 {
			switch components[0] {
			case "key":
				value := components[1]
				valueBytes, err := base64.URLEncoding.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				if string(valueBytes) != "testkey" {
					t.Fatalf("unexpected key")
				}
			case "fname":
				value := components[1]
				valueBytes, err := base64.URLEncoding.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				if string(valueBytes) != "testfilename" {
					t.Fatalf("unexpected fname")
				}
			case "mimeType":
				value := components[1]
				valueBytes, err := base64.URLEncoding.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				if string(valueBytes) != "application/json" {
					t.Fatalf("unexpected mimeType")
				}
			case "x-qn-meta-a":
				value := components[1]
				valueBytes, err := base64.URLEncoding.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				if string(valueBytes) != "b" {
					t.Fatalf("unexpected x-qn-meta-a")
				}
			case "x-qn-meta-c":
				value := components[1]
				valueBytes, err := base64.URLEncoding.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				if string(valueBytes) != "d" {
					t.Fatalf("unexpected x-qn-meta-c")
				}
			case "x:a":
				value := components[1]
				valueBytes, err := base64.URLEncoding.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				if string(valueBytes) != "b" {
					t.Fatalf("unexpected x:a")
				}
			case "x:c":
				value := components[1]
				valueBytes, err := base64.URLEncoding.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				if string(valueBytes) != "d" {
					t.Fatalf("unexpected x:c")
				}
			default:
				t.Fatalf("unexpected component key: %s", components[0])
			}
			components = components[2:]
		}
		actualBody, err := internal_io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(actualBody) != "testctx1,testctx2" {
			t.Fatalf("unexpected body")
		}
		w.Write([]byte(`{"ok":true}`))
	}).Methods(http.MethodPost)
	server = httptest.NewServer(serveMux)
	defer server.Close()

	multiPartsUploader := uploader.NewMultiPartsUploader(uploader.NewConcurrentMultiPartsUploaderScheduler(
		uploader.NewMultiPartsUploaderV1(&uploader.MultiPartsUploaderOptions{
			Options: http_client.Options{
				Regions:     &region.Region{Up: region.Endpoints{Preferred: []string{server.URL}}},
				Credentials: credentials.NewCredentials("testak", "testsk"),
			},
		}), &uploader.ConcurrentMultiPartsUploaderSchedulerOptions{PartSize: 1 << 22, Concurrency: 2},
	))

	var (
		key         = "testkey"
		returnValue struct {
			Ok bool `json:"ok"`
		}
		lastUploaded uint64
	)
	if err = multiPartsUploader.UploadFile(context.Background(), tmpFile.Name(), &uploader.ObjectOptions{
		BucketName:  "testbucket",
		ObjectName:  &key,
		FileName:    "testfilename",
		ContentType: "application/json",
		Metadata:    map[string]string{"a": "b", "c": "d"},
		CustomVars:  map[string]string{"a": "b", "c": "d"},
		OnUploadingProgress: func(uploaded uint64, fileSize uint64) {
			if fileSize != 5*1024*1024 {
				t.Fatalf("unexpected file size")
			} else if uploaded > fileSize {
				t.Fatalf("unexpected uploaded")
			} else if lu := atomic.SwapUint64(&lastUploaded, uploaded); lu > uploaded || lu > fileSize {
				t.Fatalf("unexpected uploaded")
			}
		},
	}, &returnValue); err != nil {
		t.Fatal(err)
	} else if !returnValue.Ok {
		t.Fatalf("unexpected response body")
	}
}

func TestMultiPartsUploaderResuming(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "multi-parts-uploader-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile, err := ioutil.TempFile("", "multi-parts-uploader-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	if _, err = io.CopyN(tmpFile, r, 5*1024*1024); err != nil {
		t.Fatal(err)
	}
	if _, err = tmpFile.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}

	var server *httptest.Server
	serveMux := mux.NewRouter()
	serveMux.PathPrefix("/mkfile/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "UpToken testak:") {
			t.Fatalf("unexpected authorization")
		}
		components := strings.Split(strings.TrimPrefix(r.URL.Path, "/mkfile/"), "/")
		if components[0] != strconv.FormatInt(5*1024*1024, 10) {
			t.Fatalf("unexpected fileSize")
		}
		components = components[1:]
		for len(components) > 0 {
			switch components[0] {
			case "key":
				value := components[1]
				valueBytes, err := base64.URLEncoding.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				if string(valueBytes) != "testkey" {
					t.Fatalf("unexpected key")
				}
			case "fname":
				value := components[1]
				valueBytes, err := base64.URLEncoding.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				if string(valueBytes) != "testfilename" {
					t.Fatalf("unexpected fname")
				}
			case "mimeType":
				value := components[1]
				valueBytes, err := base64.URLEncoding.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				if string(valueBytes) != "application/json" {
					t.Fatalf("unexpected mimeType")
				}
			case "x-qn-meta-a":
				value := components[1]
				valueBytes, err := base64.URLEncoding.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				if string(valueBytes) != "b" {
					t.Fatalf("unexpected x-qn-meta-a")
				}
			case "x-qn-meta-c":
				value := components[1]
				valueBytes, err := base64.URLEncoding.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				if string(valueBytes) != "d" {
					t.Fatalf("unexpected x-qn-meta-c")
				}
			case "x:a":
				value := components[1]
				valueBytes, err := base64.URLEncoding.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				if string(valueBytes) != "b" {
					t.Fatalf("unexpected x:a")
				}
			case "x:c":
				value := components[1]
				valueBytes, err := base64.URLEncoding.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				if string(valueBytes) != "d" {
					t.Fatalf("unexpected x:c")
				}
			default:
				t.Fatalf("unexpected component key: %s", components[0])
			}
			components = components[2:]
		}
		actualBody, err := internal_io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(actualBody) != "testctx1,testctx2" {
			t.Fatalf("unexpected body")
		}
		w.Write([]byte(`{"ok":true}`))
	}).Methods(http.MethodPost)
	server = httptest.NewServer(serveMux)
	defer server.Close()

	tmpFileStat, err := tmpFile.Stat()
	if err != nil {
		t.Fatal(err)
	}
	tmpFileSourceID := fmt.Sprintf("%d:%d:%s", tmpFileStat.Size(), tmpFileStat.ModTime().UnixNano(), tmpFile.Name())

	resumableRecorder := resumablerecorder.NewJsonFileSystemResumableRecorder(tmpDir)
	medium := resumableRecorder.OpenForCreatingNew(&resumablerecorder.ResumableRecorderOpenOptions{
		AccessKey:   "testak",
		BucketName:  "testbucket",
		ObjectName:  "testkey",
		SourceID:    tmpFileSourceID,
		PartSize:    4 * 1024 * 1024,
		TotalSize:   5 * 1024 * 1024,
		UpEndpoints: region.Endpoints{Preferred: []string{server.URL}},
	})
	if err = medium.Write(&resumablerecorder.ResumableRecord{
		PartId:     "testctx1",
		Offset:     0,
		PartSize:   4 * 1024 * 1024,
		PartNumber: 1,
		ExpiredAt:  time.Now().Add(1 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if err = medium.Write(&resumablerecorder.ResumableRecord{
		PartId:     "testctx2",
		Offset:     4 * 1024 * 1024,
		PartSize:   1 * 1024 * 1024,
		PartNumber: 2,
		ExpiredAt:  time.Now().Add(1 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if err = medium.Close(); err != nil {
		t.Fatal(err)
	}

	multiPartsUploader := uploader.NewMultiPartsUploader(uploader.NewConcurrentMultiPartsUploaderScheduler(
		uploader.NewMultiPartsUploaderV1(&uploader.MultiPartsUploaderOptions{
			ResumableRecorder: resumableRecorder,
			Options: http_client.Options{
				Regions:     &region.Region{Up: region.Endpoints{Preferred: []string{server.URL}}},
				Credentials: credentials.NewCredentials("testak", "testsk"),
			},
		}), &uploader.ConcurrentMultiPartsUploaderSchedulerOptions{PartSize: 1 << 22, Concurrency: 2},
	))

	var (
		key         = "testkey"
		returnValue struct {
			Ok bool `json:"ok"`
		}
		lastUploaded uint64
	)
	if err = multiPartsUploader.UploadFile(context.Background(), tmpFile.Name(), &uploader.ObjectOptions{
		BucketName:  "testbucket",
		ObjectName:  &key,
		FileName:    "testfilename",
		ContentType: "application/json",
		Metadata:    map[string]string{"a": "b", "c": "d"},
		CustomVars:  map[string]string{"a": "b", "c": "d"},
		OnUploadingProgress: func(uploaded uint64, fileSize uint64) {
			if fileSize != 5*1024*1024 {
				t.Fatalf("unexpected file size")
			} else if uploaded > fileSize {
				t.Fatalf("unexpected uploaded")
			} else if lu := atomic.SwapUint64(&lastUploaded, uploaded); lu > uploaded || lu > fileSize {
				t.Fatalf("unexpected uploaded")
			}
		},
	}, &returnValue); err != nil {
		t.Fatal(err)
	} else if !returnValue.Ok {
		t.Fatalf("unexpected response body")
	}
}

func TestMultiPartsUploaderRetry(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "multi-parts-uploader-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	if _, err = io.CopyN(tmpFile, r, 5*1024*1024); err != nil {
		t.Fatal(err)
	}
	if _, err = tmpFile.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}

	var handlerCalled_1, handlerCalled_2, handlerCalled_3 uint64
	serveMux_1 := mux.NewRouter()
	serveMux_1.HandleFunc("/mkblk/{blockSize}", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&handlerCalled_1, 1)
		w.WriteHeader(599)
	}).Methods(http.MethodPost)
	server_1 := httptest.NewServer(serveMux_1)
	defer server_1.Close()

	serveMux_2 := mux.NewRouter()
	serveMux_2.HandleFunc("/mkblk/{blockSize}", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&handlerCalled_2, 1)
		w.WriteHeader(599)
	}).Methods(http.MethodPost)
	server_2 := httptest.NewServer(serveMux_2)
	defer server_2.Close()

	var server_3 *httptest.Server
	serveMux_3 := mux.NewRouter()
	serveMux_3.HandleFunc("/mkblk/4194304", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&handlerCalled_3, 1)
		if !strings.HasPrefix(r.Header.Get("Authorization"), "UpToken testak:") {
			t.Fatalf("unexpected authorization")
		}
		actualBody, err := internal_io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		expectedBody, err := internal_io.ReadAll(io.NewSectionReader(tmpFile, 0, 4*1024*1024))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(actualBody, expectedBody) {
			t.Fatalf("unexpected body")
		}
		jsonBody, err := json.Marshal(&apis.ResumableUploadV1MakeBlockResponse{
			Ctx:       "testctx1",
			Checksum:  "testchecksum1",
			Crc32:     int64(crc32.ChecksumIEEE(actualBody)),
			Host:      server_3.URL,
			ExpiredAt: time.Now().Add(1 * time.Hour).Unix(),
		})
		if err != nil {
			t.Fatal(err)
		}
		w.Write(jsonBody)
	}).Methods(http.MethodPost)
	serveMux_3.HandleFunc("/mkblk/1048576", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&handlerCalled_3, 1)
		if !strings.HasPrefix(r.Header.Get("Authorization"), "UpToken testak:") {
			t.Fatalf("unexpected authorization")
		}
		actualBody, err := internal_io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		expectedBody, err := internal_io.ReadAll(io.NewSectionReader(tmpFile, 4*1024*1024, 1*1024*1024))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(actualBody, expectedBody) {
			t.Fatalf("unexpected body")
		}
		jsonBody, err := json.Marshal(&apis.ResumableUploadV1MakeBlockResponse{
			Ctx:       "testctx2",
			Checksum:  "testchecksum2",
			Crc32:     int64(crc32.ChecksumIEEE(actualBody)),
			Host:      server_3.URL,
			ExpiredAt: time.Now().Add(1 * time.Hour).Unix(),
		})
		if err != nil {
			t.Fatal(err)
		}
		w.Write(jsonBody)
	}).Methods(http.MethodPost)
	serveMux_3.PathPrefix("/mkfile/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&handlerCalled_3, 1)
		if !strings.HasPrefix(r.Header.Get("Authorization"), "UpToken testak:") {
			t.Fatalf("unexpected authorization")
		}
		components := strings.Split(strings.TrimPrefix(r.URL.Path, "/mkfile/"), "/")
		if components[0] != strconv.FormatInt(5*1024*1024, 10) {
			t.Fatalf("unexpected fileSize")
		}
		components = components[1:]
		for len(components) > 0 {
			switch components[0] {
			case "key":
				value := components[1]
				valueBytes, err := base64.URLEncoding.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				if string(valueBytes) != "testkey" {
					t.Fatalf("unexpected key")
				}
			case "fname":
				value := components[1]
				valueBytes, err := base64.URLEncoding.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				if string(valueBytes) != "testfilename" {
					t.Fatalf("unexpected fname")
				}
			case "mimeType":
				value := components[1]
				valueBytes, err := base64.URLEncoding.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				if string(valueBytes) != "application/json" {
					t.Fatalf("unexpected mimeType")
				}
			case "x-qn-meta-a":
				value := components[1]
				valueBytes, err := base64.URLEncoding.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				if string(valueBytes) != "b" {
					t.Fatalf("unexpected x-qn-meta-a")
				}
			case "x-qn-meta-c":
				value := components[1]
				valueBytes, err := base64.URLEncoding.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				if string(valueBytes) != "d" {
					t.Fatalf("unexpected x-qn-meta-c")
				}
			case "x:a":
				value := components[1]
				valueBytes, err := base64.URLEncoding.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				if string(valueBytes) != "b" {
					t.Fatalf("unexpected x:a")
				}
			case "x:c":
				value := components[1]
				valueBytes, err := base64.URLEncoding.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				if string(valueBytes) != "d" {
					t.Fatalf("unexpected x:c")
				}
			default:
				t.Fatalf("unexpected component key: %s", components[0])
			}
			components = components[2:]
		}
		actualBody, err := internal_io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(actualBody) != "testctx1,testctx2" {
			t.Fatalf("unexpected body")
		}
		w.Write([]byte(`{"ok":true}`))
	}).Methods(http.MethodPost)
	server_3 = httptest.NewServer(serveMux_3)
	defer server_3.Close()

	multiPartsUploader := uploader.NewMultiPartsUploader(uploader.NewSerialMultiPartsUploaderScheduler(
		uploader.NewMultiPartsUploaderV1(&uploader.MultiPartsUploaderOptions{
			Options: http_client.Options{
				Regions: regions{[]*region.Region{
					{Up: region.Endpoints{Preferred: []string{server_1.URL}}},
					{Up: region.Endpoints{Preferred: []string{server_2.URL}}},
					{Up: region.Endpoints{Preferred: []string{server_3.URL}}},
				}},
				Credentials: credentials.NewCredentials("testak", "testsk"),
			},
		}), &uploader.SerialMultiPartsUploaderSchedulerOptions{PartSize: 1 << 22},
	))

	var (
		key         = "testkey"
		returnValue struct {
			Ok bool `json:"ok"`
		}
		lastUploaded uint64
	)
	if err = multiPartsUploader.UploadFile(context.Background(), tmpFile.Name(), &uploader.ObjectOptions{
		BucketName:  "testbucket",
		ObjectName:  &key,
		FileName:    "testfilename",
		ContentType: "application/json",
		Metadata:    map[string]string{"a": "b", "c": "d"},
		CustomVars:  map[string]string{"a": "b", "c": "d"},
		OnUploadingProgress: func(uploaded uint64, fileSize uint64) {
			if fileSize != 5*1024*1024 {
				t.Fatalf("unexpected file size")
			}
			atomic.StoreUint64(&lastUploaded, uploaded)
		},
	}, &returnValue); err != nil {
		t.Fatal(err)
	} else if !returnValue.Ok {
		t.Fatalf("unexpected response body")
	}

	if fileSize := atomic.LoadUint64(&lastUploaded); fileSize != 5*1024*1024 {
		t.Fatalf("unexpected file size: %d", fileSize)
	}
	if count := atomic.LoadUint64(&handlerCalled_1); count != 4 {
		t.Fatalf("unexpected handler call count: %d", count)
	}
	if count := atomic.LoadUint64(&handlerCalled_2); count != 4 {
		t.Fatalf("unexpected handler call count: %d", count)
	}
	if count := atomic.LoadUint64(&handlerCalled_3); count != 3 {
		t.Fatalf("unexpected handler call count: %d", count)
	}
}
