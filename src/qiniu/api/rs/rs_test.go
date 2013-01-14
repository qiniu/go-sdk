package rs

import (
	"os"
	"testing"
	"qiniu/api/conf"
)

const (
	Bucket = "gosdktest"
)

var (
	rs Service
)

func init() {

	conf.ACCESS_KEY = "iN7NgwM31j4-BZacMjPrOQBs34UG1maYCAQmhdCV"
	conf.SECRET_KEY = "6QTOr2Jg1gcZEWDQXKOGZh5PziC2MCV5KsntT70j"
	rs = New()
}

func doTestPut(t *testing.T) {

	file := "rs_test.go"
	entryURI := Bucket + ":" + file
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fi, _ := f.Stat()
	
	ret, err := rs.Put(nil, entryURI, "", f, fi.Size())
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ret)
}

func TestDo(t *testing.T) {

	doTestPut(t)
}