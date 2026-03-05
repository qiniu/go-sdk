// Package reqid 提供通过 Context 传递七牛请求 ID 的工具函数。
//
// 请求 ID 用于在七牛 API 调用中追踪请求链路，SDK 的 [client] 包会自动
// 从 Context 中提取请求 ID 并设置到 HTTP 请求头 X-Reqid。
//
// # 使用方式
//
//	ctx := reqid.WithReqid(ctx, "my-request-id")
//	// 后续使用该 ctx 发起的 API 调用会自动携带 X-Reqid 请求头
//
//	// 从 Context 中读取请求 ID
//	id, ok := reqid.ReqidFromContext(ctx)
package reqid
