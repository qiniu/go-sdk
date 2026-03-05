// Package uptoken 提供七牛云对象存储的上传凭证管理。
//
// 上传凭证（Upload Token）用于授权客户端上传文件到指定存储空间。
// 通过 [PutPolicy] 定义上传策略，再用 [NewSigner] 生成签名后的凭证。
//
// # 创建上传策略
//
//	// 允许上传到整个存储空间
//	putPolicy, err := uptoken.NewPutPolicy("my-bucket", time.Now().Add(time.Hour))
//
//	// 仅允许上传到指定 key
//	putPolicy, err := uptoken.NewPutPolicyWithKey("my-bucket", "my-file.txt", time.Now().Add(time.Hour))
//
//	// 仅允许上传到指定前缀
//	putPolicy, err := uptoken.NewPutPolicyWithKeyPrefix("my-bucket", "images/", time.Now().Add(time.Hour))
//
// # 生成上传凭证
//
//	signer := uptoken.NewSigner(putPolicy, cred)
//	// signer 实现了 Provider 接口，可传递给 uploader 使用
//
// # 上传策略选项
//
// [PutPolicy] 支持丰富的配置，通过 Get/Set 方法访问：
//
//   - SetInsertOnly: 仅允许新增（禁止覆盖）
//   - SetCallbackUrl / SetCallbackBody: 设置上传回调
//   - SetReturnBody: 自定义返回内容
//   - SetPersistentOps: 设置持久化数据处理
//   - SetFsizeLimit / SetFsizeMin: 限制文件大小
//   - SetMimeLimit: 限制文件 MIME 类型
//   - SetFileType: 指定存储类型
//
// # 解析已有凭证
//
//	provider := uptoken.NewParser("existing-upload-token-string")
//	policy, err := provider.GetPutPolicy(ctx)
//
// # Provider 接口
//
// [Provider] 统一了上传凭证的获取方式，组合了三个子接口：
//
//   - [UpTokenProvider]: 获取上传凭证字符串
//   - [PutPolicyProvider]: 获取上传策略
//   - [AccessKeyProvider]: 获取 AccessKey
package uptoken
