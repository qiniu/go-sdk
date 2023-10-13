//go:build integration
// +build integration

package storage

import (
	"io"
	"testing"
	"time"

	clientv1 "github.com/qiniu/go-sdk/v7/client"
)

func TestGet(t *testing.T) {

	key := "TestGet_A" + time.Now().String()

	data := "just test get object!"
	err := putDataByResumableV2(key, []byte(data))
	if err != nil {
		t.Logf("StatWithOption test upload data error, %s", err)
	}

	bm := NewBucketManager(mac, &Config{})
	resp, err := bm.Get(testBucket, key, &GetObjectInput{
		DownloadDomains: []string{
			testBucketDomain,
		},
		PresignUrl: true,
		Range:      "",
	})
	if err != nil {
		t.Logf("Get test error, %s", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Logf("Get test read body error, %s", err)
	}

	if string(body) != data {
		t.Logf("Get test body diff:\n%s \n%s", data, string(body))
	}

	if len(resp.Metadata) == 0 {
		t.Fatal("Get test Metadata empty")
	}

	if resp.LastModified.IsZero() {
		t.Fatal("Get test LastModified empty")
	}

	if len(resp.ETag) == 0 {
		t.Fatal("Get test ETag empty")
	}

	if len(resp.ContentType) == 0 {
		t.Fatal("Get test ContentType empty")
	}

	if resp.ContentLength <= 0 {
		t.Fatal("Get test ContentLength empty")
	}

	// Get With Range
	resp, err = bm.Get(testBucket, key, &GetObjectInput{
		DownloadDomains: []string{},
		PresignUrl:      true,
		Range:           "bytes=2-5",
	})
	if err != nil {
		t.Logf("Get test error, %s", err)
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Logf("Get test read body error, %s", err)
	}

	if string(body) != "st t" {
		t.Logf("Get test body Range diff:\n%s \nst ", string(body))
	}
}

func TestGetTrafficLimit(t *testing.T) {
	clientv1.DeepDebugInfo = true
	key := "TestGetTrafficLimit_A" + time.Now().String()
	data := make([]byte, 410*1024)
	err := putDataByResumableV2(key, data)
	if err != nil {
		t.Fatalf("TestGetTrafficLimit test upload data error, %s", err)
	}

	st := time.Now().UnixMilli()
	bm := NewBucketManager(mac, &Config{})
	resp, err := bm.Get(testBucket, key, &GetObjectInput{
		DownloadDomains: []string{
			testBucketDomain,
		},
		PresignUrl:   true,
		Range:        "",
		TrafficLimit: 819200, // 限速：100KB/s 单位：Kb/s
	})
	if err != nil {
		t.Fatalf("Get test error, %s", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil || body == nil {
		t.Logf("Get test read body error, %s", err)
	}

	et := time.Now().UnixMilli()
	duration := et - st
	// 限速后，至少需要 4s
	if duration < 4000 {
		//t.Fatal("TestGetTrafficLimit() error, TrafficLimit invalid")
	}
}
