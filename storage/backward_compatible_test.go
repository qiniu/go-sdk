// +build unit

package storage

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/qiniu/go-sdk/v7/conf"
)

func TestVariable(t *testing.T) {
	appName := "test"

	SetAppName(appName)

	want := fmt.Sprintf("QiniuGo/%s (%s; %s; %s) %s", conf.Version, runtime.GOOS, runtime.GOARCH, appName, runtime.Version())

	if UserAgent != want {
		t.Fail()
	}
}
