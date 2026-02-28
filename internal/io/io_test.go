//go:build unit
// +build unit

package io_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
)

func TestReadAll(t *testing.T) {
	runTestCase := func(t *testing.T, r io.Reader, expected []byte) {
		if b, err := internal_io.ReadAll(r); err != nil {
			t.Fatal(err)
		} else if !bytes.Equal(expected, b) {
			t.Fatalf("unexpected read content: b=%#v, expected=%#v", b, expected)
		} else if n, err := r.Read(make([]byte, 1)); err != nil && err != io.EOF {
			t.Fatal(err)
		} else if n != 0 {
			t.Fatal("unexpected read size")
		}
	}
	buf := new(bytes.Buffer)
	for i := 0; i < 16; i++ {
		if _, err := buf.Write([]byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xA, 0xB, 0xC, 0xD, 0xE, 0xF}); err != nil {
			t.Fatal(err)
		}
	}
	expected := buf.Bytes()
	runTestCase(t, buf, expected)

	file, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())
	defer file.Close()

	if _, err = io.Copy(file, bytes.NewReader(expected)); err != nil {
		t.Fatal(err)
	} else if _, err = file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	runTestCase(t, file, expected)

	bytesNopCloser := internal_io.NewBytesNopCloser(expected)
	runTestCase(t, bytesNopCloser, expected)
}

func TestSinkAll(t *testing.T) {
	runTestCase := func(t *testing.T, r io.Reader) {
		if err := internal_io.SinkAll(r); err != nil {
			t.Fatal(err)
		} else if n, err := r.Read(make([]byte, 1)); err != nil && err != io.EOF {
			t.Fatal(err)
		} else if n != 0 {
			t.Fatal("unexpected read size")
		}
	}

	buf := new(bytes.Buffer)
	for i := 0; i < 16; i++ {
		if _, err := buf.Write([]byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xA, 0xB, 0xC, 0xD, 0xE, 0xF}); err != nil {
			t.Fatal(err)
		}
	}
	expected := buf.Bytes()
	runTestCase(t, buf)

	file, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())
	defer file.Close()

	if _, err = io.Copy(file, bytes.NewReader(expected)); err != nil {
		t.Fatal(err)
	} else if _, err = file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	runTestCase(t, file)

	bytesNopCloser := internal_io.NewBytesNopCloser(expected)
	runTestCase(t, bytesNopCloser)
}
