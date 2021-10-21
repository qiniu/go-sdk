// +build uint integration

package pili

import (
	"os"

	"github.com/qiniu/go-sdk/v7/client"
	"github.com/stretchr/testify/assert"
)

var (
	ApiHost      = os.Getenv("apiHost")
	AccessKey    = os.Getenv("accessKey")
	SecretKey    = os.Getenv("secretKey")
	IAMAccessKey = os.Getenv("iamAccessKey")
	IAMSecretKey = os.Getenv("iamSecretKey")

	// 部分单元测试需配置额外参数
	TestHub       = os.Getenv("QINIU_TEST_HUB")
	TestDomain    = os.Getenv("QINIU_TEST_DOMAIN")
	TestVodDomain = os.Getenv("QINIU_TEST_VOD_DOMAIN")
	TestCertName  = os.Getenv("QINIU_TEST_CERT_NAME")
	TestStream    = os.Getenv("QINIU_TEST_STREAM")
	TestBucket    = os.Getenv("QINIU_TEST_BUCKET")
)

func init() {
	client.DebugMode = true
}

func SkipTest() bool {
	return AccessKey == "" || SecretKey == "" || TestHub == ""
}

func CheckErr(ast *assert.Assertions, err error, errMsg string) {
	if errMsg == "" {
		ast.Nil(err)
	} else {
		ast.EqualError(err, errMsg)
	}
}
