package main

import (
	"fmt"
	"os"

	"context"

	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/storage"
)

var (
	accessKey = os.Getenv("QINIU_ACCESS_KEY")
	secretKey = os.Getenv("QINIU_SECRET_KEY")
	bucket    = os.Getenv("QINIU_TEST_BUCKET")
	localFile = os.Getenv("LOCAL_FILE")
	key       = os.Getenv("KEY")

	// 指定的进度文件保存目录，实际情况下，请确保该目录存在，而且只用于记录进度文件
	recordDir = os.Getenv("RECORD_DIR")
)

func main() {
	putPolicy := storage.PutPolicy{
		Scope: bucket,
	}
	mac := auth.New(accessKey, secretKey)
	upToken := putPolicy.UploadToken(mac)

	cfg := storage.Config{}
	// 是否使用https域名
	cfg.UseHTTPS = false
	// 上传是否使用CDN上传加速
	cfg.UseCdnDomains = false

	mErr := os.MkdirAll(recordDir, 0755)
	if mErr != nil {
		fmt.Println("mkdir for record dir error,", mErr)
		return
	}

	resumeUploader := storage.NewResumeUploader(&cfg)
	ret := storage.PutRet{}

	recorder, err := storage.NewFileRecorder(recordDir)
	if err != nil {
		fmt.Println(err)
		return
	}
	if err = resumeUploader.PutFile(context.Background(), &ret, upToken, key, localFile, &storage.RputExtra{
		Recorder: recorder,
	}); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(ret.Key, ret.Hash)
}
