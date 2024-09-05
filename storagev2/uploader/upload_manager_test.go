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
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
	"github.com/qiniu/go-sdk/v7/storagev2/apis"
	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/region"
	"github.com/qiniu/go-sdk/v7/storagev2/uploader"
)

func TestUploadManagerUploadFile(t *testing.T) {
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

	var (
		uploadManager = uploader.NewUploadManager(&uploader.UploadManagerOptions{
			Options: http_client.Options{
				Regions:     &region.Region{Up: region.Endpoints{Preferred: []string{server.URL}}},
				Credentials: credentials.NewCredentials("testak", "testsk"),
			},
			Concurrency: 2,
		})
		returnValue struct {
			Ok bool `json:"ok"`
		}
		key = "testkey"
	)

	err = uploadManager.UploadFile(context.Background(), tmpFile.Name(), &uploader.ObjectOptions{
		BucketName:  "testbucket",
		ObjectName:  &key,
		FileName:    "testfilename",
		ContentType: "application/json",
		Metadata:    map[string]string{"a": "b", "c": "d"},
		CustomVars:  map[string]string{"a": "b", "c": "d"},
	}, &returnValue)
	if err != nil {
		t.Fatal(err)
	} else if !returnValue.Ok {
		t.Fatalf("unexpected response body")
	}
}

func TestUploadManagerUploadReader(t *testing.T) {
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

	var (
		uploadManager = uploader.NewUploadManager(&uploader.UploadManagerOptions{
			Options: http_client.Options{
				Regions:     &region.Region{Up: region.Endpoints{Preferred: []string{server.URL}}},
				Credentials: credentials.NewCredentials("testak", "testsk"),
			},
			Concurrency: 2,
		})
		returnValue struct {
			Ok bool `json:"ok"`
		}
		key          = "testkey"
		pipeR, pipeW = io.Pipe()
		wg           sync.WaitGroup
	)
	wg.Add(1)
	defer pipeR.Close()

	go func(t *testing.T, w io.WriteCloser) {
		defer wg.Done()
		defer w.Close()
		if _, err = io.Copy(w, tmpFile); err != nil {
			t.Error(err)
		}
	}(t, pipeW)

	err = uploadManager.UploadReader(context.Background(), pipeR, &uploader.ObjectOptions{
		BucketName:  "testbucket",
		ObjectName:  &key,
		FileName:    "testfilename",
		ContentType: "application/json",
		Metadata:    map[string]string{"a": "b", "c": "d"},
		CustomVars:  map[string]string{"a": "b", "c": "d"},
	}, &returnValue)
	if err != nil {
		t.Fatal(err)
	} else if !returnValue.Ok {
		t.Fatalf("unexpected response body")
	}

	wg.Wait()
}

func TestUploadManagerUploadDirectory(t *testing.T) {
	testUploadManagerUploadDirectory(t, true)
	testUploadManagerUploadDirectory(t, false)
}

func testUploadManagerUploadDirectory(t *testing.T, createDirectory bool) {
	var localFiles, remoteObjects sync.Map

	tmpDir_1, err := ioutil.TempDir("", "multi-parts-uploader-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir_1)

	const objectPrefix = "remoteDirectory"
	remoteObjects.Store(objectPrefix+"/", (*os.File)(nil))

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	tmpFile_1, err := ioutil.TempFile(tmpDir_1, "multi-parts-uploader-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile_1.Name())
	defer tmpFile_1.Close()

	if _, err = io.CopyN(tmpFile_1, r, 1024*1024); err != nil {
		t.Fatal(err)
	}
	if relativePath, err := filepath.Rel(tmpDir_1, tmpFile_1.Name()); err != nil {
		t.Fatal(err)
	} else {
		remoteObjects.Store(strings.Replace(filepath.Join(objectPrefix, relativePath), string(filepath.Separator), "/", -1), tmpFile_1)
	}
	localFiles.Store(tmpFile_1.Name(), uint64(0))

	tmpDir_2, err := ioutil.TempDir(tmpDir_1, "multi-parts-uploader-test-*")
	if err != nil {
		t.Fatal(err)
	}
	if relativeDir, err := filepath.Rel(tmpDir_1, tmpDir_2); err != nil {
		t.Fatal(err)
	} else {
		remoteObjects.Store(strings.Replace(filepath.Join(objectPrefix, relativeDir)+string(filepath.Separator), string(filepath.Separator), "/", -1), (*os.File)(nil))
	}

	tmpFile_2, err := ioutil.TempFile(tmpDir_2, "multi-parts-uploader-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile_2.Name())
	defer tmpFile_2.Close()

	if _, err = io.CopyN(tmpFile_2, r, 1024*1024); err != nil {
		t.Fatal(err)
	}
	if relativePath, err := filepath.Rel(tmpDir_1, tmpFile_2.Name()); err != nil {
		t.Fatal(err)
	} else {
		remoteObjects.Store(strings.Replace(filepath.Join(objectPrefix, relativePath), string(filepath.Separator), "/", -1), tmpFile_2)
	}
	localFiles.Store(tmpFile_2.Name(), uint64(0))

	tmpDir_3, err := ioutil.TempDir(tmpDir_2, "multi-parts-uploader-test-*")
	if err != nil {
		t.Fatal(err)
	}
	if relativeDir, err := filepath.Rel(tmpDir_1, tmpDir_3); err != nil {
		t.Fatal(err)
	} else {
		remoteObjects.Store(strings.Replace(filepath.Join(objectPrefix, relativeDir)+string(filepath.Separator), string(filepath.Separator), "/", -1), (*os.File)(nil))
	}

	tmpFile_3, err := ioutil.TempFile(tmpDir_3, "multi-parts-uploader-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile_3.Name())
	defer tmpFile_3.Close()

	if _, err = io.CopyN(tmpFile_3, r, 1024*1024); err != nil {
		t.Fatal(err)
	}
	if relativePath, err := filepath.Rel(tmpDir_1, tmpFile_3.Name()); err != nil {
		t.Fatal(err)
	} else {
		remoteObjects.Store(strings.Replace(filepath.Join(objectPrefix, relativePath), string(filepath.Separator), "/", -1), tmpFile_3)
	}
	localFiles.Store(tmpFile_3.Name(), uint64(0))

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if err := r.ParseMultipartForm(2 * 1024 * 1024); err != nil {
			t.Fatal(err)
		}
		if values := r.MultipartForm.Value["token"]; len(values) != 1 || !strings.HasPrefix(values[0], "testak:") {
			t.Fatalf("unexpected token")
		}

		key := r.MultipartForm.Value["key"][0]
		if expectedValue, ok := remoteObjects.Load(key); !ok {
			t.Fatalf("unexpected key")
		} else if expectedObject, ok := expectedValue.(*os.File); !ok {
			t.Fatalf("unexpected key")
		} else {
			remoteObjects.Delete(key)
			multiPartFile := r.MultipartForm.File["file"][0]
			receivedFile, err := multiPartFile.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer receivedFile.Close()

			receivedFileBytes, err := internal_io.ReadAll(receivedFile)
			if err != nil {
				t.Fatal(err)
			}

			if expectedObject == nil {
				if !createDirectory {
					t.Fatalf("unexpected directory creation")
				}
				if len(receivedFileBytes) != 0 {
					t.Fatalf("content of directory should be empty")
				}
			} else {
				if _, err = expectedObject.Seek(0, io.SeekStart); err != nil {
					t.Fatal(err)
				}
				expectedObjectBytes, err := internal_io.ReadAll(expectedObject)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(expectedObjectBytes, receivedFileBytes) {
					t.Fatalf("unexpected content")
				}
			}
		}
		w.Header().Add("X-ReqId", "fakereqid")
		w.Write([]byte(`{"ok":true}`))
	})
	server := httptest.NewServer(serveMux)
	defer server.Close()

	var uploadManager = uploader.NewUploadManager(&uploader.UploadManagerOptions{
		Options: http_client.Options{
			Regions:     &region.Region{Up: region.Endpoints{Preferred: []string{server.URL}}},
			Credentials: credentials.NewCredentials("testak", "testsk"),
		},
	})

	err = uploadManager.UploadDirectory(context.Background(), tmpDir_1, &uploader.DirectoryOptions{
		BucketName: "testbucket",
		UpdateObjectName: func(path string) string {
			return objectPrefix + "/" + path
		},
		BeforeObjectUpload: func(filePath string, _ *uploader.ObjectOptions) {
			if _, ok := localFiles.Load(filePath); !ok {
				t.Fatalf("unexpected filePath")
			}
		},
		OnUploadingProgress: func(filePath string, progress *uploader.UploadingProgress) {
			if progress.TotalSize != 1024*1024 {
				t.Fatalf("unexpected totalSize")
			}
			if lastUploadedValue, ok := localFiles.Load(filePath); !ok {
				t.Fatalf("unexpected filePath")
			} else if lastUploaded, ok := lastUploadedValue.(uint64); !ok {
				t.Fatalf("unexpected filePath")
			} else if progress.Uploaded < lastUploaded {
				t.Fatalf("unexpected uploaded")
			} else {
				localFiles.Store(filePath, progress.Uploaded)
			}
		},
		OnObjectUploaded: func(filePath string, info *uploader.UploadedObjectInfo) {
			if info.Size != 1024*1024 {
				t.Fatalf("unexpected size")
			}
			if _, ok := localFiles.Load(filePath); !ok {
				t.Fatalf("unexpected filePath")
			}
		},
		ShouldCreateDirectory: createDirectory,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestUploadManagerUploadDirectoryWithFilter(t *testing.T) {
	tmpDir_1, err := ioutil.TempDir("", "multi-parts-uploader-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir_1)

	tmpFile_1, err := ioutil.TempFile(tmpDir_1, "multi-parts-uploader-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile_1.Name())
	defer tmpFile_1.Close()

	tmpFile_2, err := ioutil.TempFile(tmpDir_1, "multi-parts-uploader-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile_2.Name())
	defer tmpFile_2.Close()

	keys := make(map[string]struct{})
	var keysMutex sync.Mutex

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if err := r.ParseMultipartForm(2 * 1024 * 1024); err != nil {
			t.Fatal(err)
		}
		if values := r.MultipartForm.Value["token"]; len(values) != 1 || !strings.HasPrefix(values[0], "testak:") {
			t.Fatalf("unexpected token")
		}

		key := r.MultipartForm.Value["key"][0]
		keysMutex.Lock()
		keys[key] = struct{}{}
		keysMutex.Unlock()
		w.Header().Add("X-ReqId", "fakereqid")
		w.Write([]byte(`{"ok":true}`))
	})
	server := httptest.NewServer(serveMux)
	defer server.Close()

	var uploadManager = uploader.NewUploadManager(&uploader.UploadManagerOptions{
		Options: http_client.Options{
			Regions:     &region.Region{Up: region.Endpoints{Preferred: []string{server.URL}}},
			Credentials: credentials.NewCredentials("testak", "testsk"),
		},
	})

	err = uploadManager.UploadDirectory(context.Background(), tmpDir_1, &uploader.DirectoryOptions{
		BucketName:            "testbucket",
		ShouldCreateDirectory: true,
		ShouldUploadObject: func(_ string, objectOptions *uploader.ObjectOptions) bool {
			return objectOptions != nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(keys) != 2 {
		t.Fatalf("unexpected uploaded oject names count")
	}
	if _, ok := keys[filepath.Base(tmpFile_1.Name())]; !ok {
		t.Fatalf("unexpected uploaded oject name")
	}
	if _, ok := keys[filepath.Base(tmpFile_2.Name())]; !ok {
		t.Fatalf("unexpected uploaded oject name")
	}
}
