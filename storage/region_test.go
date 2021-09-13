// +build integration

package storage

import (
	"encoding/json"
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

func TestUcQueryRetUnmarshalJSON(t *testing.T) {
	retJson := `{
  "region": "z2",
  "ttl": 86400,
  "io": {
    "src": {
      "main": [
        "iovip-z2.qbox.me"
      ],
      "info": "compatible to non-SNI device"
    }
  },
  "up": {
    "acc": {
      "main": [
        "upload-z2.qiniup.com"
      ],
      "backup": [
        "upload-dg.qiniup.com",
        "upload-fs.qiniup.com"
      ]
    },
    "old_acc": {
      "main": [
        "upload-z2.qbox.me"
      ],
      "info": "compatible to non-SNI device"
    },
    "old_src": {
      "main": [
        "up-z2.qbox.me"
      ],
      "info": "compatible to non-SNI device"
    },
    "src": {
      "main": [
        "up-z2.qiniup.com"
      ],
      "backup": [
        "up-dg.qiniup.com",
        "up-fs.qiniup.com"
      ]
    }
  },
  "uc": {
    "acc": {
      "main": [
        "uc.qbox.me"
      ]
    }
  },
  "rs": {
    "acc": {
      "main": [
        "rs-z2.qbox.me"
      ]
    }
  },
  "rsf": {
    "acc": {
      "main": [
        "rsf-z2.qbox.me"
      ]
    }
  },
  "api": {
    "acc": {
      "main": [
        "api-z2.qiniu.com"
      ]
    }
  }
}`
	var ucQueryRet UcQueryRet
	err := json.Unmarshal([]byte(retJson), &ucQueryRet)
	if err != nil {
		t.Fatalf("UcQueryRetUnmarshalJSON error: %v\n", err)
	}

	if ucQueryRet.Io == nil {
		t.Fatalf("UcQueryRetUnmarshalJSON Io was nil")
	}

	if ucQueryRet.IoInfo == nil {
		t.Fatalf("UcQueryRetUnmarshalJSON IoInfo was nil")
	}

	if ucQueryRet.Up == nil {
		t.Fatalf("UcQueryRetUnmarshalJSON Up was nil")
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
