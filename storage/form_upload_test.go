// +build integration

package storage

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"
)

func TestFormUploadPutFile(t *testing.T) {
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

	upToken := putPolicy.UploadToken(mac)
	upHosts := []string{testUpHost, "https://" + testUpHost, ""}
	for _, upHost := range upHosts {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
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

func TestFormUploadPutFileWithBackup(t *testing.T) {
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

	region, err := GetRegion(mac.AccessKey, testBucket)
	if err != nil {
		t.Fatal("get region error:", err)
	}

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
}
