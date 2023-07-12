//go:build integration
// +build integration

package storage

import (
	"encoding/json"
	"github.com/qiniu/go-sdk/v7/client"
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

func TestRegionInfo(t *testing.T) {
	rs, err := GetRegionsInfo(mac)
	if err != nil {
		t.Fatalf("GetRegionsInfo error: %v\n", err)
	}
	if len(rs) == 0 {
		t.Fatal("GetRegionsInfo error: region is empty \n")
	}

	r := rs[0]
	if len(r.ID) == 0 {
		t.Fatalf("GetRegionsInfo error: r id is empty, %+v", r)
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

func TestRegionV4(t *testing.T) {
	regionGroup, err := getRegionGroup(testAK, testBucket)
	if err != nil {
		t.Fatalf("GetRegion error: %v\n", err)
	}

	if len(regionGroup.regions) == 0 {
		t.Fatalf("region1.IovipHost is wrong")
	}
}

func TestRegionV4WithNoProtocol(t *testing.T) {
	client.DebugMode = true
	ucHosts = []string{"aa.qiniu.com", "uc.qbox.me"}
	defer func() {
		client.DebugMode = false
		ucHosts = []string{"uc.qbox.me"}
	}()
	regionV4CacheLoaded = true
	regionGroup, err := getRegionGroup(testAK, testBucket)
	if err != nil {
		t.Fatalf("GetRegion error: %v\n", err)
	}

	if len(regionGroup.regions) == 0 {
		t.Fatalf("region1.IovipHost is wrong")
	}
}

func TestUcQueryV4RetUnmarshalJSON(t *testing.T) {
	retJson := `{
  "hosts": [
    {
      "region": "z2",
      "ttl": 86400,
      "features": {
        "http3": {
          "enabled": true
        },
        "ipv6": {
          "enabled": false
        }
      },
      "io": {
        "domains": [
          "iovip-z2.qbox.me"
        ]
      },
      "up": {
        "domains": [
          "upload-z2.qiniup.com",
          "up-z2.qiniup.com"
        ],
        "old": [
          "upload-z2.qbox.me",
          "up-z2.qbox.me"
        ]
      },
      "uc": {
        "domains": [
          "uc.qbox.me"
        ]
      },
      "rs": {
        "domains": [
          "rs-z2.qbox.me"
        ]
      },
      "rsf": {
        "domains": [
          "rsf-z2.qbox.me"
        ]
      },
      "api": {
        "domains": [
          "api-z2.qiniu.com"
        ]
      },
      "s3": {
        "domains": [
          "s3-cn-south-1.qiniucs.com"
        ],
        "region_alias": "cn-south-1"
      }
    }
  ]
}`
	var ret ucQueryV4Ret
	err := json.Unmarshal([]byte(retJson), &ret)
	if err != nil {
		t.Fatalf("UcQueryRetUnmarshalJSON error: %v\n", err)
	}

	region := ret.Hosts[0]

	if region.Up.Domains == nil {
		t.Fatalf("UcQueryRetUnmarshalJSON Up was nil")
	}

	if region.Io.Domains == nil {
		t.Fatalf("UcQueryRetUnmarshalJSON Io was nil")
	}

	if region.Rs.Domains == nil {
		t.Fatalf("UcQueryRetUnmarshalJSON Rs was nil")
	}

	if region.Rsf.Domains == nil {
		t.Fatalf("UcQueryRetUnmarshalJSON Rsf was nil")
	}

	if region.Api.Domains == nil {
		t.Fatalf("UcQueryRetUnmarshalJSON Api was nil")
	}
}

func TestRegionV4WithSetHost(t *testing.T) {
	SetUcHost("uc.qbox.me", true)
	regionGroup, err := getRegionGroup(testAK, testBucket)
	if err != nil {
		t.Fatalf("GetRegion error: %v\n", err)
	}

	if len(regionGroup.regions) == 0 {
		t.Fatalf("region1.IovipHost is wrong")
	}
}
