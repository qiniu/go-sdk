// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 一键删除指定存储空间的所有标签
package delete_bucket_taggings

import credentials "github.com/qiniu/go-sdk/v7/storagev2/credentials"

// 调用 API 所用的请求
type Request struct {
	BucketName  string                          // 空间名称
	Credentials credentials.CredentialsProvider // 鉴权参数，用于生成鉴权凭证，如果为空，则使用 HttpClientOptions 中的 CredentialsProvider
}

// 获取 API 所用的响应
type Response struct{}
