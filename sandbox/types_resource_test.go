//go:build unit

package sandbox

import (
	"encoding/json"
	"testing"
)

func TestSandboxResourceSpecToAPIKodoResource(t *testing.T) {
	accessKey := "ak"
	secretKey := "sk"
	prefix := "datasets/"
	readOnly := true

	resource, err := sandboxResourceSpecToAPI(SandboxResourceSpec{
		Kodo: &KodoResource{
			Bucket:    "test-bucket",
			MountPath: "/mnt/kodo",
			Prefix:    &prefix,
			ReadOnly:  &readOnly,
			AccessKey: &accessKey,
			SecretKey: &secretKey,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(mustMarshalJSON(t, resource), &got); err != nil {
		t.Fatalf("unexpected JSON error: %v", err)
	}
	if got["type"] != "kodo" {
		t.Errorf("expected type kodo, got %v", got["type"])
	}
	if got["bucket"] != "test-bucket" {
		t.Errorf("expected bucket test-bucket, got %v", got["bucket"])
	}
	if got["mount_path"] != "/mnt/kodo" {
		t.Errorf("expected mount_path /mnt/kodo, got %v", got["mount_path"])
	}
	if got["prefix"] != "datasets/" {
		t.Errorf("expected prefix datasets/, got %v", got["prefix"])
	}
	if got["read_only"] != true {
		t.Errorf("expected read_only true, got %v", got["read_only"])
	}
	if got["access_key"] != "ak" {
		t.Errorf("expected access_key ak, got %v", got["access_key"])
	}
	if got["secret_key"] != "sk" {
		t.Errorf("expected secret_key sk, got %v", got["secret_key"])
	}
}

func TestSandboxResourceSpecToAPIKodoResourceValidation(t *testing.T) {
	_, err := sandboxResourceSpecToAPI(SandboxResourceSpec{
		Kodo: &KodoResource{MountPath: "/mnt/kodo"},
	})
	if err == nil {
		t.Fatal("expected error when KodoResource.Bucket is empty")
	}

	_, err = sandboxResourceSpecToAPI(SandboxResourceSpec{
		Kodo: &KodoResource{Bucket: "test-bucket"},
	})
	if err == nil {
		t.Fatal("expected error when KodoResource.MountPath is empty")
	}
}

func mustMarshalJSON(t *testing.T, v interface{}) []byte {
	t.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("unexpected JSON marshal error: %v", err)
	}
	return data
}
