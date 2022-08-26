package storage

import (
	"io"
	"os"
)

type UploadSource interface {
	Size() int64
	Reloadable() bool
	Reload() error
}

func NewUploadSourceReader(reader io.Reader) UploadSource {
	return &uploadSourceReader{
		reader: reader,
	}
}

type uploadSourceReader struct {
	reader io.Reader
}

func (u *uploadSourceReader) Reloadable() bool {
	return false
}

func (u *uploadSourceReader) Reload() error {
	return nil
}

func (u *uploadSourceReader) Size() int64 {
	return -1
}

func NewUploadSourceReaderAt(reader io.ReaderAt, size int64) UploadSource {
	return &uploadSourceReaderAt{
		reader: reader,
		size:   size,
	}
}

type uploadSourceReaderAt struct {
	reader io.ReaderAt
	size   int64
}

func (u *uploadSourceReaderAt) Reloadable() bool {
	return true
}

func (u *uploadSourceReaderAt) Reload() error {
	return nil
}

func (u *uploadSourceReaderAt) Size() int64 {
	return u.size
}

func NewUploadSourceFile(filePath string) (UploadSource, error) {
	if fileInfo, err := os.Stat(filePath); err != nil {
		return nil, err
	} else {
		return &uploadSourceFile{
			fileInfo: fileInfo,
			filePath: filePath,
		}, nil
	}
}

type uploadSourceFile struct {
	filePath string
	fileInfo os.FileInfo
}

func (u *uploadSourceFile) Reloadable() bool {
	return false
}

func (u *uploadSourceFile) Reload() error {
	return nil
}

func (u *uploadSourceFile) Size() int64 {
	return u.fileInfo.Size()
}
