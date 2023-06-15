// +build integration

package storage

import (
	"encoding/json"
	clientV1 "github.com/qiniu/go-sdk/v7/client"
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

func TestRegionUCQueryV4Test(t *testing.T) {
	jsonString := "{\"hosts\":[{\"region\":\"z0\",\"ttl\":86400,\"features\":{\"http3\":{\"enabled\":true},\"ipv6\":{\"enabled\":false}},\"io\":{\"domains\":[\"iovip.qbox.me\"]},\"io_src\":{\"domains\":[\"kodo-phone-zone0-space.kodo-cn-east-1.qiniucs.com\"]},\"up\":{\"domains\":[\"upload.qiniup.com\",\"up.qiniup.com\"],\"old\":[\"upload.qbox.me\",\"up.qbox.me\"]},\"uc\":{\"domains\":[\"uc.qbox.me\"]},\"rs\":{\"domains\":[\"rs-z0.qbox.me\"]},\"rsf\":{\"domains\":[\"rsf-z0.qbox.me\"]},\"api\":{\"domains\":[\"api.qiniu.com\"]},\"s3\":{\"domains\":[\"s3-cn-east-1.qiniucs.com\"],\"region_alias\":\"cn-east-1\"}}]}"

	var ret ucQueryV4Ret
	if err := json.Unmarshal([]byte(jsonString), &ret); err != nil {
		t.Fatalf("json unmarshal error:%v", ret)
	}

	if len(ret.Hosts) == 0 {
		t.Fatalf("Hosts empty:%v", ret)
	}

	host := ret.Hosts[0]
	if len(host.Up.getOneServer()) == 0 {
		t.Fatalf("up empty:%v", ret)
	}

	if len(host.Io.getOneServer()) == 0 {
		t.Fatalf("io empty:%v", ret)
	}

	if len(host.Rs.getOneServer()) == 0 {
		t.Fatalf("rs empty:%v", ret)
	}

	if len(host.Rsf.getOneServer()) == 0 {
		t.Fatalf("rsf empty:%v", ret)
	}

	if len(host.Api.getOneServer()) == 0 {
		t.Fatalf("api empty:%v", ret)
	}

	if len(host.IoSrc.getOneServer()) == 0 {
		t.Fatalf("io src empty:%v", ret)
	}
}

func TestUCRetry(t *testing.T) {
	clientV1.DeepDebugInfo = true
	SetUcHosts("aaa.aaa.com", "uc.qbox.me")
	r, err := GetRegion(testAK, testBucket)
	if err != nil {
		t.Fatalf("GetRegion error:%v", err)
	}

	if len(r.SrcUpHosts) == 0 {
		t.Fatalf("GetRegion up hosts empty:%+v", r)
	}
}
