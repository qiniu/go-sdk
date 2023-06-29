//go:build integration
// +build integration

package client

import (
	"context"
	"fmt"
	"github.com/qiniu/go-sdk/v7/auth"
	"net/http"
	"os"
	"testing"
)

func TestQiniuCredentialMultiXQiniuHeader(t *testing.T) {
	header := http.Header{}
	header.Add("X-Qiniu-a", "a")
	header.Add("X-Qiniu-a", "a1")
	header.Add("X-Qiniu-b", "b")
	_, err := bucketsWithHeader(header)
	if err != nil {
		t.Fatalf("TestMultiXQiniuHeader error:%v", err)
	}
}

func bucketsWithHeader(header http.Header) (buckets []string, err error) {
	testAK := os.Getenv("accessKey")
	testSK := os.Getenv("secretKey")
	mac := auth.New(testAK, testSK)
	reqURL := fmt.Sprintf("https://uc.qbox.me/buckets?shared=%v", true)
	err = DefaultClient.CredentialedCall(context.Background(), mac, auth.TokenQiniu, &buckets, "POST", reqURL, header)
	return
}

func TestEnableTimeStampSignature(t *testing.T) {
	os.Setenv("DISABLE_QINIU_TIMESTAMP_SIGNATURE", "false")
	header := http.Header{}
	if err := addDefaultHeader(header); err != nil {
		t.Fatalf("TestEnableXQiniuDate error:%v", err)
	}
	xQiniuDate := header.Values(RequestHeaderKeyXQiniuDate)
	if len(xQiniuDate) == 0 || len(xQiniuDate[0]) == 0 {
		t.Fatal("TestEnableXQiniuDate xQiniuDate header not set success")
	}
}

func TestDisableQiniuTimeStampSignature(t *testing.T) {
	os.Setenv("DISABLE_QINIU_TIMESTAMP_SIGNATURE", "true")

	header := http.Header{}
	if err := addDefaultHeader(header); err != nil {
		t.Fatalf("TestDisableXQiniuDate error:%v", err)
	}
	xQiniuDate := header.Values(RequestHeaderKeyXQiniuDate)
	if len(xQiniuDate) > 0 {
		t.Fatal("TestDisableXQiniuDate xQiniuDate header should not set success")
	}
}
