// +build integration

package storage

import (
	"strings"
	"testing"
)

func TestRegion(t *testing.T) {
	region1, err := GetRegion(testAK, testBucket)
	if err != nil {
		t.Fatalf("GetRegion error: %v\n", err)
	}

	if !strings.HasPrefix(region1.IovipHost, "iovip") || !strings.HasSuffix(region1.IovipHost, ".qbox.me") {
		t.Fatalf("region1.IovipHost is wrong")
	}
}

func TestRegionWithNoProtocol(t *testing.T) {
	UcHost = "uc.qbox.me"
	region1, err := GetRegion(testAK, testBucket)
	if err != nil {
		t.Fatalf("GetRegion error: %v\n", err)
	}

	if !strings.HasPrefix(region1.IovipHost, "iovip") || !strings.HasSuffix(region1.IovipHost, ".qbox.me") {
		t.Fatalf("region1.IovipHost is wrong: %v\v", region1.IovipHost)
	}
}

func TestRegionWithSetHost(t *testing.T) {
	SetUcHost("uc.qbox.me", true)
	region1, err := GetRegion(testAK, testBucket)
	if err != nil {
		t.Fatalf("GetRegion error: %v\n", err)
	}

	if !strings.HasPrefix(region1.IovipHost, "iovip") || !strings.HasSuffix(region1.IovipHost, ".qbox.me") {
		t.Fatalf("region1.IovipHost is wrong: %v\v", region1.IovipHost)
	}
}
