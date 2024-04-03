//go:build integration
// +build integration

package apistest_test

import (
	"bytes"
	"context"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/qiniu/go-sdk/v7/client"
	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
	"github.com/qiniu/go-sdk/v7/storagev2/apis"
	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/errors"
	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"
)

var (
	testAK     = os.Getenv("accessKey")
	testSK     = os.Getenv("secretKey")
	testBucket = os.Getenv("QINIU_TEST_BUCKET")
	testDebug  = os.Getenv("QINIU_SDK_DEBUG")

	testKey = "qiniu.png"
)

func init() {
	if testDebug == "true" {
		client.TurnOnDebug()
	}
}

func TestMkBlk(t *testing.T) {
	credentials := credentials.NewCredentials(testAK, testSK)
	storageClient := apis.NewStorage(&http_client.Options{
		Credentials: credentials,
	})
	putPolicy, err := uptoken.NewPutPolicy(testBucket, time.Now().Add(30*time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	buf := make([]byte, 4*1024*1024)
	if _, err = r.Read(buf); err != nil {
		t.Fatal(err)
	}
	bufReader := bytes.NewReader(buf)

	if _, err = storageClient.ResumableUploadV1MakeBlock(context.Background(), &apis.ResumableUploadV1MakeBlockRequest{
		BlockSize: 4 * 1024 * 1024,
		UpToken:   uptoken.NewSigner(putPolicy, credentials),
		Body:      internal_io.NewReadSeekableNopCloser(bufReader),
	}, nil); err != nil {
		t.Fatal(err)
	}

	if _, err = storageClient.ResumableUploadV1MakeBlock(context.Background(), &apis.ResumableUploadV1MakeBlockRequest{
		BlockSize: 4 * 1024 * 1024,
		Body:      internal_io.NewReadSeekableNopCloser(bufReader),
	}, nil); err != nil {
		if err.(errors.MissingRequiredFieldError).Name != "UpToken" {
			t.FailNow()
		}
	}
}

func TestCreateBucket(t *testing.T) {
	credentials := credentials.NewCredentials(testAK, testSK)
	_, err := apis.NewStorage(&http_client.Options{Credentials: credentials}).CreateBucket(context.Background(), &apis.CreateBucketRequest{
		Bucket: testBucket,
		Region: "z0",
	}, nil)
	if err != nil {
		if err.Error() != "the bucket already exists and you own it." {
			t.Fatal(err)
		}
	} else {
		t.FailNow()
	}

	_, err = apis.NewStorage(nil).CreateBucket(context.Background(), &apis.CreateBucketRequest{
		Bucket:      testBucket,
		Credentials: credentials,
		Region:      "z0",
	}, nil)
	if err != nil {
		if err.Error() != "the bucket already exists and you own it." {
			t.Fatal(err)
		}
	} else {
		t.FailNow()
	}

	_, err = apis.NewStorage(nil).CreateBucket(context.Background(), &apis.CreateBucketRequest{
		Bucket: testBucket,
		Region: "z0",
	}, nil)
	if err == nil || err.(errors.MissingRequiredFieldError).Name != "Credentials" {
		t.FailNow()
	}
}
