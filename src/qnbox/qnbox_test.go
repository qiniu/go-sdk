package qnbox

import (
	"os"
	"io/ioutil"
	"testing"
	"encoding/json"
)

// global testing variables
const (
	testfile = "qnbox.conf"
	testbucket = "test_bucket"
	testkey = "qnbox.conf"
)

var (
	s *Service
)

func doTestSetWatermark(t *testing.T) {

}

func doTestGetWatermark(t *testing.T) {

}

func doTestImage(t *testing.T) {
	urls := make([]string, 2)
	urls[0] = "www.google.com"
	urls[1] = "www.baidu.com"
	host := "mydomain.qbox.me"
	code, err := s.Image(testbucket, urls, host, 0)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestUnimage(t *testing.T) {
	code, err := s.Unimage(testbucket)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestInfo(t *testing.T) {
	bi, code, err := s.Info(testbucket)
	if code/100 != 2 {
		t.Fatal(err)
	}
	t.Log(bi)
}

func doTestAccessMode(t *testing.T) {
	code, err := s.AccessMode(testbucket, 1)
	if code/100 != 2 {
		t.Fatal(err)
	}
	code, err = s.AccessMode(testbucket, 0)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestSeparator(t *testing.T) {
	code, err := s.Separator(testbucket, "-")
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestStyle(t *testing.T) {
	style := "imageMogr/auto-orient/thumbnail/!120x120r/gravity/center/crop/!120x120/quality/80"
	code, err := s.Style(testbucket, "small.jpg", style)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestUnstyle(t *testing.T) {
	code, err := s.Unstyle(testbucket, "small.jpg")
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestPut(t *testing.T) {
	entryURI := testbucket + ":" + testkey
	mimeType := "application/json"
	f, err := os.Open(testfile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	ret, code, err := s.Put(entryURI, mimeType, f, fi.Size())
	if code/100 != 2 {
		t.Fatal(err)
	}
	t.Log(ret)
}

func doTestGet(t *testing.T) {
	entryURI := testbucket + ":" + testkey
	ret, code, err := s.Get(entryURI, "", "", 0)
	if code/100 != 2 {
		t.Fatal(err)
	}
	t.Log(ret)
}

func doTestStat(t *testing.T) {
	entryURI := testbucket + ":" + testkey
	ret, code, err := s.Stat(entryURI)
	if code/100 != 2 {
		t.Fatal(err)
	}
	t.Log(ret)
}

func doTestDelete(t *testing.T) {
	entryURI := testbucket + ":" + testkey
	ret, code, err := s.Stat(entryURI)
	if code/100 != 2 {
		t.Fatal(err)
	}
	t.Log(ret)
}

func doTestMkbucket(t *testing.T) {
	bucketname := testbucket + "1"
	code, err := s.Mkbucket(bucketname)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestDrop(t *testing.T) {
	bucketname := testbucket + "1"
	code, err := s.Drop(bucketname)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestMove(t *testing.T) {
	srcURI := testbucket + ":" + testkey
	destURI := srcURI + "1"
	code, err := s.Move(srcURI, destURI)
	if code/100 != 2 {
		t.Fatal(err)
	}
	code, err = s.Move(destURI, srcURI)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestCopy(t *testing.T) {
	srcURI := testbucket + ":" + testkey
	destURI := srcURI + "1"
	code, err := s.Copy(srcURI, destURI)
	if code/100 != 2 {
		t.Fatal(err)
	}
	code, err = s.Delete(destURI)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestPublish(t *testing.T) {
	domain := "mydomain.qboxtest.me"
	code, err := s.Publish(domain, testbucket)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestUnpublish(t *testing.T) {
	domain := "mydomain.qboxtest.me"
	code, err := s.Publish(domain, testbucket)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestAntiLeechMode(t *testing.T) {
	code, err := s.AntiLeechMode(testbucket, 1)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestAddAntiLeech(t *testing.T) {
	code, err := s.AddAntiLeech(testbucket, 1, "12.34.56.*")
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestCleanCache(t *testing.T) {
	code, err := s.CleanCache(testbucket)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestDelAntiLeech(t *testing.T) {
	code, err := s.DelAntiLeech(testbucket, 1, "12.34.56.*")
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestResumablePut(t *testing.T) {
	entryURI := testbucket + ":" + testkey
	f, err := os.Open(testfile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	code, err := s.ResumablePut(entryURI, "application/json", f, fi.Size())
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func TestDo(t *testing.T) {
	var c Config
	b, err := ioutil.ReadFile("qnbox.conf")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	s = New(c, nil)

	doTestSetWatermark(t)
	doTestGetWatermark(t)
	doTestImage(t)
	doTestUnimage(t)
	doTestInfo(t)
	doTestAccessMode(t)
	doTestSeparator(t)
	doTestStyle(t)
	doTestUnstyle(t)
	doTestPut(t)
	doTestGet(t)
	doTestStat(t)
	doTestMove(t)
//	doTestCopy(t)
	doTestDelete(t)
	doTestMkbucket(t)
	doTestDrop(t)
	doTestPublish(t)
	doTestUnpublish(t)
//	doTestAntiLeechMode(t)  // not suport digest
//	doTestAddAntiLeech(t)
//	doTestDelAntiLeech(t)
//	doTestCleanCache(t)
	doTestResumablePut(t)
}