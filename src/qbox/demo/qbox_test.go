package main

import (
	"os"
	"testing"
	"time"
	"qbox/api/rs"
	"qbox/api/eu"
	"qbox/api/pub"
	"qbox/api/uc"
	"qbox/api/up"
	. "qbox/api/conf"
	"qbox/auth/digest"
	"qbox/auth/uptoken"
)

var t = digest.NewTransport(QBOX_ACCESS_KEY,QBOX_SECRET_KEY,nil)

var aP = & uptoken.AuthPolicy {
	Scope: "fd1",
	Customer: "qboxuser",
	Deadline: 3600 + uint32(time.Now().Unix()),
}
var token = uptoken.MakeAuthTokenString(QBOX_ACCESS_KEY, QBOX_SECRET_KEY, aP)
var t1 = uptoken.NewTransport(token, nil)

var rsClient = rs.New(t)
var euClient = eu.New(t)
var pubClient = pub.New(t)
var ucClient = uc.New(t)
var upClient = up.New(1, 1, t1)

func doTestPut(t *testing.T, file string) {

	f, err := os.Open(file)
	if err != nil {
		t.Fatal("Cant't open this file : ", file)
	}
	fi, _ := os.Stat(file)
	fsize := fi.Size()
	ret, code, err := rsClient.Put("fd1:ruby.md", "", f, fsize)
	if err != nil {
		t.Fatal("Can't put this file to qbox : ", code, err)
	} else {
		t.Log(ret)
	}
}

func doTestGetWithExpires(t *testing.T, bucket, key string) {
	ret, code, err := rsClient.GetWithExpires(bucket + ":" + key, "", 0)
	if err != nil {
		t.Fatal("Can't get this file from qbox : ", code, err)
	} else {
		t.Log(ret)
	}
}

func doTestWatermark(t *testing.T) {
	params := &eu.Watermark {
		Text: "qboxtest",
	}
	customer := "qboxuser"
	code, err := euClient.SetWatermark(customer, params)
	if code != 200 {
		t.Fatal("Set Watermark error : ", code, err)
	}
	_, code, err = euClient.GetWatermark(customer)
	if code != 200 {
		t.Fatal("Get Watermark error : ", code, err)
	}
}

func doTestPub(t *testing.T) {
	ret, code, err := pubClient.Info("fd1")
	if code != 200 {
		t.Fatal("Can't get the bucket infomation : ", ret, code, err)
	}
}


func doTestAppInfo(t *testing.T) {
	appName := "default"
	ret, code, err := ucClient.AppInfo(appName)
	if code != 200 {
		t.Fatal("Can't get the app infomation : ", ret, code, err)
	}
}


func doTestResumablePut(t *testing.T) {
	f, err := os.Open("ruby.md")
	if err != nil {
		t.Fatal("Can't open the file : ruby.md", err)
	}
	fi, _ := os.Stat("ruby.md")
	c := up.BlockCount(fi.Size())
	checksums := make([]string, c)
	progs := make([]up.BlockputProgress, c)
	code, err := upClient.Put(f, fi.Size(), checksums, progs, nil, nil)

	if code != 200 {
		t.Fatal("ResumablePut file error : ", code, err, checksums)
	}
}


func TestDo(t *testing.T) {
//	doTestPut(t, "ruby.md")
//	doTestGetWithExpires(t, "fd1", "ruby.md")
//	doTestWatermark(t)
//	doTestPub(t)
	doTestResumablePut(t)
}