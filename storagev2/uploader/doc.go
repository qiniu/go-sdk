// Package uploader 提供七牛云对象存储的上传管理。
//
// [UploadManager] 是推荐的上传入口，根据文件大小自动选择表单上传或分片上传：
//
//   - 小于 MultiPartsThreshold（默认 4MB）：使用表单上传
//   - 大于等于该阈值：使用分片上传（支持 v1 和 v2 两种协议）
//
// # 基本用法
//
//	uploadManager := uploader.NewUploadManager(&uploader.UploadManagerOptions{
//	    Options: http_client.Options{Credentials: cred},
//	})
//
//	objectName := "my-file.txt"
//	err := uploadManager.UploadFile(ctx, "/path/to/file", &uploader.ObjectOptions{
//	    BucketName: "my-bucket",
//	    ObjectName: &objectName,
//	    UpToken:    uptoken.NewSigner(putPolicy, cred),
//	}, nil)
//
// # 上传 Reader
//
//	err := uploadManager.UploadReader(ctx, reader, &uploader.ObjectOptions{
//	    BucketName: "my-bucket",
//	    ObjectName: &objectName,
//	    UpToken:    uptoken.NewSigner(putPolicy, cred),
//	}, nil)
//
// # 上传目录
//
//	err := uploadManager.UploadDirectory(ctx, "/path/to/dir", &uploader.DirectoryOptions{
//	    BucketName: "my-bucket",
//	    UpToken:    uptoken.NewSigner(putPolicy, cred),
//	})
//
// # 上传进度
//
// 通过 [ObjectOptions.OnUploadingProgress] 回调获取上传进度：
//
//	opts := &uploader.ObjectOptions{
//	    OnUploadingProgress: func(progress *uploader.UploadingProgress) {
//	        fmt.Printf("%d / %d\n", progress.Uploaded, progress.TotalSize)
//	    },
//	}
//
// # 底层接口
//
// 如需精细控制，可直接使用：
//
//   - [NewFormUploader]: 表单上传
//   - [NewMultiPartsUploaderV1] / [NewMultiPartsUploaderV2]: 分片上传
package uploader
