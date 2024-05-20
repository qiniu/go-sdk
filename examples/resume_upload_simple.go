package main

import (
	"context"
	"fmt"
	"os"

	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/storage"
)

var (
	accessKey = os.Getenv("QINIU_ACCESS_KEY")
	secretKey = os.Getenv("QINIU_SECRET_KEY")
	bucket    = os.Getenv("QINIU_TEST_BUCKET")
	localFile = os.Getenv("LOCAL_FILE")
	key       = os.Getenv("KEY")
)

func main() {
	putPolicy := storage.PutPolicy{
		Scope: bucket,
	}
	mac := auth.New(accessKey, secretKey)

	cfg := storage.Config{}
	// 是否使用https域名
	cfg.UseHTTPS = false
	// 上传是否使用CDN上传加速
	cfg.UseCdnDomains = false

	resumeUploader := storage.NewResumeUploader(&cfg)
	upToken := putPolicy.UploadToken(mac)

	ret := storage.PutRet{}

	err := resumeUploader.PutFile(context.Background(), &ret, upToken, key, localFile, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(ret.Key, ret.Hash)
}
