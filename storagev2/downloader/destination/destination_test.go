//go:build unit
// +build unit

package destination_test

import (
	"bytes"
	"crypto/md5"
	"errors"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/downloader/destination"
)

func TestSeekableDestination(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "test-seekable-destination-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	dest := destination.NewWriteAtCloserDestination(tmpFile, tmpFile.Name())
	parts, err := dest.Slice(1024*1024*1024, 1024*1024, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 1024 {
		t.Fatalf("unexpected slices")
	}

	var (
		wg   sync.WaitGroup
		lock sync.Mutex
	)

	for i, part := range parts {
		wg.Add(1)
		go func(i int, part destination.Part) {
			defer wg.Done()

			buf := make([]byte, 1024*1024)
			for j := 0; j < 1024*1024; j++ {
				buf[j] = byte(i % 256)
			}
			var lastDownloaded uint64
			if _, e := part.CopyFrom(bytes.NewReader(buf), func(downloaded uint64) {
				if lastDownloaded > downloaded {
					lock.Lock()
					err = errors.New("unexpected downloading progress")
					lock.Unlock()
				}
				lastDownloaded = downloaded
			}); e != nil {
				lock.Lock()
				err = e
				lock.Unlock()
			}
		}(i, part)
	}
	wg.Wait()

	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 1024; i++ {
		buf := make([]byte, 1024*1024)
		_, err := tmpFile.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		for j := 0; j < len(buf); j++ {
			if buf[j] != byte(i%256) {
				t.Fatalf("unexpected buffer content")
			}
		}
	}
}

func TestUnseekableDestination(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "test-unseekable-destination-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	dest := destination.NewWriteCloserDestination(tmpFile, tmpFile.Name())
	parts, err := dest.Slice(1024*1024, 1024, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 1 {
		t.Fatalf("unexpected slices")
	}

	md5Expected := md5.New()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var lastDownloaded uint64

	if _, err = parts[0].CopyFrom(io.TeeReader(io.LimitReader(r, 1024*1024), md5Expected), func(downloaded uint64) {
		if lastDownloaded > downloaded {
			t.Fatalf("unexpected downloading progress")
		}
		lastDownloaded = downloaded
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = tmpFile.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}

	md5Actual := md5.New()
	if _, err = io.Copy(md5Actual, tmpFile); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(md5Expected.Sum(nil), md5Actual.Sum(nil)) {
		t.Fatalf("unexpected md5 checksum")
	}
	if lastDownloaded != 1024*1024 {
		t.Fatalf("unexpected downloading progress")
	}
}
