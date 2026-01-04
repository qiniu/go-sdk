package uptoken

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"sync"

	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
)

// ErrInvalidUpToken 非法的上传凭证
var ErrInvalidUpToken = errors.New("invalid upToken")

type (
	// PutPolicyProvider 获取上传策略接口
	PutPolicyProvider interface {
		GetPutPolicy(context.Context) (PutPolicy, error)
	}
	// AccessKeyProvider 获取 AccessKey 接口
	AccessKeyProvider interface {
		GetAccessKey(context.Context) (string, error)
	}
	// UpTokenProvider 获取上传凭证接口
	UpTokenProvider interface {
		GetUpToken(context.Context) (string, error)
	}
	// Provider 获取上传凭证，AccessKey 和上传策略接口
	Provider interface {
		PutPolicyProvider
		AccessKeyProvider
		UpTokenProvider
	}
	signer struct {
		putPolicy           PutPolicy
		credentialsProvider credentials.CredentialsProvider
		mu                  sync.Mutex
		credentials         *credentials.Credentials
		upToken             string
	}
	parser struct {
		upToken   string
		mu        sync.Mutex
		putPolicy PutPolicy
		accessKey string
		splits    []string
	}
)

// NewSigner 创建上传凭证签发器
//
// 需要注意的是 NewSigner 仅仅只会通过 credentials.CredentialsProvider 获取一次鉴权参数，之后就会缓存该鉴权参数，不会反复获取
func NewSigner(putPolicy PutPolicy, credentialsProvider credentials.CredentialsProvider) Provider {
	return &signer{putPolicy: putPolicy, credentialsProvider: credentialsProvider}
}

func (signer *signer) GetPutPolicy(context.Context) (PutPolicy, error) {
	return signer.putPolicy, nil
}

func (signer *signer) GetAccessKey(ctx context.Context) (string, error) {
	credentials, err := signer.getCredentials(ctx)
	if err != nil {
		return "", err
	}
	return credentials.AccessKey, nil
}

func (signer *signer) GetUpToken(ctx context.Context) (string, error) {
	signer.mu.Lock()
	if signer.upToken != "" {
		defer signer.mu.Unlock()
		return signer.upToken, nil
	}
	signer.mu.Unlock()

	credentials, err := signer.getCredentials(ctx)
	if err != nil {
		return "", err
	}

	putPolicyJson, err := json.Marshal(signer.putPolicy)
	if err != nil {
		return "", err
	}

	upToken := credentials.SignWithData(putPolicyJson)

	signer.mu.Lock()
	if signer.upToken == "" {
		signer.upToken = upToken
	}
	signer.mu.Unlock()

	return signer.upToken, nil
}

func (signer *signer) getCredentials(ctx context.Context) (*credentials.Credentials, error) {
	signer.mu.Lock()
	if signer.credentials != nil {
		defer signer.mu.Unlock()
		return signer.credentials, nil
	}
	signer.mu.Unlock()

	var creds *credentials.Credentials
	var err error
	if signer.credentialsProvider != nil {
		creds, err = signer.credentialsProvider.Get(ctx)
		if err != nil {
			return nil, err
		}
	} else if defaultCreds := credentials.Default(); defaultCreds != nil {
		creds = defaultCreds
	}

	if creds != nil {
		signer.mu.Lock()
		if signer.credentials == nil {
			signer.credentials = creds
		}
		signer.mu.Unlock()
	}

	return creds, nil
}

// NewParser 创建上传凭证签发器
func NewParser(upToken string) Provider {
	return &parser{upToken: upToken}
}

func (parser *parser) GetPutPolicy(ctx context.Context) (PutPolicy, error) {
	parser.mu.Lock()
	if parser.putPolicy != nil {
		defer parser.mu.Unlock()
		return parser.putPolicy, nil
	}
	parser.mu.Unlock()

	splits, err := parser.getSplits()
	if err != nil {
		return PutPolicy{}, err
	}

	putPolicyJson, err := base64.URLEncoding.DecodeString(splits[2])
	if err != nil {
		return PutPolicy{}, ErrInvalidUpToken
	}

	var putPolicy PutPolicy
	if err := json.Unmarshal(putPolicyJson, &putPolicy); err != nil {
		return PutPolicy{}, err
	}

	parser.mu.Lock()
	if parser.putPolicy == nil {
		parser.putPolicy = putPolicy
	}
	parser.mu.Unlock()

	return parser.putPolicy, nil
}

func (parser *parser) GetAccessKey(ctx context.Context) (string, error) {
	parser.mu.Lock()
	if parser.accessKey != "" {
		defer parser.mu.Unlock()
		return parser.accessKey, nil
	}
	parser.mu.Unlock()

	splits, err := parser.getSplits()
	if err != nil {
		return "", err
	}

	accessKey := splits[0]

	parser.mu.Lock()
	if parser.accessKey == "" {
		parser.accessKey = accessKey
	}
	parser.mu.Unlock()

	return parser.accessKey, nil
}

func (parser *parser) getSplits() ([]string, error) {
	parser.mu.Lock()
	if len(parser.splits) > 0 {
		defer parser.mu.Unlock()
		return parser.splits, nil
	}
	parser.mu.Unlock()

	splits := strings.Split(parser.upToken, ":")
	if len(splits) == 5 && splits[0] == "" {
		splits = splits[2:]
	}
	if len(splits) != 3 {
		return nil, ErrInvalidUpToken
	}

	parser.mu.Lock()
	if len(parser.splits) == 0 {
		parser.splits = splits
	}
	parser.mu.Unlock()

	return parser.splits, nil
}

func (parser *parser) GetUpToken(context.Context) (string, error) {
	return parser.upToken, nil
}
