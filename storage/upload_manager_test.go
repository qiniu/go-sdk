//go:build integration
// +build integration

package storage

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

	clientV1 "github.com/qiniu/go-sdk/v7/client"
)

func getUploadManager() *UploadManager {
	region01 := &Region{
		SrcUpHosts: []string{"mock01.qiniu.com", "mock02.qiniu.com"},
	}

	region02, err := GetRegion(testAK, testBucket)
	if err != nil {
		return nil
	}

	regionGroup := NewRegionGroup(region01, region02)
	return NewUploadManager(&UploadConfig{
		UseHTTPS:      true,
		UseCdnDomains: false,
		Regions:       regionGroup,
	})
}

func getUploadToken() string {
	putPolicy := PutPolicy{
		Scope:           testBucket,
		DeleteAfterDays: 7,
	}
	return putPolicy.UploadToken(mac)
}

type Ret struct {
	UploadRet

	Foo string `json:"x:foo"`
}

func TestUploadManagerFormUpload(t *testing.T) {
	clientV1.DebugMode = true
	defer func() {
		clientV1.DebugMode = false
	}()

	data := []byte("hello, 七牛！！！")
	tempFile, err := ioutil.TempFile("", "TestUploadManagerFormPut")
	if err != nil {
		t.Fatalf("create temp file error:%v", err)
	}
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()
	tempFile.Write(data)
	size := int64(len(data))

	params := make(map[string]string)
	params["x:foo"] = "foo"
	params["x-qn-meta-A"] = "meta-A"

	uploadManager := getUploadManager()
	var ret Ret

	// 上传 file
	source, err := NewUploadSourceFile(tempFile.Name())
	err = uploadManager.Put(context.Background(), &ret, getUploadToken(), nil, source, &UploadExtra{
		Params:             params,
		TryTimes:           1,
		HostFreezeDuration: 0,
		MimeType:           "",
		OnProgress: func(fileSize, uploaded int64) {
			fmt.Printf("form upload file progress: %d-%d \n", uploaded, fileSize)
		},
		UploadResumeVersion: 0,
		UploadThreshold:     0,
		Recorder:            nil,
		PartSize:            0,
	})
	if err != nil {
		t.Fatalf("form upload file error:%v", err)
	}
	if len(ret.Key) == 0 || len(ret.Hash) == 0 {
		t.Fatal("form upload file error, key or hash is empty")
	}
	if len(ret.Foo) == 0 {
		t.Fatal("form upload file error, foo is empty")
	}

	// 上传 reader no size
	tempFile.Seek(0, io.SeekStart)
	source, _ = NewUploadSourceReader(tempFile, size)
	err = uploadManager.Put(context.Background(), &ret, getUploadToken(), nil, source, &UploadExtra{
		Params:             params,
		TryTimes:           1,
		HostFreezeDuration: 0,
		MimeType:           "",
		OnProgress: func(fileSize, uploaded int64) {
			fmt.Printf("form upload reader progress: %d-%d \n", uploaded, fileSize)
		},
		UploadResumeVersion: 0,
		UploadThreshold:     0,
		Recorder:            nil,
		PartSize:            0,
	})
	if err == nil {
		t.Fatal("form upload: reader source should not support region backup")
	}

	// 上传 readerAt
	tempFile.Seek(0, io.SeekStart)
	source, _ = NewUploadSourceReaderAt(tempFile, size)
	err = uploadManager.Put(context.Background(), &ret, getUploadToken(), nil, source, &UploadExtra{
		Params:             params,
		TryTimes:           1,
		HostFreezeDuration: 0,
		MimeType:           "",
		OnProgress: func(fileSize, uploaded int64) {
			fmt.Printf("form upload reader at progress: %d-%d \n", uploaded, fileSize)
		},
		UploadResumeVersion: 0,
		UploadThreshold:     0,
		Recorder:            nil,
		PartSize:            0,
	})
	if err != nil {
		t.Fatalf("form upload reader at error:%v", err)
	}
	if len(ret.Key) == 0 || len(ret.Hash) == 0 {
		t.Fatal("form upload reader at error, key or hash is empty")
	}
	if len(ret.Foo) == 0 {
		t.Fatal("form upload reader at error, foo is empty")
	}
}

func TestUploadManagerResumeV1Upload(t *testing.T) {
	length := 1024 * 1024 * 4
	data := make([]byte, length, length)
	data[0] = 8
	data[length-1] = 8
	tempFile, err := ioutil.TempFile("", "TestUploadManagerFormPut")
	if err != nil {
		t.Fatalf("create temp file error:%v", err)
	}
	defer func() {
		tempFile.Close()
		os.ReadFile(tempFile.Name())
	}()
	tempFile.Write(data)
	size := int64(len(data))

	params := make(map[string]string)
	params["x:foo"] = "foo"
	params["x-qn-meta-A"] = "meta-A"

	uploadManager := getUploadManager()
	var ret Ret

	// 上传 file
	source, err := NewUploadSourceFile(tempFile.Name())
	err = uploadManager.Put(context.Background(), &ret, getUploadToken(), nil, source, &UploadExtra{
		Params:             params,
		TryTimes:           1,
		HostFreezeDuration: 0,
		MimeType:           "",
		OnProgress: func(fileSize, uploaded int64) {
			fmt.Printf("resume v1: upload file progress: %d-%d \n", uploaded, fileSize)
		},
		UploadResumeVersion: UploadResumeV1,
		UploadThreshold:     1024 * 1024 * 2,
		Recorder:            nil,
		PartSize:            1024 * 1024,
	})
	if err != nil {
		t.Fatalf("form upload file error:%v", err)
	}
	if len(ret.Key) == 0 || len(ret.Hash) == 0 {
		t.Fatal("form upload file error, key or hash is empty")
	}
	if len(ret.Foo) == 0 {
		t.Fatal("form upload file error, foo is empty")
	}

	// 上传 reader no size
	tempFile.Seek(0, io.SeekStart)
	source, _ = NewUploadSourceReader(tempFile, size)
	err = uploadManager.Put(context.Background(), &ret, getUploadToken(), nil, source, &UploadExtra{
		Params:             params,
		TryTimes:           1,
		HostFreezeDuration: 0,
		MimeType:           "",
		OnProgress: func(fileSize, uploaded int64) {
			fmt.Printf("resume v1: upload reader progress: %d-%d \n", uploaded, fileSize)
		},
		UploadResumeVersion: UploadResumeV1,
		UploadThreshold:     1024 * 1024 * 2,
		Recorder:            nil,
		PartSize:            1024 * 1024,
	})
	if err == nil {
		t.Fatal("resume v1:: reader source should not support region backup")
	}

	// 上传 readerAt
	tempFile.Seek(0, io.SeekStart)
	source, _ = NewUploadSourceReaderAt(tempFile, size)
	err = uploadManager.Put(context.Background(), &ret, getUploadToken(), nil, source, &UploadExtra{
		Params:             params,
		TryTimes:           1,
		HostFreezeDuration: 0,
		MimeType:           "",
		OnProgress: func(fileSize, uploaded int64) {
			fmt.Printf("resume v1: upload reader at progress: %d-%d \n", uploaded, fileSize)
		},
		UploadResumeVersion: UploadResumeV1,
		UploadThreshold:     1024 * 1024 * 2,
		Recorder:            nil,
		PartSize:            1024 * 1024,
	})
	if err != nil {
		t.Fatalf("resume v1: upload reader at error:%v", err)
	}
	if len(ret.Key) == 0 || len(ret.Hash) == 0 {
		t.Fatal("resume v1: upload reader at error, key or hash is empty")
	}
	if len(ret.Foo) == 0 {
		t.Fatal("resume v1: upload reader at error, foo is empty")
	}
}

func TestUploadManagerResumeV1UploadRecord(t *testing.T) {
	// 文件比较小，小并发方便取消
	settings.Workers = 2

	length := 1024 * 1024 * 9
	data := make([]byte, length, length)
	data[0] = 8
	data[length-1] = 8
	tempFile, err := ioutil.TempFile("", "TestUploadManagerFormPut")
	if err != nil {
		t.Fatalf("create temp file error:%v", err)
	}
	defer func() {
		tempFile.Close()
		os.ReadFile(tempFile.Name())
	}()
	tempFile.Write(data)

	params := make(map[string]string)
	params["x:foo"] = "foo"
	params["x-qn-meta-A"] = "meta-A"

	uploadManager := getUploadManager()
	var ret Ret

	ctx, cancel := context.WithCancel(context.Background())

	recorder, err := NewFileRecorder(os.TempDir())
	if err != nil {
		t.Fatalf("create recorder error:%v", err)
	}
	uploadedSizeWhileCancel := int64(1024 * 1024 * 2)
	// 上传 file
	source, err := NewUploadSourceFile(tempFile.Name())
	err = uploadManager.Put(ctx, &ret, getUploadToken(), nil, source, &UploadExtra{
		Params:             params,
		TryTimes:           1,
		HostFreezeDuration: 0,
		MimeType:           "",
		OnProgress: func(fileSize, uploaded int64) {
			fmt.Printf("resume v1 record 01: upload file progress: %d-%d \n", uploaded, fileSize)
			if uploaded >= uploadedSizeWhileCancel {
				cancel()
			}
		},
		UploadResumeVersion: UploadResumeV1,
		UploadThreshold:     1024 * 1024 * 2,
		Recorder:            recorder,
		PartSize:            1024 * 1024,
	})
	if !isCancelErr(err) {
		t.Fatalf("resume upload file with record error:%v", err)
	}

	// 再次上传 file
	err = uploadManager.Put(context.Background(), &ret, getUploadToken(), nil, source, &UploadExtra{
		Params:             params,
		TryTimes:           1,
		HostFreezeDuration: 0,
		MimeType:           "",
		OnProgress: func(fileSize, uploaded int64) {
			fmt.Printf("resume v1 record 02: upload file progress: %d-%d \n", uploaded, fileSize)
			if uploaded < uploadedSizeWhileCancel {
				t.Fatalf("resume upload file with record error: uploaded size should bigger than %d", uploadedSizeWhileCancel)
			}
		},
		UploadResumeVersion: UploadResumeV1,
		UploadThreshold:     1024 * 1024 * 2,
		Recorder:            recorder,
		PartSize:            1024 * 1024,
	})

	if err != nil {
		t.Fatalf("resume v1 upload file with record error:%v", err)
	}
	if len(ret.Key) == 0 || len(ret.Hash) == 0 {
		t.Fatal("resume v1 upload file with record error, key or hash is empty")
	}
	if len(ret.Foo) == 0 {
		t.Fatal("resume v1 upload file with record error, foo is empty")
	}
}

func TestUploadManagerResumeV2Upload(t *testing.T) {
	length := 1024 * 1024 * 4
	data := make([]byte, length, length)
	data[0] = 8
	data[length-1] = 8
	tempFile, err := ioutil.TempFile("", "TestUploadManagerFormPut")
	if err != nil {
		t.Fatalf("create temp file error:%v", err)
	}
	defer func() {
		tempFile.Close()
		os.ReadFile(tempFile.Name())
	}()
	tempFile.Write(data)
	size := int64(len(data))

	params := make(map[string]string)
	params["x:foo"] = "foo"
	params["x-qn-meta-A"] = "meta-A"

	uploadManager := getUploadManager()
	var ret Ret

	// 上传 file
	source, err := NewUploadSourceFile(tempFile.Name())
	err = uploadManager.Put(context.Background(), &ret, getUploadToken(), nil, source, &UploadExtra{
		Params:             params,
		TryTimes:           1,
		HostFreezeDuration: 0,
		MimeType:           "",
		OnProgress: func(fileSize, uploaded int64) {
			fmt.Printf("resume v2: upload file progress: %d-%d \n", uploaded, fileSize)
		},
		UploadResumeVersion: UploadResumeV2,
		UploadThreshold:     1024 * 1024 * 2,
		Recorder:            nil,
		PartSize:            1024 * 1024,
	})
	if err != nil {
		t.Fatalf("form upload file error:%v", err)
	}
	if len(ret.Key) == 0 || len(ret.Hash) == 0 {
		t.Fatal("form upload file error, key or hash is empty")
	}
	if len(ret.Foo) == 0 {
		t.Fatal("form upload file error, foo is empty")
	}

	// 上传 reader no size
	tempFile.Seek(0, io.SeekStart)
	source, _ = NewUploadSourceReader(tempFile, size)
	err = uploadManager.Put(context.Background(), &ret, getUploadToken(), nil, source, &UploadExtra{
		Params:             params,
		TryTimes:           1,
		HostFreezeDuration: 0,
		MimeType:           "",
		OnProgress: func(fileSize, uploaded int64) {
			fmt.Printf("resume v2: upload reader progress: %d-%d \n", uploaded, fileSize)
		},
		UploadResumeVersion: UploadResumeV2,
		UploadThreshold:     1024 * 1024 * 2,
		Recorder:            nil,
		PartSize:            1024 * 1024,
	})
	if err == nil {
		t.Fatal("resume v2:: reader source should not support region backup")
	}

	// 上传 readerAt
	tempFile.Seek(0, io.SeekStart)
	source, _ = NewUploadSourceReaderAt(tempFile, size)
	err = uploadManager.Put(context.Background(), &ret, getUploadToken(), nil, source, &UploadExtra{
		Params:             params,
		TryTimes:           1,
		HostFreezeDuration: 0,
		MimeType:           "",
		OnProgress: func(fileSize, uploaded int64) {
			fmt.Printf("resume v2: upload reader at progress: %d-%d \n", uploaded, fileSize)
		},
		UploadResumeVersion: UploadResumeV2,
		UploadThreshold:     1024 * 1024 * 2,
		Recorder:            nil,
		PartSize:            1024 * 1024,
	})
	if err != nil {
		t.Fatalf("resume v2: upload reader at error:%v", err)
	}
	if len(ret.Key) == 0 || len(ret.Hash) == 0 {
		t.Fatal("resume v2: upload reader at error, key or hash is empty")
	}
	if len(ret.Foo) == 0 {
		t.Fatal("resume v2: upload reader at error, foo is empty")
	}
}

func TestUploadManagerResumeV2UploadRecord(t *testing.T) {
	// 文件比较小，小并发方便取消
	settings.Workers = 2

	length := 1024 * 1024 * 9
	data := make([]byte, length, length)
	data[0] = 8
	data[length-1] = 8
	tempFile, err := ioutil.TempFile("", "TestUploadManagerFormPut")
	if err != nil {
		t.Fatalf("create temp file error:%v", err)
	}
	defer func() {
		tempFile.Close()
		os.ReadFile(tempFile.Name())
	}()
	tempFile.Write(data)

	params := make(map[string]string)
	params["x:foo"] = "foo"
	params["x-qn-meta-A"] = "meta-A"

	uploadManager := getUploadManager()
	var ret Ret

	ctx, cancel := context.WithCancel(context.Background())

	recorder, err := NewFileRecorder(os.TempDir())
	if err != nil {
		t.Fatalf("create recorder error:%v", err)
	}
	uploadedSizeWhileCancel := int64(1024 * 1024 * 4)
	// 上传 file
	source, err := NewUploadSourceFile(tempFile.Name())
	err = uploadManager.Put(ctx, &ret, getUploadToken(), nil, source, &UploadExtra{
		Params:             params,
		TryTimes:           1,
		HostFreezeDuration: 0,
		MimeType:           "",
		OnProgress: func(fileSize, uploaded int64) {
			fmt.Printf("resume v2 record 01: upload file progress: %d-%d \n", uploaded, fileSize)
			if uploaded >= uploadedSizeWhileCancel {
				cancel()
			}
		},
		UploadResumeVersion: UploadResumeV2,
		UploadThreshold:     1024 * 1024 * 2,
		Recorder:            recorder,
		PartSize:            1024 * 1024,
	})
	if !isCancelErr(err) {
		t.Fatalf("resume upload file with record error:%v", err)
	}

	// 再次上传 file
	err = uploadManager.Put(context.Background(), &ret, getUploadToken(), nil, source, &UploadExtra{
		Params:             params,
		TryTimes:           1,
		HostFreezeDuration: 0,
		MimeType:           "",
		OnProgress: func(fileSize, uploaded int64) {
			fmt.Printf("resume v2 record 02: upload file progress: %d-%d \n", uploaded, fileSize)
			if uploaded < uploadedSizeWhileCancel {
				t.Fatalf("resume v2 upload file with record error: uploaded size should bigger than %d", uploadedSizeWhileCancel)
			}
		},
		UploadResumeVersion: UploadResumeV2,
		UploadThreshold:     1024 * 1024 * 2,
		Recorder:            recorder,
		PartSize:            1024 * 1024,
	})

	if err != nil {
		t.Fatalf("resume v2 upload file with record error:%v", err)
	}
	if len(ret.Key) == 0 || len(ret.Hash) == 0 {
		t.Fatal("resume v2 upload file with record error, key or hash is empty")
	}
	if len(ret.Foo) == 0 {
		t.Fatal("resume v2 upload file with record error, foo is empty")
	}
}
