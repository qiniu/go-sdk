//go:build unit
// +build unit

package uptoken_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"
)

func TestSignPutPolicy(t *testing.T) {
	const expectedBucketName = "testbucket"
	const expectedAccessKey = "testaccesskey"
	const expectedSecretKey = "testsecretkey"
	const expectedExpires = int64(1675937798)

	putPolicy, err := uptoken.NewPutPolicy(expectedBucketName, time.Unix(expectedExpires, 0))
	if err != nil {
		t.Fatalf("failed to create put policy: %s", err)
	}

	if scope, _ := putPolicy.GetScope(); scope != expectedBucketName {
		t.Fatalf("unexpected bucket name: %s", expectedBucketName)
	}

	if actualDeadline, _ := putPolicy.GetDeadline(); actualDeadline != expectedExpires {
		t.Fatalf("unexpected deadline: %d", actualDeadline)
	}

	signer := uptoken.NewSigner(putPolicy, credentials.NewCredentials(expectedAccessKey, expectedSecretKey))
	upToken, err := signer.GetUpToken(context.Background())
	if err != nil {
		t.Fatalf("failed to retrieve uptoken: %s", err)
	}

	parser := uptoken.NewParser(upToken)
	if actualAccessKey, err := parser.GetAccessKey(context.Background()); err != nil {
		t.Fatalf("failed to retrieve accessKey: %s", err)
	} else if actualAccessKey != expectedAccessKey {
		t.Fatalf("unexpected accessKey: %s", actualAccessKey)
	}

	if actualPutPolicy, err := parser.GetPutPolicy(context.Background()); err != nil {
		t.Fatalf("failed to retrieve putPolicy: %s", err)
	} else if actualScope, _ := actualPutPolicy.GetScope(); actualScope != expectedBucketName {
		t.Fatalf("unexpected scope: %s", actualScope)
	}
}

func TestSignPutPolicyByDefaultCredentials(t *testing.T) {
	const expectedBucketName = "testbucket"
	const expectedAccessKey = "testaccesskey"
	const expectedSecretKey = "testsecretkey"
	const expectedExpires = int64(1675937798)

	os.Setenv("QINIU_ACCESS_KEY", expectedAccessKey)
	os.Setenv("QINIU_SECRET_KEY", expectedSecretKey)
	defer func() {
		os.Unsetenv("QINIU_ACCESS_KEY")
		os.Unsetenv("QINIU_SECRET_KEY")
	}()

	putPolicy, err := uptoken.NewPutPolicy(expectedBucketName, time.Unix(expectedExpires, 0))
	if err != nil {
		t.Fatalf("failed to create put policy: %s", err)
	}
	signer := uptoken.NewSigner(putPolicy, nil)
	if accessKey, _ := signer.GetAccessKey(context.Background()); accessKey != "testaccesskey" {
		t.Fatalf("failed to retrieve accessKey: %s", err)
	}
}
