package storage

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/qiniu/go-sdk/v7/auth"
)

func TestForceSaveKeyFalse(t *testing.T) {
	p := PutPolicy{}
	pj, _ := json.Marshal(p)
	s := string(pj)
	t.Log(s)
	if strings.Contains(s, "forceSaveKey") {
		t.Fail()
	}
}

func TestForceSaveKeyTrue(t *testing.T) {
	p := PutPolicy{}
	p.ForceSaveKey = true
	pj, _ := json.Marshal(p)
	s := string(pj)
	t.Log(s)
	if !strings.Contains(s, "forceSaveKey") {
		t.Fail()
	}
}

func TestGetAkBucketFromUploadToken(t *testing.T) {
	bucketName := "fakebucket"
	keyName := "fakekey"
	accessKey := "fakeaccesskey"
	secretKey := []byte("fakesecretkey")
	policy := PutPolicy{Scope: fmt.Sprintf("%s:%s", bucketName, keyName), Expires: uint64(time.Now().Unix()) + 24*3600}
	cred := auth.Credentials{AccessKey: accessKey, SecretKey: secretKey}
	token := policy.UploadToken(&cred)

	ak, bucket, err := getAkBucketFromUploadToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if ak != accessKey || bucket != bucketName {
		t.Fail()
	}

	b, _ := json.Marshal(policy)
	data := base64.URLEncoding.EncodeToString(b)
	suInfo := ":12345/0:"
	hash := hmac.New(sha1.New, secretKey)
	hash.Write([]byte(suInfo))
	hash.Write([]byte(data))
	sign := base64.URLEncoding.EncodeToString(hash.Sum(nil))
	suToken := fmt.Sprintf("%s%s:%s:%s", suInfo, accessKey, sign, data)
	ak, bucket, err = getAkBucketFromUploadToken(suToken)
	if err != nil {
		t.Fatal(err)
	}
	if ak != accessKey || bucket != bucketName {
		t.Fail()
	}
}
