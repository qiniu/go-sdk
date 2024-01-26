//go:build unit
// +build unit

package configfile

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	file, err := ioutil.TempFile("", "config.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())
	defer file.Close()

	_, err = file.WriteString(`
[default]
access_key = "QINIU_ACCESS_KEY_1"
secret_key = "QINIU_SECRET_KEY_1"

[private-cloud]
access_key = "QINIU_ACCESS_KEY_2"
secret_key = "QINIU_SECRET_KEY_2"
bucket_url = "https://uc.qbox.me"
disable_secure_protocol = true

[private-cloud-2]
access_key = "QINIU_ACCESS_KEY_3"
secret_key = "QINIU_SECRET_KEY_3"
bucket_url = ["https://uc.qbox.me", "https://uc.qiniuapi.com"]
	`)
	if err != nil {
		t.Fatal(err)
	}

	os.Setenv("QINIU_CONFIG_FILE", file.Name())
	defer os.Unsetenv("QINIU_CONFIG_FILE")
	if err = load(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		profileConfigs = nil
	}()
	if len(profileConfigs) != 3 {
		t.Fatal("Unexpected profile configs")
	}
	if profileConfigs["default"].AccessKey != "QINIU_ACCESS_KEY_1" {
		t.Fatal("Unexpected access key")
	}
	if profileConfigs["default"].SecretKey != "QINIU_SECRET_KEY_1" {
		t.Fatal("Unexpected secret key")
	}
	if profileConfigs["default"].BucketURL != nil {
		t.Fatal("Unexpected bucket url")
	}
	if profileConfigs["default"].DisableSecureProtocol {
		t.Fatal("Unexpected disable secure protocol")
	}
	if profileConfigs["private-cloud"].AccessKey != "QINIU_ACCESS_KEY_2" {
		t.Fatal("Unexpected access key")
	}
	if profileConfigs["private-cloud"].SecretKey != "QINIU_SECRET_KEY_2" {
		t.Fatal("Unexpected secret key")
	}
	if profileConfigs["private-cloud"].BucketURL.(string) != "https://uc.qbox.me" {
		t.Fatal("Unexpected bucket url")
	}
	if !profileConfigs["private-cloud"].DisableSecureProtocol {
		t.Fatal("Unexpected disable secure protocol")
	}
	if profileConfigs["private-cloud-2"].AccessKey != "QINIU_ACCESS_KEY_3" {
		t.Fatal("Unexpected access key")
	}
	if profileConfigs["private-cloud-2"].SecretKey != "QINIU_SECRET_KEY_3" {
		t.Fatal("Unexpected secret key")
	}
	if bucketUrls, ok := profileConfigs["private-cloud-2"].BucketURL.([]interface{}); !ok {
		t.Fatal("Unexpected bucket url")
	} else {
		if bucketUrls[0].(string) != "https://uc.qbox.me" {
			t.Fatal("Unexpected bucket url")
		}
		if bucketUrls[1].(string) != "https://uc.qiniuapi.com" {
			t.Fatal("Unexpected bucket url")
		}
	}
	if profileConfigs["private-cloud-2"].DisableSecureProtocol {
		t.Fatal("Unexpected disable secure protocol")
	}

	os.Setenv("QINIU_PROFILE", "private-cloud")
	defer os.Unsetenv("QINIU_PROFILE")

	accessKey, secretKey, err := CredentialsFromConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	if accessKey != "QINIU_ACCESS_KEY_2" {
		t.Fatal("Unexpected access key")
	}
	if secretKey != "QINIU_SECRET_KEY_2" {
		t.Fatal("Unexpected secret key")
	}

	bucketUrls, err := BucketURLsFromConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	if len(bucketUrls) != 1 {
		t.Fatal("Unexpected bucket urls")
	}
	if bucketUrls[0] != "https://uc.qbox.me" {
		t.Fatal("Unexpected bucket url")
	}

	os.Setenv("QINIU_PROFILE", "private-cloud-2")
	defer os.Unsetenv("QINIU_PROFILE")

	accessKey, secretKey, err = CredentialsFromConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	if accessKey != "QINIU_ACCESS_KEY_3" {
		t.Fatal("Unexpected access key")
	}
	if secretKey != "QINIU_SECRET_KEY_3" {
		t.Fatal("Unexpected secret key")
	}

	bucketUrls, err = BucketURLsFromConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	if len(bucketUrls) != 2 {
		t.Fatal("Unexpected bucket urls")
	}
	if bucketUrls[0] != "https://uc.qbox.me" {
		t.Fatal("Unexpected bucket url")
	}
	if bucketUrls[1] != "https://uc.qiniuapi.com" {
		t.Fatal("Unexpected bucket url")
	}
}
