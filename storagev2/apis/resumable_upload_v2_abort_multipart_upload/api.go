// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 根据 UploadId 终止 Multipart Upload
package resumable_upload_v2_abort_multipart_upload

import uptoken "github.com/qiniu/go-sdk/v7/storagev2/uptoken"

// 调用 API 所用的请求
type Request struct {
	BucketName string           // 存储空间名称
	ObjectName string           // 对象名称
	UploadId   string           // 在服务端申请的 Multipart Upload 任务 id
	UpToken    uptoken.Provider // 上传凭证，如果为空，则使用 HttpClientOptions 中的 UpToken
}

// 获取 API 所用的响应
type Response struct{}
