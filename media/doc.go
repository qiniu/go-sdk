// Package media 提供七牛云多媒体数据处理的 Go 客户端。
//
// 通过 media/apis 子包的 [apis.NewMedia] 创建客户端，触发和查询持久化数据处理任务。
//
//	mediaClient := apis.NewMedia(&http_client.Options{Credentials: cred})
//
// 主要操作：
//
//   - Pfop: 触发持久化数据处理（如音视频转码、图片处理等）
//   - Prefop: 查询持久化数据处理任务状态
//
// 官方文档: https://developer.qiniu.com/dora
package media

//go:generate go run ../internal/api-generator -- --api-specs=../api-specs/media --output=apis/ --struct-name=Media --api-package=github.com/qiniu/go-sdk/v7/media/apis
//go:generate go build ./apis/...
