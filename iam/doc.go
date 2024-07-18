// iam 包提供了 IAM 子账号管理等功能。
package iam

//go:generate go run ../internal/api-generator -- --api-specs=../api-specs/iam --output=apis/ --struct-name=IAM --api-package=github.com/qiniu/go-sdk/v7/iam/apis
//go:generate go build ./apis/...
