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

func TestMakeReadSeekCloserFromReader(t *testing.T) {
	file, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())
	defer file.Close()

	for i := 0; i < 16; i++ {
		if _, err = file.Write([]byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xA, 0xB, 0xC, 0xD, 0xE, 0xF}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 16)

	reader := internal_io.MakeReadSeekCloserFromReader(file)
	if n, err := reader.Read(buf); err != nil {
		t.Fatal(err)
	} else if !bytes.Equal(buf[:n], []byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xA, 0xB, 0xC, 0xD, 0xE, 0xF}) {
		t.Fatal("unexpected read content")
	}
	if _, err = reader.Seek(8, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	if n, err := reader.Read(buf); err != nil {
		t.Fatal(err)
	} else if !bytes.Equal(buf[:n], []byte{0x8, 0x9, 0xA, 0xB, 0xC, 0xD, 0xE, 0xF, 0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7}) {
		t.Fatal("unexpected read content")
	}
	if _, err = reader.Seek(-8, io.SeekEnd); err != nil {
		t.Fatal(err)
	}
	if n, err := reader.Read(buf); err != nil {
		t.Fatal(err)
	} else if !bytes.Equal(buf[:n], []byte{0x8, 0x9, 0xA, 0xB, 0xC, 0xD, 0xE, 0xF}) {
		t.Fatal("unexpected read content")
	}
	if err = reader.Close(); err != nil {
		t.Fatal(err)
	}
}

func MakeReadSeekCloserFromLimitedReader(t *testing.T) {
	file, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())
	defer file.Close()

	for i := 0; i < 16; i++ {
		if _, err = file.Write([]byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xA, 0xB, 0xC, 0xD, 0xE, 0xF}); err != nil {
			t.Fatal(err)
		}
	}

	buf := make([]byte, 16)

	reader := internal_io.MakeReadSeekCloserFromLimitedReader(file, 16)
	if n, err := reader.Read(buf); err != nil {
		t.Fatal(err)
	} else if !bytes.Equal(buf[:n], []byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xA, 0xB, 0xC, 0xD, 0xE, 0xF}) {
		t.Fatal("unexpected read content")
	}

	if _, err = reader.Seek(8, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	if n, err := reader.Read(buf); err != nil {
		t.Fatal(err)
	} else if !bytes.Equal(buf[:n], []byte{0x8, 0x9, 0xA, 0xB, 0xC, 0xD, 0xE, 0xF}) {
		t.Fatal("unexpected read content")
	}

	if err = reader.Close(); err != nil {
		t.Fatal(err)
	}
}
