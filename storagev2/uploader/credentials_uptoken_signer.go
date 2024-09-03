package uploader

import (
	"context"
	"sync"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"
)

type credentialsUpTokenSigner struct {
	credentials credentials.CredentialsProvider
	bucketName  string
	tokenTtl    time.Duration
	cacheTtl    time.Duration

	cacheMutux          sync.Mutex
	cachedPolicy        uptoken.PutPolicy
	cachedCredentials   *credentials.Credentials
	policyCachedAt      time.Time
	credentialsCachedAt time.Time
}

func (signer *credentialsUpTokenSigner) GetPutPolicy(ctx context.Context) (uptoken.PutPolicy, error) {
	var err error

	signer.cacheMutux.Lock()
	defer signer.cacheMutux.Unlock()

	now := time.Now()
	if signer.cachedPolicy == nil || signer.policyCachedAt.Add(signer.cacheTtl).Before(now) {
		signer.cachedPolicy, err = uptoken.NewPutPolicy(signer.bucketName, now.Add(signer.tokenTtl))
		if err != nil {
			return nil, err
		}
		signer.policyCachedAt = now
	}

	return signer.cachedPolicy, nil
}

func (signer *credentialsUpTokenSigner) GetAccessKey(ctx context.Context) (string, error) {
	cred, err := signer.getCredentials(ctx)
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
	cred, err := signer.getCredentials(ctx)
	if err != nil {
		return "", err
	}
	return uptoken.NewSigner(putPolicy, cred).GetUpToken(ctx)
}

func (signer *credentialsUpTokenSigner) getCredentials(ctx context.Context) (*credentials.Credentials, error) {
	var err error

	signer.cacheMutux.Lock()
	defer signer.cacheMutux.Unlock()

	now := time.Now()
	if signer.cachedCredentials == nil || signer.credentialsCachedAt.Add(signer.cacheTtl).Before(now) {
		signer.cachedCredentials, err = signer.credentials.Get(ctx)
		if err != nil {
			return nil, err
		}
		signer.credentialsCachedAt = now
	}

	return signer.cachedCredentials, nil
}

func newCredentialsUpTokenSigner(credentials credentials.CredentialsProvider, bucketName string, tokenTtl, cacheTtl time.Duration) uptoken.Provider {
	return &credentialsUpTokenSigner{credentials: credentials, bucketName: bucketName, tokenTtl: tokenTtl, cacheTtl: cacheTtl}
}
