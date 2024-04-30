package source

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

type (
	Source interface {
		io.Closer
		Slice(uint64) (Part, error)
		SourceKey() (string, error)
	}

	SizedSource interface {
		Source
		TotalSize() (uint64, error)
	}

	ResetableSource interface {
		Source
		Reset() error
	}

	Part interface {
		io.ReadSeeker
		Offset() uint64
		Size() uint64
		PartNumber() uint32
	}

	seekablePart struct {
		*io.SectionReader
		partNumber uint32
		offset     uint64
	}

	unseekablePart struct {
		*bytes.Reader
		partNumber   uint32
		offset, size uint64
	}

	readSeekCloseSource struct {
		rscra      *readSeekCloseReaderAt
		off        uint64
		sourceKey  string
		partNumber uint32
		m          sync.Mutex
	}

	readSeekCloseReaderAt struct {
		r   io.ReadSeekCloser
		off int64
		m   sync.Mutex
	}

	readCloseSource struct {
		r          io.ReadCloser
		sourceKey  string
		offset     uint64
		partNumber uint32
	}

	readAtCloser interface {
		io.ReaderAt
		io.Closer
	}

	readAtCloseSource struct {
		r          readAtCloser
		off        uint64
		n          int64
		sourceKey  string
		partNumber uint32
		m          sync.Mutex
	}
)

func NewReadSeekCloserSource(r io.ReadSeekCloser, sourceKey string) Source {
	return &readSeekCloseSource{rscra: newReadSeekCloseReaderAt(r), sourceKey: sourceKey}
}

func (rscs *readSeekCloseSource) Slice(n uint64) (Part, error) {
	rscs.m.Lock()
	defer rscs.m.Unlock()

	offset := rscs.off
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

func (rscs *readSeekCloseSource) SourceKey() (string, error) {
	return rscs.sourceKey, nil
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

func newReadSeekCloseReaderAt(r io.ReadSeekCloser) *readSeekCloseReaderAt {
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

func NewReadAtCloserSource(r readAtCloser, n int64, sourceKey string) Source {
	return &readAtCloseSource{r: r, n: n, sourceKey: sourceKey}
}

func (racs *readAtCloseSource) Slice(n uint64) (Part, error) {
	racs.m.Lock()
	defer racs.m.Unlock()

	offset := racs.off
	racs.off += n
	racs.partNumber += 1
	return seekablePart{
		io.NewSectionReader(racs.r, int64(offset), int64(n)),
		racs.partNumber,
		uint64(offset),
	}, nil
}

func (racs *readAtCloseSource) TotalSize() (uint64, error) {
	return uint64(racs.n), nil
}

func (racs *readAtCloseSource) SourceKey() (string, error) {
	return racs.sourceKey, nil
}

func (racs *readAtCloseSource) Close() error {
	return racs.r.Close()
}

func (racs *readAtCloseSource) Reset() error {
	racs.m.Lock()
	defer racs.m.Unlock()

	racs.off = 0
	racs.partNumber = 0
	return nil
}

func NewReadCloserSource(r io.ReadCloser, sourceKey string) Source {
	return &readCloseSource{r: r, sourceKey: sourceKey}
}

func (rcs *readCloseSource) Slice(n uint64) (Part, error) {
	buf := make([]byte, 0, n)
	haveRead, err := rcs.r.Read(buf)
	if err != nil {
		return nil, err
	}
	return &unseekablePart{
		bytes.NewReader(buf[:haveRead]),
		atomic.AddUint32(&rcs.partNumber, 1),
		atomic.AddUint64(&rcs.offset, uint64(haveRead)) - uint64(haveRead),
		uint64(haveRead),
	}, nil
}

func (rcs *readCloseSource) SourceKey() (string, error) {
	return rcs.sourceKey, nil
}

func (rcs *readCloseSource) Close() error {
	return rcs.r.Close()
}

func (p seekablePart) PartNumber() uint32 {
	return p.partNumber
}

func (p seekablePart) Offset() uint64 {
	return p.offset
}

func (p seekablePart) Size() uint64 {
	return uint64(p.SectionReader.Size())
}

func (p unseekablePart) PartNumber() uint32 {
	return p.partNumber
}

func (p unseekablePart) Offset() uint64 {
	return p.offset
}

func (p unseekablePart) Size() uint64 {
	return p.size
}

func NewFileSource(filePath string) (Source, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	if _, err := file.Seek(0, io.SeekCurrent); err != nil {
		return NewReadCloserSource(file, ""), nil
	} else if fileInfo, err := file.Stat(); err != nil {
		return nil, err
	} else if absFilePath, err := filepath.Abs(filePath); err != nil {
		return nil, err
	} else {
		return NewReadAtCloserSource(file, fileInfo.Size(), absFilePath), nil
	}
}
