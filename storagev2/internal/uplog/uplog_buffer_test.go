//go:build unit
// +build unit

package uplog

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var testLock sync.Mutex

func TestUplogWriteMemoryBufferToFileBuffer(t *testing.T) {
	testLock.Lock()
	defer testLock.Unlock()

	tmpDir, err := ioutil.TempDir("", "test-uplog-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	defer func() {
		if err := FlushBuffer(); err != nil {
			t.Fatal(err)
		}
	}()

	SetUplogFileBufferDirPath(tmpDir)
	defer SetUplogFileBufferDirPath("")

	DisableUplog()
	defer EnableUplog()

	uplogBuffer := bytes.NewBuffer(make([]byte, 0, 4*1024))
	n, err := io.CopyN(uplogBuffer, rand.New(rand.NewSource(time.Now().UnixNano())), 1024)
	if err != nil {
		t.Fatal(err)
	} else if n != 1024 {
		t.Fatalf("unexpected n: %d", n)
	}

	writeMemoryBufferToFileBuffer(uplogBuffer.Bytes())
	uplogBuffer.Reset()

	n, err = io.CopyN(uplogBuffer, rand.New(rand.NewSource(time.Now().UnixNano())), 1024)
	if err != nil {
		t.Fatal(err)
	} else if n != 1024 {
		t.Fatalf("unexpected n: %d", n)
	}

	writeMemoryBufferToFileBuffer(uplogBuffer.Bytes())
	uplogBuffer.Reset()

	fi, err := os.Stat(filepath.Join(tmpDir, UPLOG_FILE_BUFFER_NAME))
	if err != nil {
		t.Fatal(err)
	} else if fi.Size() == 0 {
		t.Fatalf("unexpected fi.Size(): %d", fi.Size())
	}
}

func TestUplogArchiveFileBuffer(t *testing.T) {
	testLock.Lock()
	defer testLock.Unlock()

	tmpDir, err := ioutil.TempDir("", "test-uplog-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	defer func() {
		if err := FlushBuffer(); err != nil {
			t.Fatal(err)
		}
	}()

	var called int32
	md5HasherServer := md5.New()
	httpServerMux := http.NewServeMux()
	httpServerMux.Handle("/log/4", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("Unexpected method: %s", r.Method)
		}
		if r.URL.Query().Get("compressed") != "gzip" {
			t.Fatalf("Unexpected compressed: %s", r.URL.Query().Get("compressed"))
		}
		if r.Header.Get("Authorization") != "UpToken fakeuptoken" {
			t.Fatalf("Unexpected Authorization: %s", r.Header.Get("Authorization"))
		}
		if atomic.AddInt32(&called, 1) > 1 {
			if r.Header.Get(X_LOG_CLIENT_ID) != "fake-x-log-client-id" {
				t.Fatalf("Unexpected X-Log-Client-Id: %s", r.Header.Get("X_LOG_CLIENT_ID"))
			}
		}
		w.Header().Add(X_LOG_CLIENT_ID, "fake-x-log-client-id")
		uncompressedBody, err := gzip.NewReader(r.Body)
		if err != nil {
			log.Fatal(err)
		}
		if _, err = io.Copy(md5HasherServer, uncompressedBody); err != nil {
			log.Fatal(err)
		}
	}))
	httpServer := httptest.NewServer(httpServerMux)
	defer httpServer.Close()

	SetUplogUrl(httpServer.URL)
	defer SetUplogUrl("")

	getUpToken = func() (string, error) { return "fakeuptoken", nil }
	defer func() { getUpToken = nil }()

	SetUplogFileBufferDirPath(tmpDir)
	defer SetUplogFileBufferDirPath("")

	DisableUplog()
	defer EnableUplog()

	originalUplogFileBufferThreshold := uplogFileBufferThreshold
	uplogFileBufferThreshold = 24 * 1024
	defer func() {
		uplogFileBufferThreshold = originalUplogFileBufferThreshold
	}()

	uplogBuffer := bytes.NewBuffer(make([]byte, 0, 4*1024))
	md5HasherClient := md5.New()
	r := io.TeeReader(rand.New(rand.NewSource(time.Now().UnixNano())), md5HasherClient)

	for i := 0; i < 4*24; i++ {
		n, err := io.CopyN(uplogBuffer, r, 1024)
		if err != nil {
			t.Fatal(err)
		} else if n != 1024 {
			t.Fatalf("unexpected n: %d", n)
		}

		writeMemoryBufferToFileBuffer(uplogBuffer.Bytes())
		uplogBuffer.Reset()
		time.Sleep(10 * time.Nanosecond)
	}
	tryToArchiveFileBuffer(true)
	time.Sleep(100 * time.Millisecond)
	c := atomic.LoadInt32(&called)
	if c == 0 {
		t.Fatal("unexpected upload count")
	}
	if !bytes.Equal(md5HasherClient.Sum(nil), md5HasherServer.Sum(nil)) {
		t.Fatal("unexpected request body")
	}
	entries, err := ioutil.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 && len(entries) != 3 {
		t.Fatalf("unexpected uplog buffer files count")
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".lock") && entry.Name() != UPLOG_FILE_BUFFER_NAME {
			t.Fatalf("unexpected uplog buffer file: %s", entry.Name())
		}
	}
}

func TestUplogArchiveFileBufferFailed(t *testing.T) {
	testLock.Lock()
	defer testLock.Unlock()

	tmpDir, err := ioutil.TempDir("", "test-uplog-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	defer func() {
		if err := FlushBuffer(); err != nil {
			t.Fatal(err)
		}
	}()

	var called int32
	httpServerMux := http.NewServeMux()
	httpServerMux.Handle("/log/4", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("Unexpected method: %s", r.Method)
		}
		if r.URL.Query().Get("compressed") != "gzip" {
			t.Fatalf("Unexpected compressed: %s", r.URL.Query().Get("compressed"))
		}
		if r.Header.Get("Authorization") != "UpToken fakeuptoken" {
			t.Fatalf("Unexpected Authorization: %s", r.Header.Get("Authorization"))
		}
		if atomic.AddInt32(&called, 1) > 1 {
			if r.Header.Get(X_LOG_CLIENT_ID) != "fake-x-log-client-id" {
				t.Fatalf("Unexpected X-Log-Client-Id: %s", r.Header.Get("X_LOG_CLIENT_ID"))
			}
		}
		w.Header().Add(X_LOG_CLIENT_ID, "fake-x-log-client-id")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	httpServer := httptest.NewServer(httpServerMux)
	defer httpServer.Close()

	SetUplogUrl(httpServer.URL)
	defer SetUplogUrl("")

	getUpToken = func() (string, error) { return "fakeuptoken", nil }
	defer func() { getUpToken = nil }()

	SetUplogFileBufferDirPath(tmpDir)
	defer SetUplogFileBufferDirPath("")

	DisableUplog()
	defer EnableUplog()

	originalUplogMaxStorageBytes := GetUplogMaxStorageBytes()
	SetUplogMaxStorageBytes(48 * 1024)
	defer SetUplogMaxStorageBytes(originalUplogMaxStorageBytes)

	originalUplogFileBufferThreshold := uplogFileBufferThreshold
	uplogFileBufferThreshold = 24 * 1024
	defer func() {
		uplogFileBufferThreshold = originalUplogFileBufferThreshold
	}()

	uplogBuffer := bytes.NewBuffer(make([]byte, 0, 4*1024))
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 4*24; i++ {
		n, err := io.CopyN(uplogBuffer, r, 1024)
		if err != nil {
			t.Fatal(err)
		} else if n != 1024 {
			t.Fatalf("unexpected n: %d", n)
		}

		writeMemoryBufferToFileBuffer(uplogBuffer.Bytes())
		uplogBuffer.Reset()
		time.Sleep(10 * time.Nanosecond)
	}
	tryToArchiveFileBuffer(true)
	time.Sleep(100 * time.Millisecond)
	c := atomic.LoadInt32(&called)
	if c == 0 {
		t.Fatal("unexpected upload count")
	}

	entries, err := ioutil.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	totalSize := uint64(0)
	for _, entry := range entries {
		totalSize += uint64(entry.Size())
	}
	if totalSize > 48*1024 {
		t.Fatalf("unexpected uplog buffer file size: %d", totalSize)
	}
}
