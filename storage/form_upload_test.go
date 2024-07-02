//go:build integration
// +build integration

package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"
)

func TestFormUploadPutFileWithoutExtra(t *testing.T) {
	var putRet PutRet
	ctx := context.TODO()
	putPolicy := PutPolicy{
		Scope:           testBucket,
		DeleteAfterDays: 7,
	}

	// prepare file for test uploading
	testLocalFile, err := ioutil.TempFile("", "TestFormUploadPutFile")
	if err != nil {
		t.Fatalf("ioutil.TempFile file failed, err: %v", err)
	}
	defer os.Remove(testLocalFile.Name())

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	_, err = io.CopyN(testLocalFile, r, 10*1024*1024)
	if err != nil {
		t.Fatalf("ioutil.TempFile file write failed, err: %v", err)
	}
	_, err = testLocalFile.Seek(0, io.SeekCurrent)
	if err != nil {
		t.Fatalf("ioutil.TempFile file seek failed, err: %v", err)
	}

	upToken := putPolicy.UploadToken(mac)
	testKey := fmt.Sprintf("testPutFileWithoutExtra_%d", r.Int())

	err = formUploader.PutFile(ctx, &putRet, upToken, testKey, testLocalFile.Name(), nil)
	if err != nil {
		t.Fatalf("FormUploader#PutFile() error, %s", err)
	}
	t.Logf("Key: %s, Hash:%s", putRet.Key, putRet.Hash)
}

func TestFormUploadPutFile(t *testing.T) {
	var putRet PutRet
	ctx := context.TODO()

	// prepare file for test uploading
	testLocalFile, err := ioutil.TempFile("", "TestFormUploadPutFile")
	if err != nil {
		t.Fatalf("ioutil.TempFile file failed, err: %v", err)
	}
	defer os.Remove(testLocalFile.Name())

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	_, err = io.CopyN(testLocalFile, r, 10*1024*1024)
	if err != nil {
		t.Fatalf("ioutil.TempFile file write failed, err: %v", err)
	}
	_, err = testLocalFile.Seek(0, io.SeekCurrent)
	if err != nil {
		t.Fatalf("ioutil.TempFile file seek failed, err: %v", err)
	}

	putPolicy := PutPolicy{
		Scope:           testBucket,
		DeleteAfterDays: 7,
	}
	upToken := putPolicy.UploadToken(mac)
	upHosts := []string{testUpHost, "https://" + testUpHost, ""}
	for _, upHost := range upHosts {
		testKey := fmt.Sprintf("testPutFileKey_%d", r.Int())

		err = formUploader.PutFile(ctx, &putRet, upToken, testKey, testLocalFile.Name(), &PutExtra{
			UpHost: upHost,
		})
		if err != nil {
			t.Fatalf("FormUploader#PutFile() error, %s", err)
		}
		t.Logf("Key: %s, Hash:%s", putRet.Key, putRet.Hash)
	}
}

func TestFormUploadTrafficLimit(t *testing.T) {
	var putRet PutRet
	ctx := context.TODO()

	testLocalFile, err := ioutil.TempFile("", "TestFormUploadPutFile")
	if err != nil {
		t.Fatalf("ioutil.TempFile file failed, err: %v", err)
	}
	defer os.Remove(testLocalFile.Name())

	putPolicy := PutPolicy{
		Scope:           testBucket,
		DeleteAfterDays: 7,
		TrafficLimit:    100 * 1024 * 8, // 限速 100KB/s，范围：100KB/s - 100MB/s
	}
	upToken := putPolicy.UploadToken(mac)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	testKey := fmt.Sprintf("testPutFileKey_%d", r.Int())

	st := time.Now().UnixMilli()

	data := make([]byte, 1024*500)
	err = formUploader.Put(ctx, &putRet, upToken, testKey, bytes.NewReader(data), int64(len(data)), &PutExtra{})
	if err != nil {
		t.Fatalf("FormUploader#PutFile() error, %s", err)
	}
	t.Logf("Key: %s, Hash:%s", putRet.Key, putRet.Hash)

	et := time.Now().UnixMilli()

	duration := et - st
	// 限速后，至少需要 4s
	if duration < 5000 {
		//t.Fatal("TestFormUploadTrafficLimit() error, TrafficLimit invalid")
	}
}

func TestFormUploadPutFileWithBackup(t *testing.T) {
	var putRet PutRet
	ctx := context.TODO()
	putPolicy := PutPolicy{
		Scope:           testBucket,
		DeleteAfterDays: 7,
	}

	// prepare file for test uploading
	testLocalFile, err := ioutil.TempFile("", "TestFormUploadPutFileWithBackup")
	if err != nil {
		t.Fatalf("ioutil.TempFile file failed, err: %v", err)
	}
	defer os.Remove(testLocalFile.Name())

	region, err := GetRegion(mac.AccessKey, testBucket)
	if err != nil {
		t.Fatal("get region error:", err)
	}

	// mock host
	customizedHost := []string{"mock.qiniu.com"}
	customizedHost = append(customizedHost, region.SrcUpHosts...)
	region.SrcUpHosts = customizedHost
	cfg := Config{}
	cfg.UseCdnDomains = false
	cfg.Region = region
	uploader := NewFormUploader(&cfg)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	testKey := fmt.Sprintf("testPutFileKey_%d", r.Int())

	upToken := putPolicy.UploadToken(mac)
	err = uploader.PutFile(ctx, &putRet, upToken, testKey, testLocalFile.Name(), &PutExtra{})
	if err != nil {
		t.Fatalf("FormUploader#PutFile() error, %s", err)
	}
	t.Logf("Key: %s, Hash:%s", putRet.Key, putRet.Hash)

	// cancel
	customizedHost = []string{}
	customizedHost = append(customizedHost, region.SrcUpHosts...)
	region.SrcUpHosts = customizedHost
	cfg = Config{}
	cfg.UseCdnDomains = false
	cfg.Region = region
	uploader = NewFormUploader(&cfg)
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
	testKey = fmt.Sprintf("testPutFileKey_%d", r.Int())

	ctx, cancel := context.WithCancel(context.TODO())
	go func() {
		time.Sleep(time.Millisecond * 100)
		cancel()
	}()
	upToken = putPolicy.UploadToken(mac)
	err = uploader.PutFile(ctx, &putRet, upToken, testKey, testLocalFile.Name(), &PutExtra{})
	if err == nil || !strings.Contains(err.Error(), "canceled") {
		t.Fatal("FormUploader#PutFile() cancel error:", err)
	}

}
