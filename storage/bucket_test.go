//go:build integration
// +build integration

package storage

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/client"
)

var (
	testAK           = os.Getenv("accessKey")
	testSK           = os.Getenv("secretKey")
	testBucket       = os.Getenv("QINIU_TEST_BUCKET")
	testBucketDomain = os.Getenv("QINIU_TEST_DOMAIN")
	testPipeline     = os.Getenv("QINIU_TEST_PIPELINE")
	testDebug        = os.Getenv("QINIU_SDK_DEBUG")
	testUpHost       = os.Getenv("QINIU_TEST_UP_HOST")

	testKey      = "qiniu.png"
	testFetchUrl = "http://devtools.qiniu.com/qiniu.png"
	testSiteUrl  = "http://devtools.qiniu.com"
)

// 现在qbox.Mac是auth.Credentials的别名， 这个地方使用原来的qbox.Mac
// 测试兼容性是否正确
var (
	mac              *qbox.Mac
	bucketManager    *BucketManager
	operationManager *OperationManager
	formUploader     *FormUploader
	resumeUploader   *ResumeUploader
	resumeUploaderV2 *ResumeUploaderV2
	base64Uploader   *Base64Uploader
	clt              client.Client
)

func init() {
	if testDebug == "true" {
		client.TurnOnDebug()
	}
	clt = client.Client{
		Client: &http.Client{
			Timeout:   time.Minute * 10,
			Transport: client.DefaultTransport,
		},
	}
	mac = auth.New(testAK, testSK)
	cfg := Config{}
	cfg.UseCdnDomains = false
	cfg.UseHTTPS = true
	bucketManager = NewBucketManagerEx(mac, &cfg, &clt)
	operationManager = NewOperationManagerEx(mac, &cfg, &clt)
	formUploader = NewFormUploaderEx(&cfg, &clt)
	resumeUploader = NewResumeUploaderEx(&cfg, &clt)
	resumeUploaderV2 = NewResumeUploaderV2Ex(&cfg, &clt)
	base64Uploader = NewBase64UploaderEx(&cfg, &clt)
	rand.Seed(time.Now().Unix())
}

// Test get zone
func TestGetZone(t *testing.T) {
	zone, err := GetZone(testAK, testBucket)
	if err != nil {
		t.Fatalf("GetZone() error, %s", err)
	}
	t.Log(zone.String())
}

// TestCreate 测试创建空间的功能
func TestCreate(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	bucketName := fmt.Sprintf("gosdk-test-%d", r.Int())
	bucketManager.DropBucket(bucketName)
	err := bucketManager.CreateBucket(bucketName, RIDHuadong)
	bucketManager.DropBucket(bucketName)
	if err != nil {
		t.Fatalf("CreateBucket() error: %v\n", err)
	}
}

// TestUpdateObjectStatus 测试更新文件状态的功能
func TestUpdateObjectStatus(t *testing.T) {
	keysToStat := []string{"qiniu.png"}

	for _, eachKey := range keysToStat {
		err := bucketManager.UpdateObjectStatus(testBucket, eachKey, false)
		if err != nil {
			if !strings.Contains(err.Error(), "already disabled") {
				t.Fatalf("UpdateObjectStatus error: %v\n", err)
			}
		}
		err = bucketManager.UpdateObjectStatus(testBucket, eachKey, true)
		if err != nil {
			if !strings.Contains(err.Error(), "already enabled") {
				t.Fatalf("UpdateObjectStatus error: %v\n", err)
			}
		}
	}
}

// Test get bucket list
func TestBuckets(t *testing.T) {
	shared := true
	buckets, err := bucketManager.Buckets(shared)
	if err != nil {
		t.Fatalf("Buckets() error, %s", err)
	}

	for _, bucket := range buckets {
		t.Log(bucket)
	}
}

// Test get bucket list v4
func TestBucketsV4(t *testing.T) {
	var input BucketV4Input
	for {
		output, err := bucketManager.BucketsV4(&input)
		if err != nil {
			t.Fatalf("Buckets() error, %s", err)
		}

		for _, bucket := range output.Buckets {
			t.Log(bucket)

			info, err := bucketManager.GetBucketInfo(bucket.Name)
			if err != nil {
				t.Fatalf("GetBucketInfo() error, %s", err)
			}
			t.Log(info)
		}
		if output.IsTruncated {
			input.Marker = output.NextMarker
		} else {
			break
		}
	}
}

// Test set remark
func TestSetRemark(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	bucketName := fmt.Sprintf("gosdk-test-%d", r.Int())
	bucketManager.DropBucket(bucketName)

	err := bucketManager.CreateBucket(bucketName, RIDHuadong)
	defer bucketManager.DropBucket(bucketName)
	if err != nil {
		t.Fatalf("CreateBucket() error: %v\n", err)
	}

	info, err := bucketManager.GetBucketInfo(bucketName)
	if err != nil {
		t.Fatalf("GetBucketInfo() error, %s", err)
	}
	if info.Remark != "" {
		t.Logf("GetBucketInfo returns non-empty remark, %s", info.Remark)
		t.Fail()
	}

	remark := fmt.Sprintf("test-remark-%d", r.Int())
	err = bucketManager.SetRemark(bucketName, remark)
	if err != nil {
		t.Fatalf("SetRemark() error: %v\n", err)
	}
	info, err = bucketManager.GetBucketInfo(bucketName)
	if err != nil {
		t.Fatalf("GetBucketInfo() error, %s", err)
	}
	if info.Remark != remark {
		t.Logf("GetBucketInfo returns unexpected remark, %s", info.Remark)
		t.Fail()
	}
}

// Test get file info
func TestStat(t *testing.T) {
	key := "qiniu.png"
	copyKey := "staus_copy_" + key
	if e := bucketManager.Copy(testBucket, key, testBucket, copyKey, true); e != nil {
		t.Logf("1 Stat Copy error, %s", e)
		t.Fail()
	}
	if e := bucketManager.ChangeType(testBucket, copyKey, 2); e != nil {
		t.Logf("1 Stat Copy error, %s", e)
		t.Fail()
	}
	if e := bucketManager.RestoreAr(testBucket, copyKey, 2); e != nil {
		t.Logf("1 Stat Restore error, %s", e)
		t.Fail()
	}
	if info, e := bucketManager.Stat(testBucket, copyKey); e != nil ||
		info.Type != 2 || len(info.Hash) == 0 ||
		info.RestoreStatus == 0 {
		t.Logf("1 Stat() error, %+v", e)
		t.Fail()
	} else {
		t.Logf("1 FileInfo:\n %s", info.String())
	}

	bucketManager.Delete(testBucket, copyKey)
	copyKey = "rule_" + copyKey
	ruleName := "golangStatusTest"
	bucketManager.DelBucketLifeCycleRule(testBucket, ruleName)
	if e := bucketManager.AddBucketLifeCycleRule(testBucket, &BucketLifeCycleRule{
		Name:                   ruleName,
		Prefix:                 "",
		ToLineAfterDays:        10,
		ToArchiveIRAfterDays:   20,
		ToArchiveAfterDays:     30,
		ToDeepArchiveAfterDays: 40,
		DeleteAfterDays:        50,
	}); e != nil {
		t.Logf("Stat AddBucketLifeCycleRule() error, %s", e)
		t.Fail()
	}

	if e := bucketManager.Copy(testBucket, key, testBucket, copyKey, true); e != nil {
		t.Logf("2 Stat Copy error, %s", e)
		t.Fail()
	}
	if e := bucketManager.DeleteAfterDays(testBucket, copyKey, 1); e != nil {
		t.Logf("2 Stat Delete error, %s", e)
		t.Fail()
	}
	client.DebugMode = true
	if info, e := bucketManager.Stat(testBucket, copyKey); e != nil ||
		len(info.Hash) == 0 || info.Expiration == 0 {
		t.Logf("3 Stat() error, %v", e)
		t.Fail()
	} else {
		t.Logf("3 FileInfo:\n %s", info.String())
	}

	bucketManager.Delete(testBucket, copyKey)
	bucketManager.DelBucketLifeCycleRule(testBucket, ruleName)
}

func TestStatWithOption(t *testing.T) {

	key := "stat_with_option_" + time.Now().String()

	data := make([]byte, 1024*1024*5)
	err := putDataByResumableV2(key, data)
	if err != nil {
		t.Logf("StatWithOption test upload data error, %s", err)
		return
	}

	opt := &StatOpts{NeedParts: true}
	info, err := bucketManager.StatWithOpts(testBucket, key, opt)
	if err != nil || info.Parts == nil {
		t.Logf("StatWithOption() error, %s", err)
		t.Fail()
	} else {
		t.Logf("FileInfo:\n %s", info.String())
	}
}

func TestStatWithMeta(t *testing.T) {

	key := "meta_test_" + time.Now().String()

	data := make([]byte, 1024*1024*5)
	err := putDataByResumableV2(key, data)
	if err != nil {
		t.Logf("TestStatWithMeta test upload data error, %s", err)
		return
	}

	info, err := bucketManager.Stat(testBucket, key)
	if err != nil {
		t.Logf("TestStatWithMeta() Stat error, %s", err)
		t.Fail()
	}
	if len(info.MetaData) == 0 {
		t.Log("TestStatWithMeta() MetaData is empty")
		t.Fail()
	}

	// 同时修改 mime 和 meta
	err = bucketManager.ChangeMimeAndMeta(testBucket, key, "application/abc", map[string]string{
		"x-qn-meta-b": "x-qn-meta-bb-value",
		"x-qn-meta-c": "x-qn-meta-c-value",
	})
	if err != nil {
		t.Logf("TestStatWithMeta() ChangeMimeAndMeta error, %s", err)
		t.Fail()
	}

	info, err = bucketManager.Stat(testBucket, key)
	if err != nil {
		t.Logf("TestStatWithMeta() Stat 2 error, %s", err)
		t.Fail()
	}

	if info.MimeType != "application/abc" {
		t.Log("TestStatWithMeta() MimeType c is error")
		t.Fail()
	}

	if len(info.MetaData) == 0 {
		t.Log("TestStatWithMeta() MetaData 2 is empty")
		t.Fail()
	}

	if info.MetaData["b"] != "x-qn-meta-bb-value" {
		t.Log("TestStatWithMeta() MetaData b is error")
		t.Fail()
	}

	if info.MetaData["c"] != "x-qn-meta-c-value" {
		t.Log("TestStatWithMeta() MetaData c is error")
		t.Fail()
	}

	// 只修改 meta
	err = bucketManager.ChangeMeta(testBucket, key, map[string]string{
		"x-qn-meta-c": "x-qn-meta-cc-value",
	})
	if err != nil {
		t.Logf("TestStatWithMeta() ChangeMimeAndMeta error, %s", err)
		t.Fail()
	}
	info, err = bucketManager.Stat(testBucket, key)
	if err != nil {
		t.Logf("TestStatWithMeta() Stat 2 error, %s", err)
		t.Fail()
	}

	if info.MimeType != "application/abc" {
		t.Log("TestStatWithMeta() MimeType c is error")
		t.Fail()
	}

	if len(info.MetaData) == 0 {
		t.Log("TestStatWithMeta() MetaData 2 is empty")
		t.Fail()
	}

	if info.MetaData["b"] != "x-qn-meta-bb-value" {
		t.Log("TestStatWithMeta() MetaData b is error")
		t.Fail()
	}

	if info.MetaData["c"] != "x-qn-meta-cc-value" {
		t.Log("TestStatWithMeta() MetaData c is error")
		t.Fail()
	}

	// 只修改 meta， 不带 x-qn-meta-
	err = bucketManager.ChangeMeta(testBucket, key, map[string]string{
		"d": "x-qn-meta-d-value",
	})
	if err != nil {
		t.Logf("TestStatWithMeta() ChangeMeta error, %s", err)
		t.Fail()
	}
	info, err = bucketManager.Stat(testBucket, key)
	if err != nil {
		t.Logf("TestStatWithMeta() Stat 2 error, %s", err)
		t.Fail()
	}

	if info.MimeType != "application/abc" {
		t.Log("TestStatWithMeta() MimeType c is error")
		t.Fail()
	}

	if len(info.MetaData) == 0 {
		t.Log("TestStatWithMeta() MetaData 2 is empty")
		t.Fail()
	}

	if info.MetaData["d"] != "x-qn-meta-d-value" {
		t.Log("TestStatWithMeta() MetaData d is error")
		t.Fail()
	}
}

func putDataByResumableV2(key string, data []byte) (err error) {
	var putRet PutRet
	putPolicy := PutPolicy{
		Scope:           testBucket,
		DeleteAfterDays: 1,
	}
	upToken := putPolicy.UploadToken(mac)
	partSize := int64(1024 * 1024)
	err = resumeUploaderV2.Put(context.Background(), &putRet, upToken, key, bytes.NewReader(data), int64(len(data)), &RputV2Extra{
		PartSize: partSize,
		Metadata: map[string]string{
			"x-qn-meta-a": "x-qn-meta-a-value",
			"x-qn-meta-b": "x-qn-meta-b-value",
		},
		CustomVars: map[string]string{
			"x:var-a": "x:var-a-value",
			"x:var-b": "x:var-b-value",
		},
	})
	return
}

func TestCopyMoveDelete(t *testing.T) {
	keysCopyTarget := []string{"qiniu_1.png", "qiniu_2.png", "qiniu_3.png"}
	keysToDelete := make([]string, 0, len(keysCopyTarget))
	for _, eachKey := range keysCopyTarget {
		err := bucketManager.Copy(testBucket, testKey, testBucket, eachKey, true)
		if err != nil {
			t.Logf("Copy() error, %s", err)
			t.Fail()
		}
	}

	for _, eachKey := range keysCopyTarget {
		keyToDelete := eachKey + "_move"
		err := bucketManager.Move(testBucket, eachKey, testBucket, keyToDelete, true)
		if err != nil {
			t.Logf("Move() error, %s", err)
			t.Fail()
		} else {
			keysToDelete = append(keysToDelete, keyToDelete)
		}
	}

	for _, eachKey := range keysToDelete {
		err := bucketManager.Delete(testBucket, eachKey)
		if err != nil {
			t.Logf("Delete() error, %s", err)
			t.Fail()
		}
	}
}

func TestFetch(t *testing.T) {
	ret, err := bucketManager.Fetch(testFetchUrl, testBucket, "qiniu-fetch.png")
	if err != nil {
		t.Logf("Fetch() error, %s", err)
		t.Fail()
	} else {
		t.Logf("FetchRet:\n %s", ret.String())
	}
}

func TestAsyncFetch(t *testing.T) {

	param := AsyncFetchParam{Url: testFetchUrl, Bucket: testBucket}
	ret, err := bucketManager.AsyncFetch(param)
	if err != nil {
		t.Logf("Fetch() error, %s", err)
		t.Fail()
	} else {
		t.Logf("FetchRet:\n %#v", ret)
	}
}

func TestFetchWithoutKey(t *testing.T) {
	ret, err := bucketManager.FetchWithoutKey(testFetchUrl, testBucket)
	if err != nil {
		t.Logf("FetchWithoutKey() error, %s", err)
		t.Fail()
	} else {
		t.Logf("FetchRet:\n %s", ret.String())
	}
}

func TestDeleteAfterDays(t *testing.T) {
	deleteKey := testKey + "_deleteAfterDays"
	days := 7
	bucketManager.Copy(testBucket, testKey, testBucket, deleteKey, true)
	err := bucketManager.DeleteAfterDays(testBucket, deleteKey, days)
	if err != nil {
		t.Logf("DeleteAfterDays() error, %s", err)
		t.Fail()
	}
}

func TestChangeMime(t *testing.T) {
	toChangeKey := testKey + "_changeMime"
	bucketManager.Copy(testBucket, testKey, testBucket, toChangeKey, true)
	newMime := "text/plain"
	err := bucketManager.ChangeMime(testBucket, toChangeKey, newMime)
	if err != nil {
		t.Fatalf("ChangeMime() error, %s", err)
	}

	info, err := bucketManager.Stat(testBucket, toChangeKey)
	if err != nil || info.MimeType != newMime {
		t.Fatalf("ChangeMime() failed, %s", err)
	}
	bucketManager.Delete(testBucket, toChangeKey)
}

func TestChangeType(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	toChangeKey := fmt.Sprintf("%s_changeType_%d", testKey, r.Int())
	bucketManager.Copy(testBucket, testKey, testBucket, toChangeKey, true)
	fileType := 1
	err := bucketManager.ChangeType(testBucket, toChangeKey, fileType)
	if err != nil {
		t.Fatalf("ChangeType() error, %s", err)
	}

	info, err := bucketManager.Stat(testBucket, toChangeKey)
	if err != nil || info.Type != fileType {
		t.Fatalf("ChangeMime() failed, %s", err)
	}
	bucketManager.Delete(testBucket, toChangeKey)
}

func TestRestoreAr(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	toRestoreArKey := fmt.Sprintf("%s_RestoreAr_%d", testKey, r.Int())
	bucketManager.Copy(testBucket, testKey, testBucket, toRestoreArKey, true)
	fileType := 2
	err := bucketManager.ChangeType(testBucket, toRestoreArKey, fileType)
	if err != nil {
		t.Fatalf("ChangeType() error, %s", err)
	}

	err = bucketManager.RestoreAr(testBucket, toRestoreArKey, 5)
	if err != nil {
		t.Fatalf("RestoreAr() failed, %s", err)
	}

	info, err := bucketManager.Stat(testBucket, toRestoreArKey)
	if err != nil || info.Type != fileType {
		t.Fatalf("Stat() failed, %s", err)
	}
	defer bucketManager.Delete(testBucket, toRestoreArKey)
}

// SetImage成功以后， 后台生效需要一段时间；导致集成测试经常失败。
// 如果要修改这一部分代码可以重新开启这个测试
func aTestPrefetchAndImage(t *testing.T) {
	err := bucketManager.SetImage(testSiteUrl, testBucket)
	if err != nil {
		t.Fatalf("SetImage() error, %s", err)
	}

	t.Log("set image success for bucket", testBucket)
	//wait for image set to take effect
	time.Sleep(time.Second * 10)

	err = bucketManager.Prefetch(testBucket, testKey)
	if err != nil {
		t.Fatalf("Prefetch() error, %s", err)
	}

	err = bucketManager.UnsetImage(testBucket)
	if err != nil {
		t.Fatalf("UnsetImage() error, %s", err)
	}

	t.Log("unset image success for bucket", testBucket)
}

func TestListFiles(t *testing.T) {
	limit := 100
	prefix := "listfiles/"
	for i := 0; i < limit; i++ {
		newKey := fmt.Sprintf("%s%s/%d", prefix, testKey, i)
		bucketManager.Copy(testBucket, testKey, testBucket, newKey, true)
	}
	entries, _, _, hasNext, err := bucketManager.ListFiles(testBucket, prefix, "", "", limit)
	if err != nil {
		t.Fatalf("ListFiles() error, %s", err)
	}

	if hasNext {
		t.Fatalf("ListFiles() failed, unexpected hasNext")
	}

	if len(entries) != limit {
		t.Fatalf("ListFiles() failed, unexpected items count, expected: %d, actual: %d", limit, len(entries))
	}

	for _, entry := range entries {
		t.Logf("ListItem:\n%s", entry.String())
	}
}

func TestBatch(t *testing.T) {
	copyCnt := 100
	copyOps := make([]string, 0, copyCnt)
	testKeys := make([]string, 0, copyCnt)
	for i := 0; i < copyCnt; i++ {
		cpKey := fmt.Sprintf("%s_batchcopy_%d", testKey, i)
		testKeys = append(testKeys, cpKey)
		copyOps = append(copyOps, URICopy(testBucket, testKey, testBucket, cpKey, true))
	}

	batchCopyOpRets, bErr := bucketManager.Batch(copyOps)
	if batchCopyOpRets == nil || bErr != nil {
		t.Fatalf("BatchCopy error, %s", bErr)
	}

	statOps := make([]string, 0, copyCnt)
	for _, k := range testKeys {
		statOps = append(statOps, URIStat(testBucket, k))
	}
	batchOpRets, bErr := bucketManager.Batch(statOps)
	if bErr != nil {
		t.Fatalf("BatchStat error, %s", bErr)
	}

	t.Logf("BatchStat: %v", batchOpRets)
}

func TestBatchStat(t *testing.T) {

	statOps := make([]string, 0, 2)
	statOps = append(statOps, URIStat(testBucket, "form_test/120"))

	rets, bErr := bucketManager.Batch(statOps)
	if len(rets) == 0 || bErr != nil {
		t.Fatalf("BatchStat error, %s", bErr)
	}

	ret := rets[0]
	t.Log(ret)
	if len(ret.Data.Hash) == 0 {
		t.Fatalf("Hash error")
	}
	if len(ret.Data.MimeType) == 0 {
		t.Fatalf("MimeType error")
	}
	if ret.Data.Type != 1 {
		t.Fatalf("Type error")
	}
	if ret.Data.Status == nil {
		t.Fatalf("Status error")
	}
}

func TestBatchPartialFailure(t *testing.T) {
	copyOps := make([]string, 0, 2)
	cpKey := fmt.Sprintf("%s_batchcopy_%d", testKey, 0)
	copyOps = append(copyOps, URICopy(testBucket, testKey, testBucket, cpKey, true))
	cpKey = fmt.Sprintf("%s_batchcopy_%d", testKey, 0)
	copyOps = append(copyOps, URICopy(testBucket, testKey+".notexisted", testBucket, cpKey, true))

	rets, bErr := bucketManager.Batch(copyOps)
	if bErr != nil {
		t.Fatalf("BatchCopy error, %s", bErr)
	}
	if len(rets) != 2 {
		t.Fatalf("BatchCopy returns wrong number of results")
	}
	if rets[0].Code != 200 {
		t.Fatalf("BatchCopy[0] error, status = %d", rets[0].Code)
	}
	if rets[1].Code != 612 {
		t.Fatalf("BatchCopy[1] error, status = %d", rets[1].Code)
	}
}

func TestListBucket(t *testing.T) {
	retChan, lErr := bucketManager.ListBucket(testBucket, "form_test", "", "")
	if lErr != nil {
		t.Fatalf("ListBucket: %v\n", lErr)
	}
	for ret := range retChan {
		t.Log(ret.Item)
		if len(ret.Item.Key) == 0 {
			t.Fatalf("key error")
		}
		if len(ret.Item.Hash) == 0 {
			t.Fatalf("Hash error")
		}
		if len(ret.Item.MimeType) == 0 {
			t.Fatalf("MimeType error")
		}
		if ret.Item.Type < 0 {
			t.Fatalf("Type error")
		}
		if ret.Item.Status < 0 {
			t.Fatalf("Status error")
		}
	}

	retChan, lErr = bucketManager.ListBucket(testBucket, "form_test/120", "", "")
	if lErr != nil {
		t.Fatalf("ListBucket: %v\n", lErr)
	}
	for ret := range retChan {
		t.Log(ret.Item)
		if len(ret.Item.Key) == 0 {
			t.Fatalf("key error")
		}
		if len(ret.Item.Hash) == 0 {
			t.Fatalf("Hash error")
		}
		if len(ret.Item.MimeType) == 0 {
			t.Fatalf("MimeType error")
		}
		if ret.Item.Type != 1 {
			t.Fatalf("Type error")
		}
		if ret.Item.Status != 1 {
			t.Fatalf("Status error")
		}
	}
}

func TestListBucketWithCancel(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc()
	_, lErr := bucketManager.ListBucketContext(ctx, testBucket, "", "", "")
	if lErr == nil {
		t.Fatal("ListBucket cancel error")
	}
	if !strings.Contains(lErr.Error(), "context canceled") {
		t.Fatalf("ListBucket cancel error:%s", lErr.Error())
	}
}

func TestGetBucketInfo(t *testing.T) {
	bInfo, bErr := bucketManager.GetBucketInfo(testBucket)
	if bErr != nil {
		t.Fatalf("GetBucketInfo: %v\n", bErr)
	}
	t.Log(bInfo)
}

func TestBucketInfosInRegion(t *testing.T) {
	bInfos, bErr := bucketManager.BucketInfosInRegion(RIDHuadong, true)
	if bErr != nil {
		t.Fatalf("BucketInfosInRegion: %v\n", bErr)
	}
	for _, bInfo := range bInfos {
		t.Log(bInfo)
	}
}

func TestRefererAntiLeechMode(t *testing.T) {
	cfgs := []*ReferAntiLeechConfig{
		{
			Mode: 0, // 关闭referer防盗链
		},
		{
			Mode:    1, // 开启referer白名单
			Pattern: "*.qiniu.com",
		},
		{
			Mode:    2, // 开启referer黑名单
			Pattern: "*.qiniu.com",
		},
	}
	for _, cfg := range cfgs {
		err := bucketManager.SetReferAntiLeechMode(testBucket, cfg)
		if err != nil {
			t.Fatalf("SetReferAntiLeechMode: %v\n", err)
		}
	}

	bInfo, bErr := bucketManager.GetBucketInfo(testBucket)
	if bErr != nil {
		t.Fatalf("GetBucketInfo: %v\n", bErr)
	}
	if bInfo.AntiLeechMode != 2 {
		t.Fatalf("AntiLeechMode expected: %q, got: %q\n", 2, bInfo.AntiLeechMode)
	}
	if len(bInfo.ReferBl) != 1 || bInfo.ReferBl[0] != "*.qiniu.com" {
		t.Fatalf("Referer blacklist expected: %q, got: %q\n", "*.qiniu.com", bInfo.ReferBl[0])
	}
}

func TestBucketLifeCycleRule(t *testing.T) {
	bucketManager.DelBucketLifeCycleRule(testBucket, "golangIntegrationTest")

	err := bucketManager.AddBucketLifeCycleRule(testBucket, &BucketLifeCycleRule{
		Name:                   "golangIntegrationTest",
		Prefix:                 "testPutFileKey",
		DeleteAfterDays:        13,
		ToLineAfterDays:        1,
		ToArchiveIRAfterDays:   2,
		ToArchiveAfterDays:     6,
		ToDeepArchiveAfterDays: 10,
	})
	if err != nil {
		if !strings.Contains(err.Error(), "rule name exists") {
			t.Fatalf("TestBucketLifeCycleRule: %v\n", err)
		}
	}
	rules, err := bucketManager.GetBucketLifeCycleRule(testBucket)
	if err != nil {
		t.Fatalf("TestBucketLifeCycleRule: %v\n", err)
	}
	var foundRule *BucketLifeCycleRule
	for i := range rules {
		if rules[i].Name == "golangIntegrationTest" && rules[i].Prefix == "testPutFileKey" {
			foundRule = &rules[i]
			break
		}
	}
	if foundRule == nil {
		t.Fatalf("TestBucketLifeCycleRule: rule name not found")
	} else if foundRule.DeleteAfterDays != 13 {
		t.Fatalf("TestBucketLifeCycleRule: foundRule.DeleteAfterDays = %d", foundRule.DeleteAfterDays)
	} else if foundRule.ToLineAfterDays != 1 {
		t.Fatalf("TestBucketLifeCycleRule: foundRule.ToLineAfterDays = %d", foundRule.ToLineAfterDays)
	} else if foundRule.ToArchiveAfterDays != 6 {
		t.Fatalf("TestBucketLifeCycleRule: foundRule.ToArchiveAfterDays = %d", foundRule.ToArchiveAfterDays)
	} else if foundRule.ToDeepArchiveAfterDays != 10 {
		t.Fatalf("TestBucketLifeCycleRule: foundRule.ToDeepArchiveAfterDays = %d", foundRule.ToDeepArchiveAfterDays)
	}

	err = bucketManager.UpdateBucketLifeCycleRule(testBucket, &BucketLifeCycleRule{
		Name:                   "golangIntegrationTest",
		Prefix:                 "testPutFileKey",
		DeleteAfterDays:        22,
		ToLineAfterDays:        11,
		ToArchiveIRAfterDays:   12,
		ToArchiveAfterDays:     16,
		ToDeepArchiveAfterDays: 20,
	})

	if err != nil {
		t.Fatalf("TestBucketLifeCycleRule: %v\n", err)
	}

	rules, err = bucketManager.GetBucketLifeCycleRule(testBucket)
	if err != nil {
		t.Fatalf("TestBucketLifeCycleRule: %v\n", err)
	}
	foundRule = nil
	for i := range rules {
		if rules[i].Name == "golangIntegrationTest" && rules[i].Prefix == "testPutFileKey" {
			foundRule = &rules[i]
			break
		}
	}
	if foundRule == nil {
		t.Fatalf("TestBucketLifeCycleRule: rule name not found")
	} else if foundRule.DeleteAfterDays != 22 {
		t.Fatalf("TestBucketLifeCycleRule: foundRule.DeleteAfterDays = %d", foundRule.DeleteAfterDays)
	} else if foundRule.ToLineAfterDays != 11 {
		t.Fatalf("TestBucketLifeCycleRule: foundRule.ToLineAfterDays = %d", foundRule.ToLineAfterDays)
	} else if foundRule.ToArchiveAfterDays != 16 {
		t.Fatalf("TestBucketLifeCycleRule: foundRule.ToArchiveAfterDays = %d", foundRule.ToArchiveAfterDays)
	} else if foundRule.ToDeepArchiveAfterDays != 20 {
		t.Fatalf("TestBucketLifeCycleRule: foundRule.ToDeepArchiveAfterDays = %d", foundRule.ToDeepArchiveAfterDays)
	}

	err = bucketManager.DelBucketLifeCycleRule(testBucket, "golangIntegrationTest")

	if err != nil {
		t.Fatalf("TestBucketLifeCycleRule: %v\n", err)
	}
}

func TestBucketEventRule(t *testing.T) {
	err := bucketManager.AddBucketEvent(testBucket, &BucketEventRule{
		Name:        "golangIntegrationTest",
		Event:       []string{"put", "mkfile"},
		Host:        "www.qiniu.com",
		CallbackURL: []string{"http://www.qiniu.com"},
	})
	if err != nil {
		if !strings.Contains(err.Error(), "event name exists") {
			t.Fatalf("TestBucketEventRule: %v\n", err)
		}
	}
	rules, err := bucketManager.GetBucketEvent(testBucket)
	if err != nil {
		t.Fatalf("TestBucketEventRule: %v\n", err)
	}
	exist := false
	for _, rule := range rules {
		if rule.Name == "golangIntegrationTest" && rule.Host == "www.qiniu.com" {
			exist = true
			break
		}
	}
	if !exist {
		t.Fatalf("TestBucketEventRule: %v\n", err)
	}

	err = bucketManager.UpdateBucketEnvent(testBucket, &BucketEventRule{
		Name:        "golangIntegrationTest",
		Event:       []string{"put", "mkfile"},
		Host:        "www.qiniu.com",
		CallbackURL: []string{"http://www.qiniu.com"},
	})
	if err != nil {
		t.Fatalf("TestBucketEventRule: %v\n", err)
	}
	err = bucketManager.DelBucketEvent(testBucket, "golangIntegrationTest")
	if err != nil {
		t.Fatalf("TestBucketEventRule: %v\n", err)
	}
}

func TestCorsRules(t *testing.T) {
	err := bucketManager.AddCorsRules(testBucket, []CorsRule{
		{
			AllowedOrigin: []string{"http://www.test1.com"},
			AllowedMethod: []string{"GET", "POST"},
		},
	})
	if err != nil {
		t.Fatalf("TestCorsRules: %v\n", err)
	}
	rules, err := bucketManager.GetCorsRules(testBucket)
	if err != nil {
		t.Fatalf("TestCorsRules: %v\n", err)
	}
	for _, r := range rules {
		t.Log(r)
	}

}

func TestListBucketDomains(t *testing.T) {
	bInfos, err := bucketManager.ListBucketDomains(testBucket)
	if err != nil {
		/*
			if !strings.Contains(err.Error(), "404 page not found") {
				t.Fatalf("ListBucketDomains: %q\n", err)
			}
		*/
		t.Fatalf("ListBucketDomains: %q\n", err)
	}
	for _, info := range bInfos {
		t.Log(info)
	}
}

func TestBucketQuota(t *testing.T) {
	err := bucketManager.SetBucketQuota(testBucket, 0, 1000000000000000)
	if err != nil {
		t.Fatalf("TestBucketQuota: %q\n", err)
	}
	quota, err := bucketManager.GetBucketQuota(testBucket)
	if err != nil {
		t.Fatalf("TestBucketQuota: %q\n", err)
	}
	t.Log(quota)
}

func TestSetBucketAccessStyle(t *testing.T) {
	err := bucketManager.TurnOnBucketProtected(testBucket)
	if err != nil {
		t.Fatalf("TestSetBucketAccessStyle: %q\n", err)
	}
	err = bucketManager.TurnOffBucketProtected(testBucket)
	if err != nil {
		t.Fatalf("TestSetBucketAccessStyle: %q\n", err)
	}
}

func TestSetBucketMaxAge(t *testing.T) {
	err := bucketManager.SetBucketMaxAge(testBucket, 20)
	if err != nil {
		t.Fatalf("TestSetBucketMaxAge: %q\n", err)
	}
	err = bucketManager.SetBucketMaxAge(testBucket, 0)
	if err != nil {
		t.Fatalf("TestSetBucketMaxAge: %q\n", err)
	}
}

func TestSetBucketAccessMode(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	bucketName := fmt.Sprintf("gosdk-test-%d", r.Int())
	bucketManager.DropBucket(bucketName)
	err := bucketManager.CreateBucket(bucketName, RIDHuadong)
	if err != nil {
		t.Fatalf("CreateBucket() error: %v\n", err)
	}
	defer bucketManager.DropBucket(bucketName)

	err = bucketManager.MakeBucketPrivate(bucketName)
	if err != nil {
		t.Fatalf("TestSetBucketAccessMode: %q\n", err)
	}
	err = bucketManager.MakeBucketPublic(bucketName)
	if err != nil {
		t.Fatalf("TestSetBucketAccessMode: %q\n", err)
	}
}

func TestMakeURL(t *testing.T) {
	keys := map[string]string{ //rawKey => encodeKey,
		"":            "",
		"abc_def.mp4": "abc_def.mp4",
		"/ab/cd":      "/ab/cd",
		// "ab/中文/de":    "ab/%E4%B8%AD%E6%96%87/de",
		// "ab+-*de f":   "ab%2B-%2Ade%20f",
		"ab:cd": "ab%3Acd",
		// "ab@cd":            "ab%40cd",
		"ab?cd=ef":  "ab%3Fcd%3Def",
		"ab#e~f":    "ab%23e~f",
		"ab//cd":    "ab//cd",
		"abc%2F%2B": "abc%252F%252B",
		"ab cd":     "ab%20cd",
		// "ab/c:d?e#f//gh汉子": "ab/c%3Ad%3Fe%23f//gh%E6%B1%89%E5%AD%90",
	}
	s := MakePublicURL("https://abc.com:123/", "123/def?@#")
	if s != "https://abc.com:123/123/def?@" {
		t.Fatalf("TestMakeURL: %q\n", s)
	}

	s = MakePublicURL("abc.com:123/", "123/def?@#")
	if s != "abc.com:123/123/def?@" {
		t.Fatalf("TestMakeURL: %q\n", s)
	}

	q := make(url.Values)
	q.Add("?", "#")
	s = MakePublicURLv2WithQuery("https://abc.com:123/", "123/def?@#", q)
	if s != "https://abc.com:123/123/def%3F%40%23?%3F=%23" {
		t.Fatalf("TestMakeURL: %q\n", s)
	}

	s = MakePublicURLv2WithQueryString("http://abc.com:123/", "123/def?@#|", "123/def?@#|")
	if s != "http://abc.com:123/123/def%3F@%23%7C?123/def%3F%40%23|" {
		t.Fatalf("TestMakeURL: %q\n", s)
	}

	s = MakePublicURLv2("http://abc.com:123/", "123/def?@#")
	if s != "http://abc.com:123/123/def%3F%40%23" {
		t.Fatalf("TestMakeURL: %q\n", s)
	}

	for rawKey, encodedKey := range keys {
		s = MakePublicURLv2("http://abc.com:123/", rawKey)
		e := "http://abc.com:123/" + encodedKey
		if s != e {
			t.Fatalf("TestMakeURL: %q %q\n", s, e)
		}
	}
}
