// Package credentials 提供 storagev2 体系的凭证管理。
//
// [Credentials] 是 [auth.Credentials] 的类型别名，表示 AK/SK 认证凭证。
// [CredentialsProvider] 接口是 storagev2 中传递凭证的标准方式，
// 上传、下载、对象管理等模块均通过该接口获取凭证。
//
// # 创建凭证
//
//	cred := credentials.NewCredentials("AccessKey", "SecretKey")
//
//	// 从环境变量加载（QINIU_ACCESS_KEY、QINIU_SECRET_KEY）
//	cred := credentials.Default()
//
// # CredentialsProvider 接口
//
//	type CredentialsProvider interface {
//	    Get(context.Context) (*Credentials, error)
//	}
//
// [*Credentials] 本身实现了该接口，可直接作为 Provider 使用。
// 还可以使用 [EnvironmentVariableCredentialProvider] 从环境变量读取，
// 或 [ChainedCredentialsProvider] 组合多个 Provider 按顺序尝试。
package credentials
