package storage

import "io"

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

func NewUploadSourceFile(filePath string) UploadSource {
	return &uploadSourceFile{
		filePath: filePath,
	}
}

type uploadSourceFile struct {
	filePath string
}

func (u *uploadSourceFile) Reloadable() bool {
	return false
}

func (u *uploadSourceFile) Reload() error {
	return nil
}

func (u *uploadSourceFile) Size() int64 {
	return -1
}
