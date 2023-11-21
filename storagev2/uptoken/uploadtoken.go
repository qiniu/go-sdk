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
	// PutPolicyRetriever 获取上传策略接口
	PutPolicyRetriever interface {
		RetrievePutPolicy(context.Context) (PutPolicy, error)
	}
	// AccessKeyRetriever 获取 AccessKey 接口
	AccessKeyRetriever interface {
		RetrieveAccessKey(context.Context) (string, error)
	}
	// Retriever 获取上传凭证接口
	Retriever interface {
		RetrieveUpToken(context.Context) (string, error)
	}
	// Provider 获取上传凭证，AccessKey 和上传策略接口
	Provider interface {
		PutPolicyRetriever
		AccessKeyRetriever
		Retriever
	}
	// Signer 上传凭证签发器
	Signer struct {
		putPolicy           PutPolicy
		credentialsProvider credentials.CredentialsProvider
	}
	// Parser 上传凭证解析器
	Parser struct {
		upToken   string
		putPolicy PutPolicy
		accessKey string
		splits    []string
	}
)

// NewSigner 创建上传凭证签发器
func NewSigner(putPolicy PutPolicy, credentialsProvider credentials.CredentialsProvider) *Signer {
	return &Signer{putPolicy: putPolicy, credentialsProvider: credentialsProvider}
}

func (signer *Signer) RetrievePutPolicy(context.Context) (PutPolicy, error) {
	return signer.putPolicy, nil
}

func (signer *Signer) RetrieveAccessKey(ctx context.Context) (string, error) {
	credentials, err := signer.credentialsProvider.Get(ctx)
	if err != nil {
		return "", err
	}
	return credentials.AccessKey, nil
}

func (signer *Signer) RetrieveUpToken(ctx context.Context) (string, error) {
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

var _ Provider = (*Signer)(nil)

// NewParser 创建上传凭证签发器
func NewParser(upToken string) *Parser {
	return &Parser{upToken: upToken}
}

func (parser *Parser) RetrievePutPolicy(context.Context) (PutPolicy, error) {
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

func (parser *Parser) RetrieveAccessKey(context.Context) (string, error) {
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

func (parser *Parser) retrieveSplits() []string {
	if len(parser.splits) > 0 {
		return parser.splits
	}
	parser.splits = strings.SplitN(parser.upToken, ":", 3)
	return parser.splits
}

func (parser *Parser) RetrieveUpToken(context.Context) (string, error) {
	return parser.upToken, nil
}

var _ Provider = (*Parser)(nil)