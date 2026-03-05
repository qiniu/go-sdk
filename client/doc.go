// Package client 提供七牛云 API 的底层 HTTP 客户端。
//
// [Client] 封装了标准库 [net/http.Client]，自动处理 User-Agent 设置、
// 请求 ID 注入（X-Reqid）、认证签名、JSON/表单序列化和错误解析。
//
// 高级用户通常使用 storage 或 storagev2 包，不需要直接使用本包。
//
// # 使用默认客户端
//
//	var ret MyResponse
//	err := client.DefaultClient.CallWithJson(ctx, &ret, "POST", url, nil, body)
//
// # 错误处理
//
// API 错误以 [*ErrorInfo] 返回，包含 HTTP 状态码、错误码和请求 ID：
//
//	err := client.DefaultClient.Call(ctx, &ret, "GET", url, nil)
//	if e, ok := err.(*client.ErrorInfo); ok {
//	    fmt.Println(e.Code, e.Err, e.Reqid)
//	}
//
// # 调试模式
//
//	client.TurnOnDebug()           // 输出 HTTP 请求/响应详情
//	client.DeepDebugInfo = true    // 同时输出请求/响应 body
//
// # DNS 预解析
//
// 通过 Context 注入预解析的 IP 地址，跳过 DNS 查询：
//
//	ctx = client.WithResolvedIPs(ctx, "up.qiniup.com", []net.IP{ip})
package client
