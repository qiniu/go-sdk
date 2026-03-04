// Package storagev2 提供七牛云对象存储 v2 API 的 Go 客户端。
//
// 相比 storage 包（v1），storagev2 提供了类型化的请求/响应、自动区域检测、
// 连接池、重试和 Provider/Interface 模式，推荐新项目使用。
//
// # 子包结构
//
//   - storagev2/credentials: 凭证管理，[credentials.NewCredentials] 创建凭证
//   - storagev2/uploader: 上传管理，[uploader.NewUploadManager] 自动选择上传方式
//   - storagev2/downloader: 下载管理，[downloader.NewDownloadManager]
//   - storagev2/objects: 对象管理，[objects.NewObjectsManager] 提供流式 API
//   - storagev2/uptoken: 上传凭证，[uptoken.NewPutPolicy] 创建上传策略
//   - storagev2/apis: 低级 API 客户端，[apis.NewStorage] 提供所有类型化 API 方法
//   - storagev2/region: 区域信息，RegionsProvider 接口
//   - storagev2/http_client: HTTP 客户端选项
//
// # 快速开始
//
//	cred := credentials.NewCredentials("AccessKey", "SecretKey")
//	putPolicy, _ := uptoken.NewPutPolicy("bucket", time.Now().Add(time.Hour))
//
//	uploadManager := uploader.NewUploadManager(&uploader.UploadManagerOptions{
//	    Options: http_client.Options{Credentials: cred},
//	})
//
//	objectName := "my-file.txt"
//	err := uploadManager.UploadFile(ctx, "/path/to/file", &uploader.ObjectOptions{
//	    BucketName: "bucket",
//	    ObjectName: &objectName,
//	    UpToken:    uptoken.NewSigner(putPolicy, cred),
//	}, nil)
package storagev2

//go:generate go run ../internal/api-generator -- --api-specs=../api-specs/storage --api-specs=internal/api-specs --output=apis/ --struct-name=Storage --api-package=github.com/qiniu/go-sdk/v7/storagev2/apis
//go:generate go build ./apis/...
