package env

import (
	"os"
	"strings"
)

const (
	environmentVariableNameQiniuAccessKey                 = "QINIU_ACCESS_KEY"
	environmentVariableNameQiniuSecretKey                 = "QINIU_SECRET_KEY"
	environmentVariableNameQiniuConfigFile                = "QINIU_CONFIG_FILE"
	environmentVariableNameQiniuProfile                   = "QINIU_PROFILE"
	environmentVariableNameQiniuBucketURL                 = "QINIU_BUCKET_URL"
	environmentVariableNameDisableQiniuSecureProtocol     = "DISABLE_QINIU_SECURE_PROTOCOL"
	environmentVariableNameDisableQiniuTimestampSignature = "DISABLE_QINIU_TIMESTAMP_SIGNATURE"
)

func CredentialsFromEnvironment() (string, string) {
	accessKey := os.Getenv(environmentVariableNameQiniuAccessKey)
	secretKey := os.Getenv(environmentVariableNameQiniuSecretKey)
	if accessKey == "" || secretKey == "" {
		return "", ""
	}
	return accessKey, secretKey
}

func ConfigFileFromEnvironment() string {
	return os.Getenv(environmentVariableNameQiniuConfigFile)
}

func ProfileFromEnvironment() string {
	return os.Getenv(environmentVariableNameQiniuProfile)
}

func BucketURLsFromEnvironment() []string {
	bucketUrls := os.Getenv(environmentVariableNameQiniuBucketURL)
	if bucketUrls == "" {
		return nil
	}
	urls := strings.Split(bucketUrls, ",")
	for i := range urls {
		urls[i] = strings.TrimSpace(urls[i])
	}
	return urls
}

func DisableSecureProtocolFromEnvironment() (bool, bool) {
	value := strings.ToLower(os.Getenv(environmentVariableNameDisableQiniuSecureProtocol))
	if value == "" {
		return false, false
	}
	return value == "true" || value == "yes" || value == "y" || value == "1", true
}

func DisableQiniuTimestampSignatureFromEnvironment() (bool, bool) {
	value := strings.ToLower(os.Getenv(environmentVariableNameDisableQiniuTimestampSignature))
	if value == "" {
		return false, false
	}
	return value == "true" || value == "yes" || value == "y" || value == "1", true
}
