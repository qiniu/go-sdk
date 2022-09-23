// +build integration

package storage

import (
	"encoding/json"
	"testing"
)

func TestRegionUCQueryV2Test(t *testing.T) {
	jsonString := "{\"region\":\"z1\",\"ttl\":86400,\"io\":{\"src\":{\"main\":[\"iovip-z1.qbox.me\"]}},\"up\":{\"acc\":{\"main\":[\"upload-z1.qiniup.com\"]},\"old_acc\":{\"main\":[\"upload-z1.qbox.me\"],\"info\":\"compatible to non-SNI device\"},\"old_src\":{\"main\":[\"up-z1.qbox.me\"],\"info\":\"compatible to non-SNI device\"},\"src\":{\"main\":[\"up-z1.qiniup.com\"]}},\"uc\":{\"acc\":{\"main\":[\"uc.qbox.me\"]}},\"rs\":{\"acc\":{\"main\":[\"rs-z1.qbox.me\"]}},\"rsf\":{\"acc\":{\"main\":[\"rsf-z1.qbox.me\"]}},\"api\":{\"acc\":{\"main\":[\"api-z1.qiniu.com\"]}}}"

	var ret UcQueryRet
	if err := json.Unmarshal([]byte(jsonString), &ret); err != nil {
		t.Fatalf("json unmarshal error:%v", ret)
	}

	if len(ret.Up) == 0 {
		t.Fatalf("up empty:%v", ret)
	}

	if len(ret.IoInfo) == 0 {
		t.Fatalf("io info empty:%v", ret)
	}

	if len(ret.Io) == 0 {
		t.Fatalf("io empty:%v", ret)
	}

	if len(ret.RsInfo) == 0 {
		t.Fatalf("rs empty:%v", ret)
	}

	if len(ret.RsfInfo) == 0 {
		t.Fatalf("rsf empty:%v", ret)
	}

	if len(ret.ApiInfo) == 0 {
		t.Fatalf("api empty:%v", ret)
	}
}
