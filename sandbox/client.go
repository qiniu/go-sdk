package sandbox

import (
	"context"
	"net/http"

	"github.com/qiniu/go-sdk/v7/sandbox/apis"
)

// DefaultEndpoint 是沙箱 API 的默认服务地址。
const DefaultEndpoint = "https://cn-yangzhou-1-sandbox.qiniuapi.com"

// DefaultDomain 是沙箱的默认域名后缀。
const DefaultDomain = "e2b.dev"

// Config 是沙箱客户端的配置。
type Config struct {
	// APIKey 是用于身份认证的 API 密钥（必填）。
	APIKey string

	// Endpoint 是沙箱 API 服务地址（可选，默认值：DefaultEndpoint）。
	Endpoint string

	// Domain 是沙箱运行时域名后缀（可选，默认值：DefaultDomain）。
	// 用于构造 envd agent 和端口访问的 URL。
	Domain string

	// HTTPClient 自定义 HTTP 客户端（可选，默认值：http.DefaultClient）。
	HTTPClient *http.Client
}

// Client 是沙箱 SDK 的高级客户端。
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

	opts := []apis.ClientOption{}
	if config.HTTPClient != nil {
		opts = append(opts, apis.WithHTTPClient(config.HTTPClient))
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

// apiKeyEditor 返回一个 RequestEditorFn，用于注入 X-API-Key 请求头。
func apiKeyEditor(apiKey string) apis.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("X-API-Key", apiKey)
		return nil
	}
}

// API 返回底层 API 客户端，用于直接访问生成的 API 方法。
func (c *Client) API() apis.ClientWithResponsesInterface {
	return c.api
}
