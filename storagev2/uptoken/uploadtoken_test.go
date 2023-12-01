//go:build unit
// +build unit

package uptoken_test

import (
	"context"
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
	upToken, err := signer.RetrieveUpToken(context.Background())
	if err != nil {
		t.Fatalf("failed to retrieve uptoken: %s", err)
	}

	parser := uptoken.NewParser(upToken)
	if actualAccessKey, err := parser.RetrieveAccessKey(context.Background()); err != nil {
		t.Fatalf("failed to retrieve accessKey: %s", err)
	} else if actualAccessKey != expectedAccessKey {
		t.Fatalf("unexpected accessKey: %s", actualAccessKey)
	}

	if actualPutPolicy, err := parser.RetrievePutPolicy(context.Background()); err != nil {
		t.Fatalf("failed to retrieve putPolicy: %s", err)
	} else if actualScope, _ := actualPutPolicy.GetScope(); actualScope != expectedBucketName {
		t.Fatalf("unexpected scope: %s", actualScope)
	}
}
