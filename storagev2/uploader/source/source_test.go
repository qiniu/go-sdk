//go:build unit
// +build unit

package source_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
	uploader "github.com/qiniu/go-sdk/v7/storagev2/uploader/source"
)

func TestSeekableSource(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "test-seekable-source-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err = io.CopyN(tmpFile, rand.New(rand.NewSource(time.Now().UnixNano())), 4096); err != nil {
		t.Fatal(err)
	}
	if _, err = tmpFile.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}

	source := uploader.NewReadSeekCloserSource(tmpFile, tmpFile.Name())
	testSource(t, source, tmpFile)
	source = uploader.NewReadAtSeekCloserSource(tmpFile, tmpFile.Name())
	testSource(t, source, tmpFile)
}

func testSource(t *testing.T, source uploader.Source, originalFile *os.File) {
	if ts, err := source.(uploader.SizedSource).TotalSize(); err != nil {
		t.Fatal(err)
	} else if ts != 4096 {
		t.Fatalf("Unexpected file size: %d", ts)
	}

	if sk, err := source.SourceKey(); err != nil {
		t.Fatal(err)
	} else if sk != originalFile.Name() {
		t.Fatalf("Unexpected source key: %#v", sk)
	}

	parts := make([]uploader.Part, 0, 16)
	for i := 0; i < 16; i++ {
		part, err := source.Slice(256)
		if err != nil {
			t.Fatal(err)
		}
		parts = append(parts, part)
	}
	for i, part := range parts {
		if part.PartNumber() != uint64(i+1) {
			t.Fatalf("Unexpected part number: %d", part.PartNumber())
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			testPart(t, parts[i], int64(i*256), originalFile)
		}(i)
	}
	wg.Wait()
}

func testPart(t *testing.T, part uploader.Part, offset int64, originalFile *os.File) {
	partData, err := internal_io.ReadAll(part)
	if err != nil {
		t.Fatal(err)
	}
	assertReaderEqual(t, originalFile, offset, partData)

	if _, err := part.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}

	partData, err = internal_io.ReadAll(part)
	if err != nil {
		t.Fatal(err)
	}
	assertReaderEqual(t, originalFile, offset, partData)
}

func assertReaderEqual(t *testing.T, file *os.File, offset int64, expectedData []byte) {
	data := make([]byte, len(expectedData))
	n, err := file.ReadAt(data, offset)
	if err != nil {
		t.Fatal(err)
	} else if n != len(expectedData) {
		t.Fatalf("Unexpected read data size %d", n)
	}
	if !bytes.Equal(data, expectedData) {
		t.Fatalf("Range (%d-%d) of file (%s) is inequal", offset, offset+int64(len(expectedData)), file.Name())
	}
}
