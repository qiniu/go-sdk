//go:build unit
// +build unit

package downloader_test

import (
	"context"
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/apis/get_objects"
	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/downloader"
	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/region"
)

func TestDownloadManagerDownloadDirectory(t *testing.T) {
	rsfMux := http.NewServeMux()
	rsfMux.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method")
		}
		w.Header().Set("Content-Type", "application/json")
		query := r.URL.Query()
		if query.Get("bucket") != "bucket1" {
			t.Fatalf("unexpected bucket")
		}
		if query.Get("prefix") != "" {
			t.Fatalf("unexpected prefix")
		}
		if query.Get("limit") != "" {
			t.Fatalf("unexpected limit")
		}
		jsonData, err := json.Marshal(&get_objects.Response{
			Items: []get_objects.ListedObjectEntry{{
				Key:      "test1/file1",
				PutTime:  time.Now().UnixNano() / 100,
				Hash:     "testhash1",
				Size:     4 * 1024 * 1024,
				MimeType: "application/json",
			}, {
				Key:      "test2/file2",
				PutTime:  time.Now().UnixNano() / 100,
				Hash:     "testhash2",
				Size:     4 * 1024 * 1024,
				MimeType: "application/json",
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		w.Header().Add("X-ReqId", "fakereqid")
		w.Write(jsonData)
	})
	rsfServer := httptest.NewServer(rsfMux)
	defer rsfServer.Close()

	ioMux := http.NewServeMux()
	ioMux.HandleFunc("/test1/file1", func(w http.ResponseWriter, r *http.Request) {
		rander := rand.New(rand.NewSource(time.Now().UnixNano()))
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Add("X-ReqId", "fakereqid")
		switch r.Method {
		case http.MethodHead:
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", strconv.Itoa(4*1024*1024))
			w.Header().Set("ETag", "testetag1")
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", strconv.Itoa(4*1024*1024))
			w.Header().Set("ETag", "testetag1")
			io.CopyN(w, rander, 4*1024*1024)
		default:
			t.Fatalf("unexpected method")
		}
	})
	ioMux.HandleFunc("/test2/file2", func(w http.ResponseWriter, r *http.Request) {
		rander := rand.New(rand.NewSource(time.Now().UnixNano()))
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Add("X-ReqId", "fakereqid")
		switch r.Method {
		case http.MethodHead:
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", strconv.Itoa(4*1024*1024))
			w.Header().Set("ETag", "testetag1")
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", strconv.Itoa(4*1024*1024))
			w.Header().Set("ETag", "testetag1")
			io.CopyN(w, rander, 4*1024*1024)
		default:
			t.Fatalf("unexpected method")
		}
	})
	ioServer := httptest.NewServer(ioMux)
	defer ioServer.Close()

	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	downloadManager := downloader.NewDownloadManager(&downloader.DownloadManagerOptions{
		Options: http_client.Options{
			Regions: &region.Region{
				Rsf: region.Endpoints{Preferred: []string{rsfServer.URL}},
			},
			Credentials:         credentials.NewCredentials("testaccesskey", "testsecretkey"),
			UseInsecureProtocol: true,
		},
		DestinationDownloader: downloader.NewConcurrentDownloader(&downloader.ConcurrentDownloaderOptions{
			Concurrency: 1,
			PartSize:    10 * 1024 * 1024,
		}),
	})
	if err = downloadManager.DownloadDirectory(context.Background(), tmpDir, &downloader.DirectoryOptions{
		UseInsecureProtocol:  true,
		BucketName:           "bucket1",
		DownloadURLsProvider: downloader.NewStaticDomainBasedURLsProvider([]string{ioServer.URL}),
	}); err != nil {
		t.Fatal(err)
	}
	if fileInfo, err := os.Stat(filepath.Join(tmpDir, "test1", "file1")); err != nil {
		t.Fatal(err)
	} else if fileInfo.Size() != 4*1024*1024 {
		t.Fatalf("unexpected file size: test1/file1")
	}
	if fileInfo, err := os.Stat(filepath.Join(tmpDir, "test2", "file2")); err != nil {
		t.Fatal(err)
	} else if fileInfo.Size() != 4*1024*1024 {
		t.Fatalf("unexpected file size: test2/file2")
	}
}

func TestDownloadManagerDownloadDirectoryWithPrefix(t *testing.T) {
	rsfMux := http.NewServeMux()
	rsfMux.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method")
		}
		w.Header().Set("Content-Type", "application/json")
		query := r.URL.Query()
		if query.Get("bucket") != "bucket1" {
			t.Fatalf("unexpected bucket")
		}
		if query.Get("prefix") != "test1/" {
			t.Fatalf("unexpected prefix")
		}
		if query.Get("limit") != "" {
			t.Fatalf("unexpected limit")
		}
		jsonData, err := json.Marshal(&get_objects.Response{
			Items: []get_objects.ListedObjectEntry{{
				Key:      "test1/",
				PutTime:  time.Now().UnixNano() / 100,
				Hash:     "testhash1",
				Size:     0,
				MimeType: "application/octet-stream",
			}, {
				Key:      "test1/file1",
				PutTime:  time.Now().UnixNano() / 100,
				Hash:     "testhash1",
				Size:     4 * 1024 * 1024,
				MimeType: "application/json",
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		w.Header().Add("X-ReqId", "fakereqid")
		w.Write(jsonData)
	})
	rsfServer := httptest.NewServer(rsfMux)
	defer rsfServer.Close()

	ioMux := http.NewServeMux()
	ioMux.HandleFunc("/test1/file1", func(w http.ResponseWriter, r *http.Request) {
		rander := rand.New(rand.NewSource(time.Now().UnixNano()))
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Add("X-ReqId", "fakereqid")
		switch r.Method {
		case http.MethodHead:
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", strconv.Itoa(4*1024*1024))
			w.Header().Set("ETag", "testetag1")
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", strconv.Itoa(4*1024*1024))
			w.Header().Set("ETag", "testetag1")
			io.CopyN(w, rander, 4*1024*1024)
		default:
			t.Fatalf("unexpected method")
		}
	})
	ioServer := httptest.NewServer(ioMux)
	defer ioServer.Close()

	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	downloadManager := downloader.NewDownloadManager(&downloader.DownloadManagerOptions{
		Options: http_client.Options{
			Regions: &region.Region{
				Rsf: region.Endpoints{Preferred: []string{rsfServer.URL}},
			},
			Credentials:         credentials.NewCredentials("testaccesskey", "testsecretkey"),
			UseInsecureProtocol: true,
		},
		DestinationDownloader: downloader.NewConcurrentDownloader(&downloader.ConcurrentDownloaderOptions{
			Concurrency: 1,
			PartSize:    10 * 1024 * 1024,
		}),
	})
	if err = downloadManager.DownloadDirectory(context.Background(), tmpDir, &downloader.DirectoryOptions{
		UseInsecureProtocol:  true,
		BucketName:           "bucket1",
		ObjectPrefix:         "test1/",
		DownloadURLsProvider: downloader.NewStaticDomainBasedURLsProvider([]string{ioServer.URL}),
	}); err != nil {
		t.Fatal(err)
	}
	if fileInfo, err := os.Stat(filepath.Join(tmpDir, "file1")); err != nil {
		t.Fatal(err)
	} else if fileInfo.Size() != 4*1024*1024 {
		t.Fatalf("unexpected file size: file1")
	}
}

func TestDownloadManagerDownloadDirectoryWithFilter(t *testing.T) {
	rsfMux := http.NewServeMux()
	rsfMux.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method")
		}
		w.Header().Set("Content-Type", "application/json")
		query := r.URL.Query()
		if query.Get("bucket") != "bucket1" {
			t.Fatalf("unexpected bucket")
		}
		if query.Get("prefix") != "" {
			t.Fatalf("unexpected prefix")
		}
		if query.Get("limit") != "" {
			t.Fatalf("unexpected limit")
		}
		jsonData, err := json.Marshal(&get_objects.Response{
			Items: []get_objects.ListedObjectEntry{{
				Key:      "test1/",
				PutTime:  time.Now().UnixNano() / 100,
				Hash:     "testhash1",
				Size:     0,
				MimeType: "application/octet-stream",
			}, {
				Key:      "test1/file1",
				PutTime:  time.Now().UnixNano() / 100,
				Hash:     "testhash1",
				Size:     4 * 1024 * 1024,
				MimeType: "application/json",
			}, {
				Key:      "test2/",
				PutTime:  time.Now().UnixNano() / 100,
				Hash:     "testhash2",
				Size:     0,
				MimeType: "application/octet-stream",
			}, {
				Key:      "test2/file2",
				PutTime:  time.Now().UnixNano() / 100,
				Hash:     "testhash2",
				Size:     4 * 1024 * 1024,
				MimeType: "application/json",
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		w.Header().Add("X-ReqId", "fakereqid")
		w.Write(jsonData)
	})
	rsfServer := httptest.NewServer(rsfMux)
	defer rsfServer.Close()

	ioMux := http.NewServeMux()
	ioMux.HandleFunc("/test1/file1", func(w http.ResponseWriter, r *http.Request) {
		rander := rand.New(rand.NewSource(time.Now().UnixNano()))
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Add("X-ReqId", "fakereqid")
		switch r.Method {
		case http.MethodHead:
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", strconv.Itoa(4*1024*1024))
			w.Header().Set("ETag", "testetag1")
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", strconv.Itoa(4*1024*1024))
			w.Header().Set("ETag", "testetag1")
			io.CopyN(w, rander, 4*1024*1024)
		default:
			t.Fatalf("unexpected method")
		}
	})
	ioServer := httptest.NewServer(ioMux)
	defer ioServer.Close()

	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	downloadManager := downloader.NewDownloadManager(&downloader.DownloadManagerOptions{
		Options: http_client.Options{
			Regions: &region.Region{
				Rsf: region.Endpoints{Preferred: []string{rsfServer.URL}},
			},
			Credentials:         credentials.NewCredentials("testaccesskey", "testsecretkey"),
			UseInsecureProtocol: true,
		},
		DestinationDownloader: downloader.NewConcurrentDownloader(&downloader.ConcurrentDownloaderOptions{
			Concurrency: 1,
			PartSize:    10 * 1024 * 1024,
		}),
	})
	if err = downloadManager.DownloadDirectory(context.Background(), tmpDir, &downloader.DirectoryOptions{
		UseInsecureProtocol:  true,
		BucketName:           "bucket1",
		DownloadURLsProvider: downloader.NewStaticDomainBasedURLsProvider([]string{ioServer.URL}),
		ShouldDownloadObject: func(objectName string) bool {
			return strings.HasPrefix(objectName, "test1/")
		},
	}); err != nil {
		t.Fatal(err)
	}
	if fileInfo, err := os.Stat(filepath.Join(tmpDir, "test1", "file1")); err != nil {
		t.Fatal(err)
	} else if fileInfo.Size() != 4*1024*1024 {
		t.Fatalf("unexpected file size: test1/file1")
	}
	if _, err = os.Stat(filepath.Join(tmpDir, "test2")); err == nil {
		t.Fatalf("unexpected test2/")
	}
	if _, err = os.Stat(filepath.Join(tmpDir, "test2", "file2")); err == nil {
		t.Fatalf("unexpected test2/file2")
	}
}
