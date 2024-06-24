//go:build unit
// +build unit

package resumablerecorder_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/region"
	resumablerecorder "github.com/qiniu/go-sdk/v7/storagev2/uploader/resumable_recorder"
)

func TestJsonFileSystemResumableRecorder(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	options := resumablerecorder.ResumableRecorderOpenOptions{
		AccessKey:  "testak",
		BucketName: "test-bucket",
		ObjectName: "test-object",
		SourceKey:  "/tmp/fakeFile",
		PartSize:   4 * 1024 * 1024,
		TotalSize:  100 * 1024 * 1024,
		UpEndpoints: region.Endpoints{
			Preferred:   []string{"https://uc.qiniuapi.com", "https://kodo-config.qiniuapi.com"},
			Alternative: []string{"https://uc.qbox.me"},
		},
	}
	fs := resumablerecorder.NewJsonFileSystemResumableRecorder(tmpDir)
	writableMedium := fs.OpenForCreatingNew(&options)
	for i := uint64(0); i < 3; i++ {
		if err = writableMedium.Write(&resumablerecorder.ResumableRecord{
			UploadId:   "test-upload-id",
			PartId:     fmt.Sprintf("test-part-%d", i+1),
			Offset:     i * 4 * 1024 * 1024,
			PartNumber: i + 1,
			ExpiredAt:  time.Now().Add(10 * time.Second),
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err = writableMedium.Close(); err != nil {
		t.Fatal(err)
	}
	writableMedium = fs.OpenForAppending(&options)
	if err = writableMedium.Write(&resumablerecorder.ResumableRecord{
		UploadId:   "test-upload-id",
		PartId:     fmt.Sprintf("test-part-%d", 3+1),
		Offset:     3 * 4 * 1024 * 1024,
		PartNumber: 3 + 1,
		ExpiredAt:  time.Now().Add(10 * time.Second),
	}); err != nil {
		t.Fatal(err)
	}
	if err = writableMedium.Close(); err != nil {
		t.Fatal(err)
	}

	options2 := options
	options2.ObjectName = "test-object-2"
	writableMedium = fs.OpenForCreatingNew(&options2)
	for i := uint64(0); i < 4; i++ {
		if err = writableMedium.Write(&resumablerecorder.ResumableRecord{
			UploadId:   "test-upload-id-2",
			PartId:     fmt.Sprintf("test-part-%d", i+1),
			Offset:     i * 4 * 1024 * 1024,
			PartNumber: i + 1,
			ExpiredAt:  time.Now().Add(10 * time.Second),
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

		if rr.UploadId != "test-upload-id" {
			t.Fatalf("unexpected uploadId: %s", rr.UploadId)
		}
		if rr.PartId != fmt.Sprintf("test-part-%d", i+1) {
			t.Fatalf("unexpected partId: %s", rr.PartId)
		}
		if rr.Offset != i*4*1024*1024 {
			t.Fatalf("unexpected offset: %d", rr.Offset)
		}
		if rr.PartNumber != i+1 {
			t.Fatalf("unexpected partNumber: %d", rr.PartNumber)
		}
	}
	if err = readableMedium.Close(); err != nil {
		t.Fatal(err)
	}

	time.Sleep(11 * time.Second)
	if err = fs.ClearExpired(); err != nil {
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
