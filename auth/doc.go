// Package auth 提供七牛云 API 的认证和签名功能。
//
// 通过 AK/SK（AccessKey/SecretKey）创建凭证，用于对 API 请求进行签名认证。
// AK/SK 可从 https://portal.qiniu.com/user/key 获取。
//
// # 创建凭证
//
//	mac := auth.New("AccessKey", "SecretKey")
//
//	// 或从环境变量加载（QINIU_ACCESS_KEY、QINIU_SECRET_KEY）
//	mac := auth.Default()
//
// # 签名方式
//
// 支持多种签名方式：
//
//   - [Credentials.Sign]: HMAC-SHA1 签名
//   - [Credentials.SignRequest]: Qbox 格式请求签名
//   - [Credentials.SignRequestV2]: Qiniu V2 格式请求签名
//   - [Credentials.SignWithData]: 签名并包含编码数据（用于上传凭证）
//   - [Credentials.VerifyCallback]: 验证回调请求签名
//
// # Context 传递凭证
//
// 支持通过 Context 传递凭证信息，适用于需要在调用链中传递认证的场景：
//
//	ctx = auth.WithCredentialsType(ctx, mac, auth.TokenQiniu)
//	cred, tokenType, ok := auth.CredentialsFromContext(ctx)
package auth
