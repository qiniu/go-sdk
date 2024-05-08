package uploader

import (
	"context"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"
)

type credentialsUpTokenSigner struct {
	credentials credentials.CredentialsProvider
	bucketName  string
	ttl         time.Duration
}

func (signer *credentialsUpTokenSigner) GetPutPolicy(ctx context.Context) (uptoken.PutPolicy, error) {
	return uptoken.NewPutPolicy(signer.bucketName, time.Now().Add(signer.ttl))
}

func (signer *credentialsUpTokenSigner) GetAccessKey(ctx context.Context) (string, error) {
	cred, err := signer.credentials.Get(ctx)
	if err != nil {
		return "", err
	}
	return cred.AccessKey, nil
}

func (signer *credentialsUpTokenSigner) GetUpToken(ctx context.Context) (string, error) {
	putPolicy, err := signer.GetPutPolicy(ctx)
	if err != nil {
		return "", err
	}
	return uptoken.NewSigner(putPolicy, signer.credentials).GetUpToken(ctx)
}

func NewCredentialsUpTokenSigner(credentials credentials.CredentialsProvider, bucketName string, ttl time.Duration) uptoken.Provider {
	return &credentialsUpTokenSigner{credentials, bucketName, ttl}
}
