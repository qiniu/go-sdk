package pili

import (
	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/client"
)

// Manager 提供了 Qiniu PILI Service API 相关功能
type Manager struct {
	apiHost       string
	apiHTTPScheme string
	client        *client.Client
	mac           *auth.Credentials
}

// NewManager 用于构建一个新的 Manager
func NewManager(conf ManagerConfig) *Manager {
	if len(conf.APIHost) == 0 {
		conf.APIHost = APIHost
	}
	if len(conf.APIHTTPScheme) == 0 {
		conf.APIHTTPScheme = APIHTTPScheme
	}
	SetAppName(conf.AppName)
	mac := auth.New(conf.AccessKey, conf.SecretKey)
	client := client.DefaultClient
	client.Transport = newTransport(mac, conf.Transport)
	return &Manager{
		apiHost:       conf.APIHost,
		apiHTTPScheme: conf.APIHTTPScheme,
		mac:           mac,
		client:        &client,
	}
}
