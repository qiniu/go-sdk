package sandbox

import (
	"context"
	"net/http"

	"github.com/qiniu/go-sdk/v7/sandbox/apis"
)

// DefaultEndpoint is the default sandbox API server address.
const DefaultEndpoint = "https://cn-yangzhou-1-sandbox.qiniuapi.com"

// Config is the configuration for the sandbox client.
type Config struct {
	// APIKey is the API key for authentication (required).
	APIKey string

	// Endpoint is the sandbox API server address (optional, default: DefaultEndpoint).
	Endpoint string

	// HTTPClient provides a custom HTTP client (optional, default: http.DefaultClient).
	HTTPClient *http.Client
}

// Client is the high-level sandbox SDK client.
type Client struct {
	config *Config
	api    *apis.ClientWithResponses
}

// NewClient creates a new sandbox client.
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

// apiKeyEditor returns a RequestEditorFn that injects the X-API-Key header.
func apiKeyEditor(apiKey string) apis.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("X-API-Key", apiKey)
		return nil
	}
}

// API returns the low-level API client for direct access.
func (c *Client) API() *apis.ClientWithResponses {
	return c.api
}
