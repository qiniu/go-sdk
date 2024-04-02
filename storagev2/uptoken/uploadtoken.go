package uptoken

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
)

// ErrInvalidUpToken 非法的上传凭证
var ErrInvalidUpToken = errors.New("invalid UpToken")

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
	}
	parser struct {
		upToken   string
		putPolicy PutPolicy
		accessKey string
		splits    []string
	}
)

// NewSigner 创建上传凭证签发器
func NewSigner(putPolicy PutPolicy, credentialsProvider credentials.CredentialsProvider) Provider {
	return &signer{putPolicy: putPolicy, credentialsProvider: credentialsProvider}
}

func (signer *signer) GetPutPolicy(context.Context) (PutPolicy, error) {
	return signer.putPolicy, nil
}

func (signer *signer) GetAccessKey(ctx context.Context) (string, error) {
	credentials, err := signer.credentialsProvider.Get(ctx)
	if err != nil {
		return "", err
	}
	return credentials.AccessKey, nil
}

func (signer *signer) GetUpToken(ctx context.Context) (string, error) {
	credentials, err := signer.credentialsProvider.Get(ctx)
	if err != nil {
		return "", err
	}
	putPolicyJson, err := json.Marshal(signer.putPolicy)
	if err != nil {
		return "", err
	}
	return credentials.SignWithData(putPolicyJson), nil
}

// NewParser 创建上传凭证签发器
func NewParser(upToken string) Provider {
	return &parser{upToken: upToken}
}

func (parser *parser) GetPutPolicy(context.Context) (PutPolicy, error) {
	if parser.putPolicy != nil {
		return parser.putPolicy, nil
	}
	splits := parser.retrieveSplits()
	if len(splits) != 3 {
		return nil, ErrInvalidUpToken
	}
	putPolicyJson, err := base64.URLEncoding.DecodeString(splits[2])
	if err != nil {
		return nil, ErrInvalidUpToken
	}
	err = json.Unmarshal(putPolicyJson, &parser.putPolicy)
	return parser.putPolicy, err
}

func (parser *parser) GetAccessKey(context.Context) (string, error) {
	if parser.accessKey != "" {
		return parser.accessKey, nil
	}
	splits := parser.retrieveSplits()
	if len(splits) != 3 {
		return "", ErrInvalidUpToken
	}
	parser.accessKey = splits[0]
	return parser.accessKey, nil
}

func (parser *parser) retrieveSplits() []string {
	if len(parser.splits) > 0 {
		return parser.splits
	}
	parser.splits = strings.SplitN(parser.upToken, ":", 3)
	return parser.splits
}

func (parser *parser) GetUpToken(context.Context) (string, error) {
	return parser.upToken, nil
}
