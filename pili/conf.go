package pili

import (
	"net/http"

	"github.com/qiniu/go-sdk/v7/client"
)

const (
	// APIHost 标准 API 服务器地址
	APIHost = "pili.qiniuapi.com"

	// IAMAPIHost IAM(权限策略) API 服务器地址
	IAMAPIHost = "pili-iam.qiniuapi.com"

	// APIHTTPScheme HTTP 模式
	APIHTTPScheme = "http://"

	// APIHTTPSScheme HTTPS 模式
	APIHTTPSScheme = "https://"

	// DefaultAppName 默认 AppName 名称
	DefaultAppName = "pili"
)

// ManagerConfig 构建Manager的参数配置
type ManagerConfig struct {
	// AppName 用户自定义APP名称
	// 命名规则遵循 [A-Za-z0-9_\ \-\.]*
	// AppName 将在发送HTTP请求的User-Agent中体现
	// 留空即使用默认值，默认 AppName 名称为 `pili`
	AppName string

	// APIHost 访问API的地址
	// 标准账户 和 IAM(权限策略)账户 对应的 Host 不同，注意区分
	// 留空即使用默认值，默认 APIHost 为 `pili.qiniuapi.com`
	APIHost string

	// APIHTTPScheme 访问API使用的HTTP模式
	// 支持 APIHTTPScheme / APIHTTPSScheme 两种模式
	// 留空即使用默认值，默认使用 `http://`
	APIHTTPScheme string

	// AccessKey 访问密钥
	// 密钥对由 AccessKey(访问密钥) 和 SecretKey(安全密钥) 组成
	// 每一个七牛账户最多拥有两对密钥，在七牛控制台 个人中心 - 密钥管理 中获取
	// 密钥对用于API鉴权，会在HTTP请求的 Header Authorization 中携带鉴权签算信息
	// 必填参数
	AccessKey string

	// SecretKey 安全密钥
	// 必填参数
	SecretKey string

	// Transport 访问控制
	// 支持外部传入自定义RoundTripper，用于HTTP代理/Context/超时控制等逻辑
	// 留空即使用默认值，默认使用 http.DefaultTransport
	Transport http.RoundTripper
}

// SetAppName 设置App名称
// 命名规则遵循 [A-Za-z0-9_\ \-\.]*
func SetAppName(appName string) {
	if appName == "" {
		appName = DefaultAppName
	}
	_ = client.SetAppName(appName)
}
