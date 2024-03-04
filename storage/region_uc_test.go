//go:build unit
// +build unit

package storage

import (
	"encoding/json"
	"testing"
)

func TestRegionUCQueryV2Test(t *testing.T) {
	jsonString := "{\"region\":\"z0\",\"ttl\":86400,\"io\":{\"src\":{\"main\":[\"iovip.qbox.me\"]}},\"io_src\":{\"src\":{\"main\":[\"kodo-phone-zone0-space.kodo-cn-east-1.qiniucs.com\"]}},\"up\":{\"acc\":{\"main\":[\"upload.qiniup.com\"],\"backup\":[\"upload-jjh.qiniup.com\",\"upload-xs.qiniup.com\"]},\"old_acc\":{\"main\":[\"upload.qbox.me\"],\"info\":\"compatible to non-SNI device\"},\"old_src\":{\"main\":[\"up.qbox.me\"],\"info\":\"compatible to non-SNI device\"},\"src\":{\"main\":[\"up.qiniup.com\"],\"backup\":[\"up-jjh.qiniup.com\",\"up-xs.qiniup.com\"]}},\"uc\":{\"acc\":{\"main\":[\"uc.qbox.me\"]}},\"rs\":{\"acc\":{\"main\":[\"rs-z0.qbox.me\"]}},\"rsf\":{\"acc\":{\"main\":[\"rsf-z0.qbox.me\"]}},\"api\":{\"acc\":{\"main\":[\"api.qiniu.com\"]}},\"s3\":{\"region_alias\":\"cn-east-1\",\"src\":{\"main\":[\"s3-cn-east-1.qiniucs.com\"]}}}"

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

	if len(ret.IoSrcInfo) == 0 {
		t.Fatalf("io src empty:%v", ret)
	}
}
