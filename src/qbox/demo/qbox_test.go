package main

import (
	"os"
	"testing"
	"qbox/api/rs"
	"qbox/api/eu"
	"qbox/api/pub"
	"qbox/api/uc"
	"qbox/api/up"
	. "qbox/api/conf"
	"qbox/auth/digest"
	"qbox/auth/uptoken"
)


// digest authorization
var t = digest.NewTransport(QBOX_ACCESS_KEY,QBOX_SECRET_KEY,nil)

var rsClient = rs.New(t)
var euClient = eu.New(t)
var pubClient = pub.New(t)
var ucClient = uc.New(t)


// uptoken authorization
var aP = &uptoken.AuthPolicy{}
var token = uptoken.MakeAuthTokenString(QBOX_ACCESS_KEY, QBOX_SECRET_KEY, aP)
var t1 = uptoken.NewTransport(token, nil)

var upClient = up.New(1, 1, t1)


// global testing variables
var testfile = "demofile.md"
var testbucket = "fd1"
var testkey = "demofile.md"



// test case

func doTestRsService(t *testing.T) {

	// testing Put()
	f, err := os.Open(testfile)
	if err != nil {
		t.Fatal("Cant't open test file : ", testfile)
	}
	fi, _ := os.Stat(testfile)
	fsize := fi.Size()
	_, code, err := rsClient.Put(testbucket + ":" + testkey, "", f, fsize)
	if err != nil {
		t.Fatal("Can't put testfile to qbox : ", code, err)
	}


	// testing GetWithExpires()
	_, code, err = rsClient.GetWithExpires(testbucket + ":" + testkey, "", 0)
	if err != nil {
		t.Fatal("Can't get testfile from qbox : ", code, err)
	}
}

func doTestEuService(t *testing.T) {

	// testing Watermark()
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

func doTestPubService(t *testing.T) {

	// testing Info()
	ret, code, err := pubClient.Info("fd1")
	if code != 200 {
		t.Fatal("Can't get the bucket infomation : ", ret, code, err)
	}
}


func doTestUcService(t *testing.T) {

	// testing AppInfo()
	appName := "default"
	ret, code, err := ucClient.AppInfo(appName)
	if code != 200 {
		t.Fatal("Can't get the app infomation : ", ret, code, err)
	}
}


func doTestUpService(t *testing.T) {

	// testing resumableput
	f, err := os.Open(testfile)
	if err != nil {
		t.Fatal("Can't open the file : ", testfile, err)
	}
	fi, _ := os.Stat(testfile)
	c := up.BlockCount(fi.Size())
	checksums := make([]string, c)
	progs := make([]up.BlockputProgress, c)
	code, err := upClient.Put(f, fi.Size(), checksums, progs, nil, nil)
	if code != 200 {
		t.Fatal("ResumablePut file error : ", code, err, checksums)
	}
}


func TestDo(t *testing.T) {
	doTestRsService(t)
	doTestEuService(t)
	doTestPubService(t)
	doTestUpService(t)
}