// +build integration

package storage

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
)

func getUploadManager() *UploadManager {
	region01 := &Region{
		SrcUpHosts: []string{"mock01.qiniu.com", "mock02.qiniu.com"},
	}

	region, err := GetRegion(testAK, testBucket)
	if err != nil {
		return nil
	}

	return NewUploadManager(&UploadConfig{
		UseHTTPS:      true,
		UseCdnDomains: false,
		Regions:       NewRegionGroup(region, region01, region),
	})
}

func getUploadToken() string {
	putPolicy := PutPolicy{
		Scope:           testBucket,
		DeleteAfterDays: 7,
	}
	return putPolicy.UploadToken(mac)
}

func TestUploadManagerFormUpload(t *testing.T) {
	data := make([]byte, 1024*1024)
	data = []byte("hello")
	tempFile, err := ioutil.TempFile("", "TestResumeUploadPutFile")
	if err != nil {
		t.Fatalf("create temp file error:%v", err)
	}
	defer func() {
		tempFile.Close()
		os.ReadFile(tempFile.Name())
	}()
	tempFile.Write(data)

	uploadManager := getUploadManager()
	var ret UploadRet
	source, err := NewUploadSourceFile(tempFile.Name())
	err = uploadManager.Put(context.Background(), &ret, getUploadToken(), nil, source, &UploadExtra{
		Params:              nil,
		UpHost:              "",
		TryTimes:            0,
		HostFreezeDuration:  0,
		MimeType:            "",
		OnProgress:          nil,
		UploadResumeVersion: 0,
		UploadThreshold:     0,
		Recorder:            nil,
		PartSize:            0,
	})
	if err != nil {
		t.Fatalf("form upload error:%v", err)
	}
}

func TestUploadManagerResumeV1Upload(t *testing.T) {

}

func TestUploadManagerResumeV2Upload(t *testing.T) {

}
