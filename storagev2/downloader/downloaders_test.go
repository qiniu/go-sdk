package downloader_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/backoff"
	"github.com/qiniu/go-sdk/v7/storagev2/chooser"
	"github.com/qiniu/go-sdk/v7/storagev2/downloader"
	"github.com/qiniu/go-sdk/v7/storagev2/downloader/destination"
	resumablerecorder "github.com/qiniu/go-sdk/v7/storagev2/downloader/resumable_recorder"
	"github.com/qiniu/go-sdk/v7/storagev2/resolver"
)

func TestConcurrentDownloaderWithSinglePart(t *testing.T) {
	var (
		counts [3]uint64
		hasher = md5.New()
	)
	handler := func(id int, w http.ResponseWriter, r *http.Request) {
		counts[id-1] += 1
		switch id {
		case 1:
			w.WriteHeader(http.StatusGatewayTimeout)
		case 2:
			w.WriteHeader(http.StatusServiceUnavailable)
		case 3:
			switch r.Method {
			case http.MethodHead:
				w.Header().Set("Etag", "testetag1")
				w.Header().Set("Content-Length", strconv.Itoa(1024*1024))
			case http.MethodGet:
				w.Header().Set("Etag", "testetag1")
				w.Header().Set("Content-Length", strconv.Itoa(1024*1024))
				_, err := io.Copy(w, io.TeeReader(io.LimitReader(rand.New(rand.NewSource(time.Now().UnixNano())), 1024*1024), hasher))
				if err != nil {
					t.Fatal(err)
				}
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		default:
			w.WriteHeader(http.StatusPaymentRequired)
		}
	}
	server1 := newTestServer(1, handler)
	defer server1.Close()
	url1, err := url.Parse(server1.URL)
	if err != nil {
		t.Fatal(err)
	}
	url1.Path = "/testfile"

	server2 := newTestServer(2, handler)
	defer server2.Close()
	url2, err := url.Parse(server2.URL)
	if err != nil {
		t.Fatal(err)
	}
	url2.Path = "/testfile"

	server3 := newTestServer(3, handler)
	defer server3.Close()
	url3, err := url.Parse(server3.URL)
	if err != nil {
		t.Fatal(err)
	}
	url3.Path = "/testfile"

	d := downloader.NewConcurrentDownloader(&downloader.ConcurrentDownloaderOptions{
		Concurrency: 1,
		PartSize:    4 * 1024 * 1024,
		DownloaderOptions: downloader.DownloaderOptions{
			Backoff:  backoff.NewFixedBackoff(0),
			Resolver: resolver.NewDefaultResolver(),
			Chooser:  chooser.NewDirectChooser(),
		},
	})
	var (
		buf            closableBuffer
		lastDownloaded uint64
	)
	n, err := d.Download(
		context.Background(),
		[]downloader.URLProvider{
			downloader.NewURLProvider(url1),
			downloader.NewURLProvider(url2),
			downloader.NewURLProvider(url3),
		}, destination.NewWriteCloserDestination(&buf, ""),
		&downloader.DestinationDownloadOptions{
			OnDownloadingProgress: func(downloaded, totalSize uint64) {
				if downloaded < lastDownloaded {
					t.Fatalf("unexpected downloaded progress")
				}
				lastDownloaded = downloaded
				if totalSize != 1024*1024 {
					t.Fatalf("unexpected downloaded progress")
				}
			},
		})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1024*1024 {
		t.Fatalf("unexpected downloaded size")
	}
	if lastDownloaded != 1024*1024 {
		t.Fatalf("unexpected downloaded progress")
	}
	if counts[0] != 20 {
		t.Fatalf("unexpected called count")
	}
	if counts[1] != 20 {
		t.Fatalf("unexpected called count")
	}
	if counts[2] != 2 {
		t.Fatalf("unexpected called count")
	}
	serverMD5 := hasher.Sum(nil)
	clientMD5 := md5.Sum(buf.Bytes())
	if !bytes.Equal(serverMD5, clientMD5[:]) {
		t.Fatalf("unexpected hash")
	}
}

func TestConcurrentDownloaderWithCompression(t *testing.T) {
	var counts uint64
	hasher := md5.New()
	handler := func(id int, w http.ResponseWriter, r *http.Request) {
		counts += 1
		switch id {
		case 1:
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				t.Fatalf("unexpected accept-encoding")
			}
			switch r.Method {
			case http.MethodHead:
				w.Header().Set("Etag", "testetag1.gz")
				w.Header().Set("Content-Encoding", "gzip")
			case http.MethodGet:
				w.Header().Set("Etag", "testetag1.gz")
				w.Header().Set("Content-Encoding", "gzip")
				var (
					r   = io.TeeReader(io.LimitReader(rand.New(rand.NewSource(time.Now().UnixNano())), 1024*1024), hasher)
					err error
				)
				gw := gzip.NewWriter(w)
				if _, err = io.Copy(gw, r); err != nil {
					t.Fatal(err)
				}
				if err = gw.Close(); err != nil {
					t.Fatal(err)
				}
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		default:
			w.WriteHeader(http.StatusPaymentRequired)
		}
	}
	server1 := newTestServer(1, handler)
	defer server1.Close()
	url1, err := url.Parse(server1.URL)
	if err != nil {
		t.Fatal(err)
	}
	url1.Path = "/testfile"
	d := downloader.NewConcurrentDownloader(&downloader.ConcurrentDownloaderOptions{
		Concurrency: 1,
		PartSize:    4 * 1024 * 1024,
		DownloaderOptions: downloader.DownloaderOptions{
			Backoff:  backoff.NewFixedBackoff(0),
			Resolver: resolver.NewDefaultResolver(),
			Chooser:  chooser.NewDirectChooser(),
		},
	})
	var (
		buf            closableBuffer
		lastDownloaded uint64
	)
	n, err := d.Download(
		context.Background(),
		[]downloader.URLProvider{downloader.NewURLProvider(url1)},
		destination.NewWriteCloserDestination(&buf, ""), &downloader.DestinationDownloadOptions{
			OnDownloadingProgress: func(downloaded, totalSize uint64) {
				if downloaded < lastDownloaded {
					t.Fatalf("unexpected downloaded progress")
				}
				lastDownloaded = downloaded
				if totalSize != 0 {
					t.Fatalf("unexpected downloaded progress")
				}
			},
		})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1024*1024 {
		t.Fatalf("unexpected downloaded size")
	}
	if lastDownloaded != 1024*1024 {
		t.Fatalf("unexpected downloaded progress")
	}
	if counts != 2 {
		t.Fatalf("unexpected called count")
	}
	serverMD5 := hasher.Sum(nil)
	clientMD5 := md5.Sum(buf.Bytes())
	if !bytes.Equal(serverMD5, clientMD5[:]) {
		t.Fatalf("unexpected hash")
	}
}

func TestConcurrentDownloaderWithMultipleParts(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	srcFile, err := ioutil.TempFile(tmpDir, "testfile")
	if err != nil {
		t.Fatal(err)
	}
	defer srcFile.Close()

	dstFile, err := ioutil.TempFile(tmpDir, "testfile2")
	if err != nil {
		t.Fatal(err)
	}
	defer dstFile.Close()

	const SIZE = 1024 * 1024 * 127
	if _, err = io.CopyN(srcFile, rand.New(rand.NewSource(time.Now().UnixNano())), SIZE); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	handler := http.FileServer(http.Dir(tmpDir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Etag", "testetag1")
		handler.ServeHTTP(w, r)
	})
	server1 := httptest.NewServer(mux)
	defer server1.Close()
	url1, err := url.Parse(server1.URL)
	if err != nil {
		t.Fatal(err)
	}
	url1.Path = "/" + filepath.Base(srcFile.Name())
	d := downloader.NewConcurrentDownloader(&downloader.ConcurrentDownloaderOptions{
		Concurrency: 16,
		PartSize:    4 * 1024 * 1024,
		DownloaderOptions: downloader.DownloaderOptions{
			Backoff:  backoff.NewFixedBackoff(0),
			Resolver: resolver.NewDefaultResolver(),
			Chooser:  chooser.NewDirectChooser(),
		},
	})
	dest, err := destination.NewFileDestination(dstFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	var lastDownloaded uint64
	n, err := d.Download(
		context.Background(),
		[]downloader.URLProvider{downloader.NewURLProvider(url1)},
		dest,
		&downloader.DestinationDownloadOptions{
			OnDownloadingProgress: func(downloaded, totalSize uint64) {
				if downloaded < atomic.LoadUint64(&lastDownloaded) {
					t.Fatalf("unexpected downloaded progress")
				}
				atomic.StoreUint64(&lastDownloaded, downloaded)
				if totalSize != SIZE {
					t.Fatalf("unexpected downloaded progress")
				}
			},
		})
	if err != nil {
		t.Fatal(err)
	}
	if n != SIZE {
		t.Fatalf("unexpected downloaded size")
	}
	if lastDownloaded != SIZE {
		t.Fatalf("unexpected downloaded progress")
	}
	hasher := md5.New()
	if _, err = srcFile.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	if _, err = io.Copy(hasher, srcFile); err != nil {
		t.Fatal(err)
	}
	serverMD5 := hasher.Sum(nil)
	hasher.Reset()
	if _, err = dstFile.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	if _, err = io.Copy(hasher, dstFile); err != nil {
		t.Fatal(err)
	}
	clientMD5 := hasher.Sum(nil)
	if !bytes.Equal(serverMD5, clientMD5) {
		t.Fatalf("unexpected hash")
	}
}

func TestConcurrentDownloaderWithResumableRecorder(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	srcFile, err := ioutil.TempFile(tmpDir, "srcFile")
	if err != nil {
		t.Fatal(err)
	}
	defer srcFile.Close()

	dstFile, err := ioutil.TempFile(tmpDir, "dstFile")
	if err != nil {
		t.Fatal(err)
	}
	defer dstFile.Close()

	if _, err = io.CopyN(srcFile, rand.New(rand.NewSource(time.Now().UnixNano())), 10*1024*1024); err != nil {
		t.Fatal(err)
	}

	ranges := make(map[uint64]uint64)
	var rangesMutex sync.Mutex
	handler := func(id int, w http.ResponseWriter, r *http.Request) {
		switch id {
		case 1:
			switch r.Method {
			case http.MethodHead:
				w.Header().Set("Etag", "testetag1")
				w.Header().Set("Content-Length", strconv.Itoa(10*1024*1024))
			case http.MethodGet:
				w.Header().Set("Etag", "testetag1")

				var fromOffset, toOffset uint64
				if _, err = fmt.Sscanf(r.Header.Get("Range"), "bytes=%d-%d", &fromOffset, &toOffset); err != nil {
					t.Fatal(err)
				}
				rangesMutex.Lock()
				ranges[fromOffset] = toOffset - fromOffset + 1
				rangesMutex.Unlock()

				w.Header().Set("Content-Length", strconv.FormatUint(toOffset-fromOffset+1, 10))
				w.WriteHeader(http.StatusPartialContent)
				if _, err = io.Copy(w, io.NewSectionReader(srcFile, int64(fromOffset), int64(toOffset-fromOffset+1))); err != nil {
					t.Fatal(err)
				}
			}
		default:
			w.WriteHeader(http.StatusPaymentRequired)
		}
	}
	server1 := newTestServer(1, handler)
	defer server1.Close()
	url1, err := url.Parse(server1.URL)
	if err != nil {
		t.Fatal(err)
	}
	url1.Path = "/testfile"

	resumableRecorder := resumablerecorder.NewJsonFileSystemResumableRecorder(tmpDir)
	options := resumablerecorder.ResumableRecorderOpenOptions{
		ETag:           "testetag1",
		DestinationKey: dstFile.Name(),
		PartSize:       1024 * 1024,
		TotalSize:      10 * 1024 * 1024,
	}
	writableMedium := resumableRecorder.OpenForCreatingNew(&options)
	defer writableMedium.Close()

	if err = writableMedium.Write(&resumablerecorder.ResumableRecord{
		Offset:      0,
		PartSize:    1024 * 1024,
		PartWritten: 1024,
	}); err != nil {
		t.Fatal(err)
	}
	if err = writableMedium.Write(&resumablerecorder.ResumableRecord{
		Offset:      1024 * 1024,
		PartSize:    1024 * 1024,
		PartWritten: 1024 * 1024,
	}); err != nil {
		t.Fatal(err)
	}
	if err = writableMedium.Close(); err != nil {
		t.Fatal(err)
	}

	d := downloader.NewConcurrentDownloader(&downloader.ConcurrentDownloaderOptions{
		Concurrency:       10,
		PartSize:          1024 * 1024,
		ResumableRecorder: resumableRecorder,
		DownloaderOptions: downloader.DownloaderOptions{
			Backoff:  backoff.NewFixedBackoff(0),
			Resolver: resolver.NewDefaultResolver(),
			Chooser:  chooser.NewDirectChooser(),
		},
	})
	dest, err := destination.NewFileDestination(dstFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	var lastDownloaded uint64
	n, err := d.Download(
		context.Background(),
		[]downloader.URLProvider{downloader.NewURLProvider(url1)},
		dest,
		&downloader.DestinationDownloadOptions{
			OnDownloadingProgress: func(downloaded, totalSize uint64) {
				if downloaded < lastDownloaded {
					t.Fatalf("unexpected downloaded progress")
				}
				lastDownloaded = downloaded
				if totalSize != 10*1024*1024 {
					t.Fatalf("unexpected downloaded progress")
				}
			},
		})
	if err != nil {
		t.Fatal(err)
	}
	if n != 10*1024*1024 {
		t.Fatal(err)
	}
	if lastDownloaded != 10*1024*1024 {
		t.Fatalf("unexpected downloaded progress")
	}
	if len(ranges) != 9 {
		t.Fatalf("unexpected ranges")
	}
	if ranges[1024] != 1024*1023 {
		t.Fatalf("unexpected ranges")
	}
	if ranges[1024*1024] != 0 {
		t.Fatalf("unexpected ranges")
	}

	readableMedium := resumableRecorder.OpenForReading(&options)
	if readableMedium != nil {
		t.Fatalf("medium is not expected to be found")
	}
}

func newTestServer(id int, handler func(int, http.ResponseWriter, *http.Request)) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/testfile", func(w http.ResponseWriter, r *http.Request) {
		handler(id, w, r)
	})
	return httptest.NewServer(mux)
}

type closableBuffer struct {
	bytes.Buffer
}

func (w closableBuffer) Close() error {
	return nil
}
