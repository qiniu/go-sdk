//go:build unit
// +build unit

package resumablerecorder_test

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	resumablerecorder "github.com/qiniu/go-sdk/v7/storagev2/downloader/resumable_recorder"
)

func TestJsonFileSystemResumableRecorder(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	options := resumablerecorder.ResumableRecorderOpenArgs{
		ETag:          "testetag1",
		DestinationID: "/tmp/fakeFile",
		PartSize:      16 * 1024 * 1024,
		TotalSize:     100 * 1024 * 1024,
	}
	fs := resumablerecorder.NewJsonFileSystemResumableRecorder(tmpDir)
	writableMedium := fs.OpenForCreatingNew(&options)
	for i := uint64(0); i < 3; i++ {
		if err = writableMedium.Write(&resumablerecorder.ResumableRecord{
			Offset:      i * 16 * 1024 * 1024,
			PartSize:    16 * 1024 * 1024,
			PartWritten: 16 * 1024 * 1024,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err = writableMedium.Close(); err != nil {
		t.Fatal(err)
	}
	writableMedium = fs.OpenForAppending(&options)
	if err = writableMedium.Write(&resumablerecorder.ResumableRecord{
		Offset:      3 * 16 * 1024 * 1024,
		PartSize:    16 * 1024 * 1024,
		PartWritten: 16 * 1024 * 1024,
	}); err != nil {
		t.Fatal(err)
	}
	if err = writableMedium.Close(); err != nil {
		t.Fatal(err)
	}

	options2 := options
	options2.ETag = "testetag2"
	writableMedium = fs.OpenForCreatingNew(&options2)
	for i := uint64(0); i < 4; i++ {
		if err = writableMedium.Write(&resumablerecorder.ResumableRecord{
			Offset:      i * 16 * 1024 * 1024,
			PartSize:    16 * 1024 * 1024,
			PartWritten: 8 * 1024 * 1024,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err = writableMedium.Close(); err != nil {
		t.Fatal(err)
	}

	readableMedium := fs.OpenForReading(&options)
	for i := uint64(0); i < 4; i++ {
		var rr resumablerecorder.ResumableRecord

		if err = readableMedium.Next(&rr); err != nil {
			t.Fatal(err)
		}

		if rr.Offset != i*16*1024*1024 {
			t.Fatalf("unexpected offset: %d", rr.Offset)
		}
		if rr.PartSize != 16*1024*1024 {
			t.Fatalf("unexpected partSize: %d", rr.PartSize)
		}
		if rr.PartWritten != 16*1024*1024 {
			t.Fatalf("unexpected partWritten: %d", rr.PartWritten)
		}
	}
	if err = readableMedium.Close(); err != nil {
		t.Fatal(err)
	}

	time.Sleep(11 * time.Second)
	if err = fs.ClearOutdated(10 * time.Second); err != nil {
		t.Fatal(err)
	}

	readableMedium = fs.OpenForReading(&options)
	if readableMedium != nil {
		t.Fatalf("unexpected readable medium")
	}

	readableMedium = fs.OpenForReading(&options2)
	if readableMedium != nil {
		t.Fatalf("unexpected readable medium")
	}
}
