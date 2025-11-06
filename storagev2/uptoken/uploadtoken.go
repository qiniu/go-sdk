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
		onceCredentials     sync.Once
		onceUpToken         sync.Once
		upToken             string
		upTokenErr          error
		credentials         *credentials.Credentials
	}
	parser struct {
		upToken        string
		oncePutPolicy  sync.Once
		putPolicy      PutPolicy
		putPolicyErr   error
		onceAccessKey  sync.Once
		accessKey      string
		accessKeyErr   error
		onceSplits     sync.Once
		splits         []string
		splitsValid    bool
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
	var err error
	credentials, err := signer.onceGetCredentials(ctx)
	if err != nil {
		return "", err
	}
	return credentials.AccessKey, nil
}

func (signer *signer) GetUpToken(ctx context.Context) (string, error) {
	return signer.onceGetUpToken(ctx)
}

func (signer *signer) onceGetCredentials(ctx context.Context) (*credentials.Credentials, error) {
	var err error
	signer.onceCredentials.Do(func() {
		if signer.credentialsProvider != nil {
			signer.credentials, err = signer.credentialsProvider.Get(ctx)
		} else if defaultCreds := credentials.Default(); defaultCreds != nil {
			signer.credentials = defaultCreds
		}
	})
	return signer.credentials, err
}

func (signer *signer) onceGetUpToken(ctx context.Context) (string, error) {
	signer.onceUpToken.Do(func() {
		var credentials *credentials.Credentials
		credentials, signer.upTokenErr = signer.onceGetCredentials(ctx)
		if signer.upTokenErr != nil {
			return
		}
		var putPolicyJson []byte
		putPolicyJson, signer.upTokenErr = json.Marshal(signer.putPolicy)
		if signer.upTokenErr != nil {
			return
		}
		signer.upToken = credentials.SignWithData(putPolicyJson)
	})
	return signer.upToken, signer.upTokenErr
}

// NewParser 创建上传凭证签发器
func NewParser(upToken string) Provider {
	return &parser{upToken: upToken}
}

func (parser *parser) GetPutPolicy(context.Context) (PutPolicy, error) {
	parser.oncePutPolicy.Do(func() {
		splits, ok := parser.onceGetSplits()
		if !ok {
			parser.putPolicyErr = ErrInvalidUpToken
			return
		}
		var putPolicyJson []byte
		putPolicyJson, parser.putPolicyErr = base64.URLEncoding.DecodeString(splits[2])
		if parser.putPolicyErr != nil {
			parser.putPolicyErr = ErrInvalidUpToken
			return
		}
		parser.putPolicyErr = json.Unmarshal(putPolicyJson, &parser.putPolicy)
	})
	return parser.putPolicy, parser.putPolicyErr
}

func (parser *parser) GetAccessKey(context.Context) (string, error) {
	parser.onceAccessKey.Do(func() {
		splits, ok := parser.onceGetSplits()
		if !ok {
			parser.accessKeyErr = ErrInvalidUpToken
			return
		}
		parser.accessKey = splits[0]
	})
	return parser.accessKey, parser.accessKeyErr
}

func (parser *parser) onceGetSplits() ([]string, bool) {
	parser.onceSplits.Do(func() {
		splits := strings.Split(parser.upToken, ":")
		if len(splits) == 5 && splits[0] == "" {
			splits = splits[2:]
		}
		if len(splits) != 3 {
			parser.splitsValid = false
			return
		}
		parser.splits = splits
		parser.splitsValid = true
	})
	return parser.splits, parser.splitsValid
}

func (parser *parser) GetUpToken(context.Context) (string, error) {
	return parser.upToken, nil
}
