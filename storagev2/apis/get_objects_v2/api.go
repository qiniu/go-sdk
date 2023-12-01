// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 列举指定存储空间里的所有对象条目
package get_objects_v2

import (
	credentials "github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"io"
)

// 调用 API 所用的请求
type Request struct {
	Bucket      string                          // 指定存储空间
	Marker      string                          // 上一次列举返回的位置标记，作为本次列举的起点信息
	Limit       int64                           // 本次列举的条目数，范围为 1-1000
	Prefix      string                          // 指定前缀，只有资源名匹配该前缀的资源会被列出
	Delimiter   string                          // 指定目录分隔符，列出所有公共前缀（模拟列出目录效果）
	NeedParts   bool                            // 如果文件是通过分片上传的，是否返回对应的分片信息
	Credentials credentials.CredentialsProvider // 鉴权参数，用于生成鉴权凭证，如果为空，则使用 HttpClientOptions 中的 CredentialsProvider
}

// 获取 API 所用的响应
type Response struct {
	Body io.ReadCloser
}
