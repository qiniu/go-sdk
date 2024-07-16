package source

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
)

type (
	// 数据源
	Source interface {
		io.Closer

		// 切片
		Slice(uint64) (Part, error)

		// 数据源 ID
		SourceID() (string, error)

		// 获取文件，如果数据源不是文件，则返回 nil
		GetFile() *os.File
	}

	// 预知大小的数据源
	SizedSource interface {
		Source

		// 获取数据源大小
		TotalSize() (uint64, error)
	}

	// 可重置的数据源
	ResetableSource interface {
		Source

		// 重置数据源
		Reset() error
	}

	// 分片
	Part interface {
		io.ReadSeeker

		// 分片偏移量
		Offset() uint64

		// 分片大小
		Size() uint64

		// 分片编号，从 1 开始
		PartNumber() uint64
	}

	seekablePart struct {
		*io.SectionReader
		partNumber, offset uint64
	}

	unseekablePart struct {
		*bytes.Reader
		partNumber, offset, size uint64
	}

	readSeekCloseSource struct {
		rscra      *readSeekCloseReaderAt
		off        uint64
		sourceID   string
		partNumber uint64
		m          sync.Mutex
	}

	readSeekCloseReaderAt struct {
		r   internal_io.ReadSeekCloser
		off int64
		m   sync.Mutex
	}

	readCloseSource struct {
		r                  io.ReadCloser
		sourceID           string
		offset, partNumber uint64
	}

	ReadAtSeekCloser interface {
		io.ReaderAt
		io.Seeker
		io.Closer
	}

	readAtSeekCloseSource struct {
		r          ReadAtSeekCloser
		off        uint64
		sourceID   string
		partNumber uint64
		m          sync.Mutex
	}
)

// 将 io.ReadSeekCloser 封装为数据源
func NewReadSeekCloserSource(r internal_io.ReadSeekCloser, sourceID string) Source {
	return &readSeekCloseSource{rscra: newReadSeekCloseReaderAt(r), sourceID: sourceID}
}

func (rscs *readSeekCloseSource) Slice(n uint64) (Part, error) {
	rscs.m.Lock()
	defer rscs.m.Unlock()

	offset := rscs.off
	if totalSize, err := rscs.TotalSize(); err != nil {
		return nil, err
	} else if offset >= totalSize {
		return nil, nil
	} else if n > totalSize-offset {
		n = totalSize - offset
	}
	rscs.off += n
	rscs.partNumber += 1
	return seekablePart{
		io.NewSectionReader(rscs.rscra, int64(offset), int64(n)),
		rscs.partNumber,
		uint64(offset),
	}, nil
}

func (rscs *readSeekCloseSource) TotalSize() (uint64, error) {
	return rscs.rscra.TotalSize()
}

func (rscs *readSeekCloseSource) SourceID() (string, error) {
	return rscs.sourceID, nil
}

func (rscs *readSeekCloseSource) Close() error {
	return rscs.rscra.Close()
}

func (rscs *readSeekCloseSource) Reset() error {
	rscs.m.Lock()
	defer rscs.m.Unlock()

	rscs.off = 0
	rscs.partNumber = 0
	return nil
}

func (rscs *readSeekCloseSource) GetFile() *os.File {
	return rscs.rscra.GetFile()
}

func newReadSeekCloseReaderAt(r internal_io.ReadSeekCloser) *readSeekCloseReaderAt {
	return &readSeekCloseReaderAt{r: r, off: -1}
}

func (rscra *readSeekCloseReaderAt) ReadAt(b []byte, off int64) (n int, err error) {
	rscra.m.Lock()
	defer rscra.m.Unlock()

	if rscra.off != off {
		if rscra.off, err = rscra.r.Seek(off, io.SeekStart); err != nil {
			return
		}
	}
	n, err = rscra.r.Read(b)
	rscra.off += int64(n)
	return
}

func (rscra *readSeekCloseReaderAt) TotalSize() (uint64, error) {
	rscra.m.Lock()
	defer rscra.m.Unlock()

	var err error

	if rscra.off < 0 {
		if rscra.off, err = rscra.r.Seek(0, io.SeekCurrent); err != nil {
			return 0, err
		}
	}
	len, err := rscra.r.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}
	_, err = rscra.r.Seek(rscra.off, io.SeekStart)
	return uint64(len), err
}

func (rscra *readSeekCloseReaderAt) Close() error {
	return rscra.r.Close()
}

func (rscra *readSeekCloseReaderAt) GetFile() *os.File {
	if file, ok := rscra.r.(*os.File); ok {
		return file
	} else {
		return nil
	}
}

// 将 io.ReadAt + io.Seek + io.Closer 封装为数据源
func NewReadAtSeekCloserSource(r ReadAtSeekCloser, sourceID string) Source {
	return &readAtSeekCloseSource{r: r, sourceID: sourceID}
}

func (racs *readAtSeekCloseSource) Slice(n uint64) (Part, error) {
	racs.m.Lock()
	defer racs.m.Unlock()

	offset := racs.off
	if totalSize, err := racs.TotalSize(); err != nil {
		return nil, err
	} else if offset >= totalSize {
		return nil, nil
	} else if n > totalSize-offset {
		n = totalSize - offset
	}
	racs.off += n
	racs.partNumber += 1
	return seekablePart{
		io.NewSectionReader(racs.r, int64(offset), int64(n)),
		racs.partNumber,
		uint64(offset),
	}, nil
}

func (racs *readAtSeekCloseSource) TotalSize() (uint64, error) {
	curPos, err := racs.r.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	totalSize, err := racs.r.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}
	if _, err = racs.r.Seek(curPos, io.SeekStart); err != nil {
		return 0, err
	}
	return uint64(totalSize), nil
}

func (racs *readAtSeekCloseSource) SourceID() (string, error) {
	return racs.sourceID, nil
}

func (racs *readAtSeekCloseSource) Close() error {
	return racs.r.Close()
}

func (racs *readAtSeekCloseSource) Reset() error {
	racs.m.Lock()
	defer racs.m.Unlock()

	racs.off = 0
	racs.partNumber = 0
	return nil
}

func (racs *readAtSeekCloseSource) GetFile() *os.File {
	if file, ok := racs.r.(*os.File); ok {
		return file
	} else {
		return nil
	}
}

// 将 io.ReadCloser 封装为数据源
func NewReadCloserSource(r io.ReadCloser, sourceID string) Source {
	return &readCloseSource{r: r, sourceID: sourceID}
}

func (rcs *readCloseSource) Slice(n uint64) (Part, error) {
	buf := make([]byte, n)
	haveRead, err := io.ReadFull(rcs.r, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, err
	} else if haveRead == 0 {
		return nil, nil
	}
	return &unseekablePart{
		bytes.NewReader(buf[:haveRead]),
		atomic.AddUint64(&rcs.partNumber, 1),
		atomic.AddUint64(&rcs.offset, uint64(haveRead)) - uint64(haveRead),
		uint64(haveRead),
	}, nil
}

func (rcs *readCloseSource) SourceID() (string, error) {
	return rcs.sourceID, nil
}

func (rcs *readCloseSource) Close() error {
	return rcs.r.Close()
}

func (racs *readCloseSource) GetFile() *os.File {
	if file, ok := racs.r.(*os.File); ok {
		return file
	} else {
		return nil
	}
}

func (p seekablePart) PartNumber() uint64 {
	return p.partNumber
}

func (p seekablePart) Offset() uint64 {
	return p.offset
}

func (p seekablePart) Size() uint64 {
	return uint64(p.SectionReader.Size())
}

func (p unseekablePart) PartNumber() uint64 {
	return p.partNumber
}

func (p unseekablePart) Offset() uint64 {
	return p.offset
}

func (p unseekablePart) Size() uint64 {
	return p.size
}

// 将文件封装为数据源
func NewFileSource(filePath string) (Source, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	if !canSeekReally(file) {
		return NewReadCloserSource(file, ""), nil
	} else if absFilePath, err := filepath.Abs(filePath); err != nil {
		return nil, err
	} else if fileInfo, err := file.Stat(); err != nil {
		return nil, err
	} else {
		sourceID := fmt.Sprintf("%d:%d:%s", fileInfo.Size(), fileInfo.ModTime().UnixNano(), absFilePath)
		return NewReadAtSeekCloserSource(file, sourceID), nil
	}
}

func canSeekReally(seeker io.Seeker) bool {
	_, err := seeker.Seek(0, io.SeekCurrent)
	return err == nil
}
