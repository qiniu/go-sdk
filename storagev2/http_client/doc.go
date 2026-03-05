// Package http_client 提供 storagev2 体系的 HTTP 客户端和通用选项。
//
// [Options] 结构体是 storagev2 中配置 HTTP 行为的核心类型，
// 被 uploader、downloader、objects 等高级模块通过嵌入方式使用。
//
// # Options 配置
//
// [Options] 包含凭证、区域、重试、拦截器等配置：
//
//	opts := http_client.Options{
//	    Credentials:       cred,       // 认证凭证
//	    Regions:           provider,   // 区域信息
//	    UseInsecureProtocol: false,    // 是否使用 HTTP（默认 HTTPS）
//	}
//
// 高级模块通常嵌入此结构体：
//
//	uploadManager := uploader.NewUploadManager(&uploader.UploadManagerOptions{
//	    Options: http_client.Options{Credentials: cred},
//	})
//
// # HTTP 客户端
//
// [Client] 封装了带重试、区域路由和认证签名的 HTTP 客户端：
//
//	client := http_client.NewClient(&opts)
//	resp, err := client.Do(ctx, &http_client.Request{
//	    Method:       "GET",
//	    ServiceNames: []region.ServiceName{region.ServiceRs},
//	    Path:         "/stat/...",
//	    Credentials:  cred,
//	    AuthType:     auth.TokenQiniu,
//	})
//
// # 请求体构建
//
//   - [GetJsonRequestBody]: JSON 格式请求体
//   - [GetFormRequestBody]: 表单格式请求体
//   - [GetMultipartFormRequestBody]: Multipart 表单请求体
package http_client
