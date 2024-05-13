//go:build unit
// +build unit

package clientv2

import (
	"bytes"
	"crypto/md5"
	"io"
	"io/ioutil"
	"math/rand"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestMultipart(t *testing.T) {
	file1 := randFile(t, 1024*1024*10)
	defer file1.Close()
	file2 := randFile(t, 1024*1024*10)
	defer file2.Close()

	form := new(MultipartForm).
		SetValue("test-1", "value-1").
		SetValue("test-2", "value-2").
		SetFile("test-file-1", "test-file-name-1", "application/json", file1).
		SetFile("test-file-2", "test-file-name-2", "application/x-www-form-urlencoded", file2)
	getRequestBody := GetMultipartFormRequestBody(form)

	for i := 0; i < 5; i++ {
		header := make(http.Header)
		requestBody, err := getRequestBody(&RequestParams{Header: header})
		if err != nil {
			t.Fatal(err)
		}
		reader, err := parseMultipart(requestBody, header)
		if err != nil {
			t.Fatal(err)
		}
		if part, err := reader.NextPart(); err != nil {
			t.Fatal(err)
		} else {
			assertPartValue(t, part, "test-1", "value-1")
		}
		if part, err := reader.NextPart(); err != nil {
			t.Fatal(err)
		} else {
			assertPartValue(t, part, "test-2", "value-2")
		}
		if part, err := reader.NextPart(); err != nil {
			t.Fatal(err)
		} else if f, err := os.Open(file1.Name()); err != nil {
			t.Fatal(err)
		} else {
			assertPartFile(t, part, "test-file-1", "test-file-name-1", "application/json", f)
			f.Close()
		}
		if part, err := reader.NextPart(); err != nil {
			t.Fatal(err)
		} else if f, err := os.Open(file2.Name()); err != nil {
			t.Fatal(err)
		} else {
			assertPartFile(t, part, "test-file-2", "test-file-name-2", "application/x-www-form-urlencoded", f)
			f.Close()
		}
		if _, err = reader.NextPart(); err != io.EOF {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func randFile(t *testing.T, n int64) *os.File {
	file, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.CopyN(file, rand.New(rand.NewSource(time.Now().UnixNano())), 1024*1024*10)
	if err != nil {
		t.Fatal(err)
	}
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func parseMultipart(r io.Reader, header http.Header) (*multipart.Reader, error) {
	_, params, err := mime.ParseMediaType(header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	return multipart.NewReader(r, params["boundary"]), nil
}

func assertPartValue(t *testing.T, part *multipart.Part, key, value string) {
	if part.FormName() != key {
		t.Fatalf("unexpected form name: %s != %s", part.FormName(), key)
	}
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, part); err != nil {
		t.Fatal(err)
	}
	if buf.String() != value {
		t.Fatalf("unexpected form value: %s != %s", buf.String(), value)
	}
}

func assertPartFile(t *testing.T, part *multipart.Part, key, fileName, contentType string, value io.Reader) {
	if part.FormName() != key {
		t.Fatalf("unexpected form name: %s != %s", part.FormName(), key)
	}
	if part.FileName() != fileName {
		t.Fatalf("unexpected file name: %s != %s", part.FileName(), fileName)
	}
	if acutalContentType := part.Header.Get("Content-Type"); acutalContentType != contentType {
		t.Fatalf("unexpected content type: %s != %s", acutalContentType, contentType)
	}
	md5Hasher := md5.New()
	if _, err := io.Copy(md5Hasher, part); err != nil {
		t.Fatal(err)
	}
	actualMd5 := md5Hasher.Sum(nil)
	md5Hasher.Reset()
	if _, err := io.Copy(md5Hasher, value); err != nil {
		t.Fatal(err)
	}
	expectedMd5 := md5Hasher.Sum(nil)
	if !bytes.Equal(actualMd5, expectedMd5) {
		t.Fatalf("unexpected form file value")
	}
}
