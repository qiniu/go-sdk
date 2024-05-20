package credentials

import (
	"context"
	"errors"
	"os"

	"github.com/qiniu/go-sdk/v7/auth"
)

// Credentials 七牛鉴权类，用于生成Qbox, Qiniu, Upload签名
//
// AK/SK可以从 https://portal.qiniu.com/user/key 获取
type Credentials = auth.Credentials

// NewCredentials 构建一个 Credentials 对象
func NewCredentials(accessKey, secretKey string) *Credentials {
	return auth.New(accessKey, secretKey)
}

// Default 构建一个 Credentials 对象
func Default() *Credentials {
	return auth.Default()
}

// CredentialsProvider 获取 Credentials 对象的接口
type CredentialsProvider interface {
	Get(context.Context) (*Credentials, error)
}

// EnvironmentVariableCredentialProvider 从环境变量中获取 Credential
type EnvironmentVariableCredentialProvider struct{}

func (provider *EnvironmentVariableCredentialProvider) Get(ctx context.Context) (credential *Credentials, err error) {
	accessKey := os.Getenv("QINIU_ACCESS_KEY")
	secretKey := os.Getenv("QINIU_SECRET_KEY")
	if accessKey == "" {
		return nil, errors.New("QINIU_ACCESS_KEY is not set")
	}
	if secretKey == "" {
		return nil, errors.New("QINIU_SECRET_KEY is not set")
	}
	return NewCredentials(accessKey, secretKey), nil
}

var _ CredentialsProvider = (*EnvironmentVariableCredentialProvider)(nil)

// ChainedCredentialsProvider 存储多个 CredentialsProvider，逐个尝试直到成功获取第一个 Credentials 为止
type ChainedCredentialsProvider struct {
	providers []CredentialsProvider
}

func (provider *ChainedCredentialsProvider) Get(ctx context.Context) (credential *Credentials, err error) {
	for _, provider := range provider.providers {
		if credential, err = provider.Get(ctx); err == nil {
			return
		}
	}
	return
}

var _ CredentialsProvider = (*ChainedCredentialsProvider)(nil)
