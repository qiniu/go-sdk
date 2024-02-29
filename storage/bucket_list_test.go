//go:build integration
// +build integration

package storage

import (
	"context"
	"testing"
)

func TestList(t *testing.T) {
	ret, _, err := bucketManager.ListFilesWithContext(context.Background(), testBucket,
		ListInputOptionsLimit(1000),
		ListInputOptionsNeedParts(false),
	)
	if err != nil {
		t.Fatalf("List bucket files error: %v\n", err)
	}

	hasParts := false
	for _, item := range ret.Items {
		if len(item.Parts) > 0 {
			hasParts = true
		}
	}
	if hasParts {
		t.Fatal("list files: should no parts")
	}

	ret, _, err = bucketManager.ListFilesWithContext(context.Background(), testBucket,
		ListInputOptionsLimit(1000),
		ListInputOptionsNeedParts(true),
	)
	if err != nil {
		t.Fatalf("List bucket files error: %v\n", err)
	}

	hasParts = false
	for _, item := range ret.Items {
		if len(item.Parts) > 0 {
			hasParts = true
		}
	}
	if !hasParts {
		t.Fatal("list files: should parts")
	}
}
