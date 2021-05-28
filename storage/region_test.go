package storage

import (
	"strings"
	"testing"
)

func TestEndpoint(t *testing.T) {
	type input struct {
		UseHttps bool
		Host     string
	}
	testInputs := []input{
		{UseHttps: true, Host: "rs.qiniu.com"},
		{UseHttps: false, Host: "rs.qiniu.com"},
		{UseHttps: true, Host: ""},
		{UseHttps: false, Host: ""},
		{UseHttps: true, Host: "https://rs.qiniu.com"},
		{UseHttps: false, Host: "https://rs.qiniu.com"},
		{UseHttps: false, Host: "http://rs.qiniu.com"},
	}
	testWants := []string{"https://rs.qiniu.com", "http://rs.qiniu.com", "", "", "https://rs.qiniu.com",
		"http://rs.qiniu.com", "http://rs.qiniu.com"}

	for ind, testInput := range testInputs {
		testGot := endpoint(testInput.UseHttps, testInput.Host)
		testWant := testWants[ind]
		if testGot != testWant {
			t.Fail()
		}
	}
}

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
