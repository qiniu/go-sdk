package sandbox

import (
	"cmp"
	"context"
	"fmt"
	"net/http"

	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/sandbox/internal/apis"
)

// DefaultEndpoint 是沙箱 API 的默认服务地址。
const DefaultEndpoint = "https://cn-yangzhou-1-sandbox.qiniuapi.com"

// Config 是沙箱客户端的配置。
type Config struct {
	// APIKey 是用于身份认证的 API 密钥（必填）。
	APIKey string

	// Credentials 是用于身份认证的七牛凭证对象（可选）。
	// InjectionRule 相关接口会使用 Credentials 进行认证。
	Credentials *auth.Credentials

	// Endpoint 是沙箱 API 服务地址（可选，默认值：DefaultEndpoint）。
	Endpoint string

	// HTTPClient 自定义 HTTP 客户端（可选，默认值：http.DefaultClient）。
	HTTPClient *http.Client
}

// Client 是沙箱客户端。
type Client struct {
	config *Config
	api    apis.ClientWithResponsesInterface
}

// NewClient 创建一个新的沙箱客户端。
func NewClient(config *Config) (*Client, error) {
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	if config.HTTPClient == nil {
		config.HTTPClient = http.DefaultClient
	}

	opts := []apis.ClientOption{
		apis.WithHTTPClient(config.HTTPClient),
		apis.WithRequestEditorFn(reqidEditor()),
	}
	if config.APIKey != "" {
		opts = append(opts, apis.WithRequestEditorFn(apiKeyEditor(config.APIKey)))
	}

	client, err := apis.NewClientWithResponses(endpoint, opts...)
	if err != nil {
		return nil, err
	}

	return &Client{config: config, api: client}, nil
}

// reqidEditor 返回一个 RequestEditorFn，从 context 中提取 reqid 并注入 X-Reqid 请求头。
// 与 SDK 其他子产品（如 storage、media 等）的行为保持一致，方便统一链路追踪。
func reqidEditor() apis.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		setReqidHeader(ctx, req)
		return nil
	}
}

// apiKeyEditor 返回一个 RequestEditorFn，用于注入 API Key 认证头。
// 同时设置 X-API-Key 和 Authorization: Bearer，兼容不同端点对认证头的要求：
//   - 多数端点接受 X-API-Key
//   - 部分端点（如 POST /templates/{templateID} rebuild）要求 Authorization
//
// 与 E2B js-sdk (packages/js-sdk/src/api/index.ts) 的双 header 行为保持一致。
//
// 与 Credentials 共存：apiKeyEditor 作为 client-level editor 先于 per-request
// editor 执行；GetCredentialsOption 返回的 editor 会 Set 覆盖 Authorization
// 为 "Qiniu <sig>"，因此 Credentials 的优先级依旧生效，Bearer 只在未叠加
// Credentials editor 时才会命中。
func apiKeyEditor(apiKey string) apis.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		if req.Header.Get("Authorization") != "" {
			// 调用方已在请求前手动预设 Authorization（非常规路径），尊重其选择，
			// 同时不再追加 X-API-Key 以避免双认证头干扰。
			return nil
		}
		req.Header.Set("X-API-Key", apiKey)
		req.Header.Set("Authorization", "Bearer "+apiKey)
		return nil
	}
}

func (c *Client) GetCredentialsOption() (apis.RequestEditorFn, error) {
	cred := cmp.Or(c.config.Credentials, auth.Default())
	if cred == nil {
		return nil, fmt.Errorf("credentials not provided in client config")
	}
	return func(ctx context.Context, req *http.Request) error {
		token, err := cred.SignRequestV2(req)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", fmt.Sprintf("Qiniu %s", token))
		return nil
	}, nil
}
