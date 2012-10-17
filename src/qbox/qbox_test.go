package qbox

import (
	"os"
	"io/ioutil"
	"testing"
	"encoding/json"
	. "qbox/api"
	"qbox/api/eu"
	"qbox/api/up"
	"qbox/api/uc"
	"qbox/api/pub"
	"qbox/api/rs"
	"qbox/api/image"
)

// global testing variables
const (
	testfile = "qbox.conf"
	testbucket = "test_bucket"
	testkey = "qbox.conf"
	testimageurl = "http://qiniuphotos.dn.qbox.me/gogopher.jpg"
)

var (
	eus *eu.Service
	ups *up.Service
	ucs *uc.Service
	pubs *pub.Service
	rss *rs.Service
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
	code, err := pubs.Image(testbucket, urls, host, 0)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestUnimage(t *testing.T) {
	code, err := pubs.Unimage(testbucket)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestInfo(t *testing.T) {
	bi, code, err := pubs.Info(testbucket)
	if code/100 != 2 {
		t.Fatal(err)
	}
	t.Log(bi)
}

func doTestAccessMode(t *testing.T) {
	code, err := pubs.AccessMode(testbucket, 1)
	if code/100 != 2 {
		t.Fatal(err)
	}
	code, err = pubs.AccessMode(testbucket, 0)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestSeparator(t *testing.T) {
	code, err := pubs.Separator(testbucket, "-")
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestStyle(t *testing.T) {
	style := "imageMogr/auto-orient/thumbnail/!120x120r/gravity/center/crop/!120x120/quality/80"
	code, err := pubs.Style(testbucket, "small.jpg", style)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestUnstyle(t *testing.T) {
	code, err := pubs.Unstyle(testbucket, "small.jpg")
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
	ret, code, err := rss.Put(entryURI, mimeType, f, fi.Size())
	if code/100 != 2 {
		t.Fatal(err)
	}
	t.Log(ret)
}

func doTestGet(t *testing.T) {
	entryURI := testbucket + ":" + testkey
	ret, code, err := rss.Get(entryURI, "", "", 0)
	if code/100 != 2 {
		t.Fatal(err)
	}
	t.Log(ret)
}

func doTestStat(t *testing.T) {
	entryURI := testbucket + ":" + testkey
	ret, code, err := rss.Stat(entryURI)
	if code/100 != 2 {
		t.Fatal(err)
	}
	t.Log(ret)
}

func doTestDelete(t *testing.T) {
	entryURI := testbucket + ":" + testkey
	ret, code, err := rss.Stat(entryURI)
	if code/100 != 2 {
		t.Fatal(err)
	}
	t.Log(ret)
}

func doTestMkbucket(t *testing.T) {
	bucketname := testbucket + "1"
	code, err := rss.Mkbucket(bucketname)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestDrop(t *testing.T) {
	bucketname := testbucket + "1"
	code, err := rss.Drop(bucketname)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestMove(t *testing.T) {
	srcURI := testbucket + ":" + testkey
	destURI := srcURI + "1"
	code, err := rss.Move(srcURI, destURI)
	if code/100 != 2 {
		t.Fatal(err)
	}
	code, err = rss.Move(destURI, srcURI)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestCopy(t *testing.T) {
	srcURI := testbucket + ":" + testkey
	destURI := srcURI + "1"
	code, err := rss.Copy(srcURI, destURI)
	if code/100 != 2 {
		t.Fatal(err)
	}
	code, err = rss.Delete(destURI)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestPublish(t *testing.T) {
	domain := "mydomain.qboxtest.me"
	code, err := rss.Publish(domain, testbucket)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestUnpublish(t *testing.T) {
	domain := "mydomain.qboxtest.me"
	code, err := rss.Unpublish(domain)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestAntiLeechMode(t *testing.T) {
	code, err := ucs.AntiLeechMode(testbucket, 1)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestAddAntiLeech(t *testing.T) {
	code, err := ucs.AddAntiLeech(testbucket, 1, "12.34.56.*")
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestCleanCache(t *testing.T) {
	code, err := ucs.CleanCache(testbucket)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestDelAntiLeech(t *testing.T) {
	code, err := ucs.DelAntiLeech(testbucket, 1, "12.34.56.*")
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestRSSResumablePut(t *testing.T) {
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
	code, err := rss.ResumablePut(entryURI, "application/json", f, fi.Size())
	if code/100 != 2 {
		t.Fatal(err)
	}
}


func doTestUPSResumablePut(t *testing.T) {
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
	code, err := ups.ResumablePut(entryURI, "application/json", f, fi.Size())
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestInit(t *testing.T) {
	var c Config
	b, err := ioutil.ReadFile("qbox.conf")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	if eus, err = eu.New(&c); err != nil {
		t.Fatal(err)
	}
	if pubs, err = pub.New(&c); err != nil {
		t.Fatal(err)
	}
	if rss, err = rs.New(&c); err != nil {
		t.Fatal(err)
	}
	if ucs, err = uc.New(&c); err != nil {
		t.Fatal(err)
	}
	if ups, err = up.New(&c); err != nil {
		t.Fatal(err)
	}
}

func doTestImageInfo(t *testing.T) {
	ret, code, err := image.Info(testimageurl)
	if code/100 != 2 {
		t.Fatal(err)
	}
	t.Log(ret)
}

func doTestImageExif(t *testing.T) {
	ret, code, err := image.Exif(testimageurl)
	if code/100 != 2 {
		t.Fatal(err)
	}
	t.Log(ret)
}

func doTestImageView(t *testing.T) {
	f, err := ioutil.TempFile("", "imageview.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	p := map[string]string {
		"Mode": "1",
		"Width": "200",
		"Height": "200",
		"Format": "gif",
		// "Sharpen": "",
		// "HasWatermark": "",
	}
	code, err := image.View(f, testimageurl, p)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func doTestImageMogr(t *testing.T) {
	f, err := ioutil.TempFile("", "imagemogr.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	p := map[string]string {
		"Thumbnail": "!100x100",
		"Gravity": "center",
		"Crop": "!100x100",
		"Quality": "80",
	}
	code, err := image.Mogr(f, testimageurl, p)
	if code/100 != 2 {
		t.Fatal(err)
	}
}

func TestDo(t *testing.T) {

	doTestInit(t)
/*
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
	doTestRSSResumablePut(t)
	doTestUPSResumablePut(t)
*/
	doTestImageInfo(t)
	doTestImageExif(t)
	doTestImageView(t)
	doTestImageMogr(t)
}