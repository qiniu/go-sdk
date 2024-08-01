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
	"github.com/qiniu/go-sdk/v7/storagev2/uploader/source"
)

func TestMultiPartsUploaderV1(t *testing.T) {
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
		w.Header().Add("X-ReqId", "fakereqid")
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
		w.Header().Add("X-ReqId", "fakereqid")
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
		w.Header().Add("X-ReqId", "fakereqid")
		w.Write([]byte(`{"ok":true}`))
	}).Methods(http.MethodPost)
	server = httptest.NewServer(serveMux)
	defer server.Close()

	multiPartsUploaderV1 := uploader.NewMultiPartsUploaderV1(&uploader.MultiPartsUploaderOptions{
		Options: http_client.Options{
			Regions:     &region.Region{Up: region.Endpoints{Preferred: []string{server.URL}}},
			Credentials: credentials.NewCredentials("testak", "testsk"),
		},
	})

	src, err := source.NewFileSource(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	key := "testkey"
	initializedPart, err := multiPartsUploaderV1.InitializeParts(context.Background(), src, &uploader.MultiPartsObjectOptions{
		uploader.ObjectOptions{
			BucketName:  "testbucket",
			ObjectName:  &key,
			FileName:    "testfilename",
			ContentType: "application/json",
			Metadata:    map[string]string{"a": "b", "c": "d"},
			CustomVars:  map[string]string{"a": "b", "c": "d"},
		},
		4 * 1024 * 1024,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer initializedPart.Close()

	part, err := src.Slice(4 * 1024 * 1024)
	if err != nil {
		t.Fatal(err)
	}
	lastUploaded := uint64(0)
	uploadedPart_1, err := multiPartsUploaderV1.UploadPart(context.Background(), initializedPart, part, &uploader.UploadPartOptions{
		OnUploadingProgress: func(progress *uploader.UploadingPartProgress) {
			if progress.PartSize != 4*1024*1024 {
				t.Fatalf("unexpected partSize")
			}
			if progress.Uploaded < lastUploaded || progress.Uploaded > progress.PartSize {
				t.Fatalf("unexpected uploaded")
			}
			lastUploaded = progress.Uploaded
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	part, err = src.Slice(4 * 1024 * 1024)
	if err != nil {
		t.Fatal(err)
	}
	lastUploaded = 0
	uploadedPart_2, err := multiPartsUploaderV1.UploadPart(context.Background(), initializedPart, part, &uploader.UploadPartOptions{
		OnUploadingProgress: func(progress *uploader.UploadingPartProgress) {
			if progress.PartSize != 1*1024*1024 {
				t.Fatalf("unexpected partSize")
			}
			if progress.Uploaded < lastUploaded || progress.Uploaded > progress.PartSize {
				t.Fatalf("unexpected uploaded")
			}
			lastUploaded = progress.Uploaded
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var returnValue struct {
		Ok bool `json:"ok"`
	}
	err = multiPartsUploaderV1.CompleteParts(context.Background(), initializedPart, []uploader.UploadedPart{uploadedPart_1, uploadedPart_2}, &returnValue)
	if err != nil {
		t.Fatal(err)
	} else if !returnValue.Ok {
		t.Fatalf("unexpected response body")
	}
}

func TestMultiPartsUploaderV1Resuming(t *testing.T) {
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
		w.Header().Add("X-ReqId", "fakereqid")
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
	medium := resumableRecorder.OpenForCreatingNew(&resumablerecorder.ResumableRecorderOpenArgs{
		AccessKey:   "testak",
		BucketName:  "testbucket",
		ObjectName:  "testkey",
		SourceID:    tmpFileSourceID,
		PartSize:    4 * 1024 * 1024,
		TotalSize:   5 * 1024 * 1024,
		UpEndpoints: region.Endpoints{Preferred: []string{server.URL}},
	})
	if err = medium.Write(&resumablerecorder.ResumableRecord{
		PartID:     "testctx1",
		Offset:     0,
		PartSize:   4 * 1024 * 1024,
		PartNumber: 1,
		ExpiredAt:  time.Now().Add(1 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if err = medium.Write(&resumablerecorder.ResumableRecord{
		PartID:     "testctx2",
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

	multiPartsUploaderV1 := uploader.NewMultiPartsUploaderV1(&uploader.MultiPartsUploaderOptions{
		ResumableRecorder: resumableRecorder,
		Options: http_client.Options{
			Regions:     &region.Region{Up: region.Endpoints{Preferred: []string{server.URL}}},
			Credentials: credentials.NewCredentials("testak", "testsk"),
		},
	})

	src, err := source.NewFileSource(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	key := "testkey"
	initializedPart := multiPartsUploaderV1.TryToResume(context.Background(), src, &uploader.MultiPartsObjectOptions{
		uploader.ObjectOptions{
			BucketName:  "testbucket",
			ObjectName:  &key,
			FileName:    "testfilename",
			ContentType: "application/json",
			Metadata:    map[string]string{"a": "b", "c": "d"},
			CustomVars:  map[string]string{"a": "b", "c": "d"},
		},
		4 * 1024 * 1024,
	})
	if initializedPart == nil {
		t.Fatalf("initializedPart is nil")
	}
	defer initializedPart.Close()

	part, err := src.Slice(4 * 1024 * 1024)
	if err != nil {
		t.Fatal(err)
	}
	uploadedPart_1, err := multiPartsUploaderV1.UploadPart(context.Background(), initializedPart, part, &uploader.UploadPartOptions{
		OnUploadingProgress: func(progress *uploader.UploadingPartProgress) {
			if progress.PartSize != 4*1024*1024 {
				t.Fatalf("unexpected partSize")
			}
			if progress.Uploaded != 4*1024*1024 {
				t.Fatalf("unexpected uploaded")
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	part, err = src.Slice(4 * 1024 * 1024)
	if err != nil {
		t.Fatal(err)
	}
	uploadedPart_2, err := multiPartsUploaderV1.UploadPart(context.Background(), initializedPart, part, &uploader.UploadPartOptions{
		OnUploadingProgress: func(progress *uploader.UploadingPartProgress) {
			if progress.PartSize != 1024*1024 {
				t.Fatalf("unexpected partSize")
			}
			if progress.Uploaded != 1024*1024 {
				t.Fatalf("unexpected uploaded")
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var returnValue struct {
		Ok bool `json:"ok"`
	}
	err = multiPartsUploaderV1.CompleteParts(context.Background(), initializedPart, []uploader.UploadedPart{uploadedPart_1, uploadedPart_2}, &returnValue)
	if err != nil {
		t.Fatal(err)
	} else if !returnValue.Ok {
		t.Fatalf("unexpected response body")
	}
}
