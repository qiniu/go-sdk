//go:build integration
// +build integration

package storage

import (
	"os"
	"strings"
	"sync"
	"testing"

	clientV1 "github.com/qiniu/go-sdk/v7/client"
)

func TestUCRetry(t *testing.T) {
	clientV1.DebugMode = true
	clientV1.DeepDebugInfo = true
	defer func() {
		clientV1.DebugMode = false
		clientV1.DeepDebugInfo = false
	}()

	SetUcHosts("aaa.aaa.com", "uc.qbox.me")
	defer SetUcHosts("uc.qbox.me")

	_ = os.Remove(regionV2CachePath)
	regionV2Cache = sync.Map{}

	r, err := GetRegion(testAK, testBucket)
	if err != nil {
		t.Fatalf("GetRegion error:%v", err)
	}

	if !strings.Contains(r.SrcUpHosts[0], "up-") {
		t.Fatal("GetRegion: SrcUpHosts error")
	}

	if !strings.Contains(r.CdnUpHosts[0], "upload-") {
		t.Fatal("GetRegion: CdnUpHosts error")
	}

	if !strings.Contains(r.RsHost, "rs-") {
		t.Fatal("GetRegion: RsHost error")
	}

	if !strings.Contains(r.RsfHost, "rsf-") {
		t.Fatal("GetRegion: RsfHost error")
	}

	if !strings.Contains(r.ApiHost, "api-") {
		t.Fatal("GetRegion: ApiHost error")
	}

	if !strings.Contains(r.IovipHost, "iovip-") {
		t.Fatal("GetRegion: IovipHost error")
	}

	if !strings.Contains(r.IoSrcHost, testBucket) {
		t.Fatal("GetRegion: IoSrcHost error")
	}
}
