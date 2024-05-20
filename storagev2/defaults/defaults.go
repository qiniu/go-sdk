package defaults

import (
	"strings"

	"github.com/qiniu/go-sdk/v7/internal/configfile"
	"github.com/qiniu/go-sdk/v7/internal/env"
)

func Credentials() (string, string, error) {
	accessKey, secretKey := env.CredentialsFromEnvironment()
	if accessKey != "" && secretKey != "" {
		return accessKey, secretKey, nil
	}
	accessKey, secretKey, err := configfile.CredentialsFromConfigFile()
	if err != nil {
		return "", "", err
	}
	if accessKey != "" && secretKey != "" {
		return accessKey, secretKey, nil
	}
	return "", "", nil
}

func BucketURLs() ([]string, error) {
	normalizeBucketUrl := func(bucketUrl string) string {
		if !strings.Contains(bucketUrl, "://") {
			var (
				isDisabled bool
				err        error
			)
			if isDisabled, err = DisableSecureProtocol(); err != nil {
				isDisabled = false
			}
			if isDisabled {
				bucketUrl = "http://" + bucketUrl
			} else {
				bucketUrl = "https://" + bucketUrl
			}
		}
		return bucketUrl
	}

	normalizeBucketUrls := func(bucketUrls []string) []string {
		normalizedBucketUrls := make([]string, 0, len(bucketUrls))
		for _, bucketUrl := range bucketUrls {
			normalizedBucketUrls = append(normalizedBucketUrls, normalizeBucketUrl(bucketUrl))
		}
		return normalizedBucketUrls
	}

	bucketUrls := env.BucketURLsFromEnvironment()
	if bucketUrls != nil {
		return normalizeBucketUrls(bucketUrls), nil
	}
	bucketUrls, err := configfile.BucketURLsFromConfigFile()
	if err != nil {
		return nil, err
	}
	return normalizeBucketUrls(bucketUrls), nil
}

func DisableSecureProtocol() (bool, error) {
	isDisabled, ok := env.DisableSecureProtocolFromEnvironment()
	if ok {
		return isDisabled, nil
	}
	return configfile.DisableSecureProtocolFromConfigFile()
}
