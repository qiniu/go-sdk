//go:build unit
// +build unit

package uploader_test

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestMultiPartsUploaderV2(t *testing.T) {
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

	serveMux := mux.NewRouter()
	serveMux.HandleFunc("/buckets/{bucketName}/objects/{encodedObjectName}/uploads", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "UpToken testak:") {
			t.Fatalf("unexpected authorization")
		}
		vars := mux.Vars(r)
		if vars["bucketName"] != "testbucket" {
			t.Fatalf("unexpected bucket name")
		}
		objectBytes, err := base64.URLEncoding.DecodeString(vars["encodedObjectName"])
		if err != nil {
			t.Fatal(err)
		} else if string(objectBytes) != "testkey" {
			t.Fatalf("unexpected object name")
		}
		jsonBytes, err := json.Marshal(&apis.ResumableUploadV2InitiateMultipartUploadResponse{
			UploadId:  "testuploadID",
			ExpiredAt: time.Now().Add(1 * time.Hour).Unix(),
		})
		if err != nil {
			t.Fatal(err)
		}
		w.Header().Add("X-ReqId", "fakereqid")
		w.Write(jsonBytes)
	}).Methods(http.MethodPost)
	serveMux.HandleFunc("/buckets/{bucketName}/objects/{encodedObjectName}/uploads/{uploadID}/{partNumber}", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "UpToken testak:") {
			t.Fatalf("unexpected authorization")
		}
		vars := mux.Vars(r)
		if vars["bucketName"] != "testbucket" {
			t.Fatalf("unexpected bucket name")
		}
		objectBytes, err := base64.URLEncoding.DecodeString(vars["encodedObjectName"])
		if err != nil {
			t.Fatal(err)
		} else if string(objectBytes) != "testkey" {
			t.Fatalf("unexpected object name")
		}
		if vars["uploadID"] != "testuploadID" {
			t.Fatalf("unexpected upload id")
		}
		actualBody, err := internal_io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		var expectedBody, jsonBody []byte
		switch vars["partNumber"] {
		case "1":
			expectedBody, err = internal_io.ReadAll(io.NewSectionReader(tmpFile, 0, 4*1024*1024))
			if err != nil {
				t.Fatal(err)
			}
		case "2":
			expectedBody, err = internal_io.ReadAll(io.NewSectionReader(tmpFile, 4*1024*1024, 1024*1024))
			if err != nil {
				t.Fatal(err)
			}
		default:
			t.Fatalf("unexpected part number")
		}
		if !bytes.Equal(actualBody, expectedBody) {
			t.Fatalf("unexpected body")
		}
		md5Sum := md5.Sum(actualBody)
		if r.Header.Get("Content-MD5") != hex.EncodeToString(md5Sum[:]) {
			t.Fatalf("unexpected content-md5")
		}
		switch vars["partNumber"] {
		case "1":
			jsonBody, err = json.Marshal(&apis.ResumableUploadV2UploadPartResponse{
				Etag: "testetag1",
				Md5:  r.Header.Get("Content-MD5"),
			})
		case "2":
			jsonBody, err = json.Marshal(&apis.ResumableUploadV2UploadPartResponse{
				Etag: "testetag2",
				Md5:  r.Header.Get("Content-MD5"),
			})
		}
		if err != nil {
			t.Fatal(err)
		}
		w.Header().Add("X-ReqId", "fakereqid")
		w.Write(jsonBody)
	}).Methods(http.MethodPut)
	serveMux.HandleFunc("/buckets/{bucketName}/objects/{encodedObjectName}/uploads/{uploadID}", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "UpToken testak:") {
			t.Fatalf("unexpected authorization")
		}
		vars := mux.Vars(r)
		if vars["bucketName"] != "testbucket" {
			t.Fatalf("unexpected bucket name")
		}
		objectBytes, err := base64.URLEncoding.DecodeString(vars["encodedObjectName"])
		if err != nil {
			t.Fatal(err)
		} else if string(objectBytes) != "testkey" {
			t.Fatalf("unexpected object name")
		}
		if vars["uploadID"] != "testuploadID" {
			t.Fatalf("unexpected upload id")
		}
		requestBodyBytes, err := internal_io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		var body apis.ResumableUploadV2CompleteMultipartUploadRequest
		if err = body.UnmarshalJSON(requestBodyBytes); err != nil {
			t.Fatalf("unexpected request body")
		}
		if len(body.Parts) != 2 {
			t.Fatalf("unexpected parts")
		} else if body.Parts[0].PartNumber != 1 {
			t.Fatalf("unexpected part number")
		} else if body.Parts[0].Etag != "testetag1" {
			t.Fatalf("unexpected part number")
		} else if body.Parts[1].PartNumber != 2 {
			t.Fatalf("unexpected part number")
		} else if body.Parts[1].Etag != "testetag2" {
			t.Fatalf("unexpected part number")
		}
		if body.FileName != "testfilename" {
			t.Fatalf("unexpected fileName")
		}
		if body.MimeType != "application/json" {
			t.Fatalf("unexpected mimeType")
		}
		if len(body.Metadata) != 2 {
			t.Fatalf("unexpected metadata")
		} else if body.Metadata["x-qn-meta-a"] != "b" {
			t.Fatalf("unexpected x-qn-meta-a")
		} else if body.Metadata["x-qn-meta-c"] != "d" {
			t.Fatalf("unexpected x-qn-meta-c")
		} else if body.CustomVars["x:a"] != "b" {
			t.Fatalf("unexpected x:a")
		} else if body.CustomVars["x:c"] != "d" {
			t.Fatalf("unexpected x:c")
		}
		w.Header().Add("X-ReqId", "fakereqid")
		w.Write([]byte(`{"ok":true}`))
	}).Methods(http.MethodPost)
	server := httptest.NewServer(serveMux)
	defer server.Close()

	multiPartsUploaderV2 := uploader.NewMultiPartsUploaderV2(&uploader.MultiPartsUploaderOptions{
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
	initializedPart, err := multiPartsUploaderV2.InitializeParts(context.Background(), src, &uploader.MultiPartsObjectOptions{
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
	uploadedPart_1, err := multiPartsUploaderV2.UploadPart(context.Background(), initializedPart, part, &uploader.UploadPartOptions{
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
	uploadedPart_2, err := multiPartsUploaderV2.UploadPart(context.Background(), initializedPart, part, &uploader.UploadPartOptions{
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
	err = multiPartsUploaderV2.CompleteParts(context.Background(), initializedPart, []uploader.UploadedPart{uploadedPart_1, uploadedPart_2}, &returnValue)
	if err != nil {
		t.Fatal(err)
	} else if !returnValue.Ok {
		t.Fatalf("unexpected response body")
	}
}

func TestMultiPartsUploaderV2Resuming(t *testing.T) {
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

	serveMux := mux.NewRouter()
	serveMux.HandleFunc("/buckets/{bucketName}/objects/{encodedObjectName}/uploads/{uploadID}", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "UpToken testak:") {
			t.Fatalf("unexpected authorization")
		}
		vars := mux.Vars(r)
		if vars["bucketName"] != "testbucket" {
			t.Fatalf("unexpected bucket name")
		}
		objectBytes, err := base64.URLEncoding.DecodeString(vars["encodedObjectName"])
		if err != nil {
			t.Fatal(err)
		} else if string(objectBytes) != "testkey" {
			t.Fatalf("unexpected object name")
		}
		if vars["uploadID"] != "testuploadID" {
			t.Fatalf("unexpected upload id")
		}
		requestBodyBytes, err := internal_io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		var body apis.ResumableUploadV2CompleteMultipartUploadRequest
		if err = body.UnmarshalJSON(requestBodyBytes); err != nil {
			t.Fatalf("unexpected request body")
		}
		if len(body.Parts) != 2 {
			t.Fatalf("unexpected parts")
		} else if body.Parts[0].PartNumber != 1 {
			t.Fatalf("unexpected part number")
		} else if body.Parts[0].Etag != "testetag1" {
			t.Fatalf("unexpected part number")
		} else if body.Parts[1].PartNumber != 2 {
			t.Fatalf("unexpected part number")
		} else if body.Parts[1].Etag != "testetag2" {
			t.Fatalf("unexpected part number")
		}
		if body.FileName != "testfilename" {
			t.Fatalf("unexpected fileName")
		}
		if body.MimeType != "application/json" {
			t.Fatalf("unexpected mimeType")
		}
		if len(body.Metadata) != 2 {
			t.Fatalf("unexpected metadata")
		} else if body.Metadata["x-qn-meta-a"] != "b" {
			t.Fatalf("unexpected x-qn-meta-a")
		} else if body.Metadata["x-qn-meta-c"] != "d" {
			t.Fatalf("unexpected x-qn-meta-c")
		} else if body.CustomVars["x:a"] != "b" {
			t.Fatalf("unexpected x:a")
		} else if body.CustomVars["x:c"] != "d" {
			t.Fatalf("unexpected x:c")
		}
		w.Header().Add("X-ReqId", "fakereqid")
		w.Write([]byte(`{"ok":true}`))
	}).Methods(http.MethodPost)
	server := httptest.NewServer(serveMux)
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
		UploadID:   "testuploadID",
		PartID:     "testetag1",
		Offset:     0,
		PartSize:   4 * 1024 * 1024,
		PartNumber: 1,
		ExpiredAt:  time.Now().Add(1 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if err = medium.Write(&resumablerecorder.ResumableRecord{
		UploadID:   "testuploadID",
		PartID:     "testetag2",
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

	multiPartsUploaderV2 := uploader.NewMultiPartsUploaderV2(&uploader.MultiPartsUploaderOptions{
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
	initializedPart := multiPartsUploaderV2.TryToResume(context.Background(), src, &uploader.MultiPartsObjectOptions{
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
	uploadedPart_1, err := multiPartsUploaderV2.UploadPart(context.Background(), initializedPart, part, &uploader.UploadPartOptions{
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
	uploadedPart_2, err := multiPartsUploaderV2.UploadPart(context.Background(), initializedPart, part, &uploader.UploadPartOptions{
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
	err = multiPartsUploaderV2.CompleteParts(context.Background(), initializedPart, []uploader.UploadedPart{uploadedPart_1, uploadedPart_2}, &returnValue)
	if err != nil {
		t.Fatal(err)
	} else if !returnValue.Ok {
		t.Fatalf("unexpected response body")
	}
}
