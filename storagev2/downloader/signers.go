package downloader

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
)

type credentialsSigner struct {
	credentials credentials.CredentialsProvider
}

// 创建基于七牛鉴权的下载 URL 签名
func NewCredentialsSigner(credentials credentials.CredentialsProvider) Signer {
	return &credentialsSigner{credentials}
}

func (signer credentialsSigner) Sign(ctx context.Context, u *url.URL, options *SignOptions) error {
	if options == nil {
		options = &SignOptions{}
	}
	ttl := options.TTL
	if ttl == 0 {
		ttl = 3 * time.Minute
	}

	cred, err := signer.credentials.Get(ctx)
	if err != nil {
		return err
	}
	u.RawQuery += signURL(u.String(), cred, time.Now().Add(ttl).Unix())
	return nil
}

func signURL(url string, cred *credentials.Credentials, deadline int64) string {
	var appendUrl string

	if isURLSigned(url) {
		return ""
	}

	urlToSign := url
	if strings.Contains(url, "?") {
		appendUrl = fmt.Sprintf("&e=%d", deadline)
		urlToSign += appendUrl
	} else {
		appendUrl = fmt.Sprintf("e=%d", deadline)
		urlToSign += "?"
		urlToSign += appendUrl
	}
	token := cred.Sign([]byte(urlToSign))
	return fmt.Sprintf("%s&token=%s", appendUrl, token)
}

func isURLSigned(url string) bool {
	return (strings.Contains(url, "&e=") || strings.Contains(url, "?e=")) &&
		strings.Contains(url, "&token=")
}
