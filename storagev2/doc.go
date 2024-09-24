// storagev2 包提供了资源管理等功能。
package storagev2

//go:generate go run ../internal/api-generator -- --api-specs=../api-specs/storage --api-specs=internal/api-specs --output=apis/ --struct-name=Storage --api-package=github.com/qiniu/go-sdk/v7/storagev2/apis
//go:generate go build ./apis/...
