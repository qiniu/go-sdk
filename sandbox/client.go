package sandbox

import (
	"context"
	"net/http"

	"github.com/qiniu/go-sdk/v7/reqid"
	"github.com/qiniu/go-sdk/v7/sandbox/internal/apis"
)

// DefaultEndpoint 是沙箱 API 的默认服务地址。
const DefaultEndpoint = "https://cn-yangzhou-1-sandbox.qiniuapi.com"

// Config 是沙箱客户端的配置。
type Config struct {
	// APIKey 是用于身份认证的 API 密钥（必填）。
	APIKey string

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
		if id, ok := reqid.ReqidFromContext(ctx); ok {
			req.Header.Set("X-Reqid", id)
		}
		return nil
	}
}

// apiKeyEditor 返回一个 RequestEditorFn，用于注入 X-API-Key 请求头。
func apiKeyEditor(apiKey string) apis.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("X-API-Key", apiKey)
		return nil
	}
}
