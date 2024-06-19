package destination

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
	resumablerecorder "github.com/qiniu/go-sdk/v7/storagev2/downloader/resumable_recorder"
)

type (
	SliceOptions struct {
		// 只读可恢复记录仪介质
		Medium resumablerecorder.ReadableResumableRecorderMedium
	}

	// 数据目标
	Destination interface {
		PartReader
		io.Closer

		// 切片
		Slice(totalSize, partSize uint64, options *SliceOptions) ([]Part, error)

		// 数据目标 KEY
		DestinationKey() (string, error)
	}

	// 分片
	Part interface {
		PartReader

		// 分片大小
		Size() uint64

		// 分片偏移量
		Offset() uint64

		// 分片偏移量
		HaveDownloaded() uint64
	}

	// 分片读取接口
	PartReader interface {
		// 从 `io.Reader` 复制数据写入
		CopyFrom(io.Reader, func(uint64)) (uint64, error)
	}

	writerAtDestination struct {
		wr             WriteAtCloser
		destinationKey string
	}

	writerAtPart struct {
		*internal_io.OffsetWriter
		offset, totalSize, restSize uint64
	}

	writeCloserDestination struct {
		wr             io.WriteCloser
		destinationKey string
	}

	writeCloserPart struct {
		wr                  io.Writer
		totalSize, restSize uint64
	}

	WriteAtCloser interface {
		io.WriterAt
		io.WriteSeeker
		io.Closer
	}
)

func NewWriteCloserDestination(wr io.WriteCloser, destinationKey string) Destination {
	return &writeCloserDestination{wr, destinationKey}
}

func (wcd *writeCloserDestination) CopyFrom(r io.Reader, progress func(uint64)) (uint64, error) {
	return copyBuffer(wcd.wr, r, progress)
}

func (wcd *writeCloserDestination) Slice(totalSize, _ uint64, _ *SliceOptions) ([]Part, error) {
	return []Part{&writeCloserPart{wcd.wr, totalSize, totalSize}}, nil
}

func (wcd *writeCloserDestination) DestinationKey() (string, error) {
	return wcd.destinationKey, nil
}

func (wcd *writeCloserDestination) Close() error {
	return wcd.wr.Close()
}

func (wcp *writeCloserPart) Size() uint64 {
	return wcp.totalSize
}

func (wcp *writeCloserPart) Offset() uint64 {
	return 0
}

func (wcp *writeCloserPart) HaveDownloaded() uint64 {
	return 0
}

var errInvalidWrite = errors.New("invalid write result")

func (wcp *writeCloserPart) CopyFrom(r io.Reader, progress func(uint64)) (uint64, error) {
	var newProgress func(uint64)

	if wcp.restSize == 0 {
		return 0, nil
	}

	haveCopied := wcp.HaveDownloaded()
	if progress != nil {
		newProgress = func(downloaded uint64) { progress(haveCopied + downloaded) }
	}
	n, err := copyBuffer(wcp.wr, io.LimitReader(r, int64(wcp.restSize)), newProgress)
	if n > 0 {
		wcp.restSize -= n
	}
	return n, err
}

func NewWriteAtCloserDestination(wr WriteAtCloser, destinationKey string) Destination {
	return &writerAtDestination{wr, destinationKey}
}

func (wad *writerAtDestination) CopyFrom(r io.Reader, progress func(uint64)) (uint64, error) {
	n, err := io.Copy(wad.wr, r)
	return uint64(n), err
}

func (wad *writerAtDestination) Slice(totalSize, partSize uint64, options *SliceOptions) ([]Part, error) {
	var (
		parts           []Part
		offsetMap       = make(map[uint64]uint64)
		resumableRecord resumablerecorder.ResumableRecord
		err             error
	)
	if options == nil {
		options = &SliceOptions{}
	}

	if medium := options.Medium; medium != nil {
		for {
			if err = medium.Next(&resumableRecord); err != nil {
				break
			}
			offsetMap[resumableRecord.Offset] = resumableRecord.PartWritten
		}
	}

	parts = make([]Part, 0, (totalSize+partSize-1)/partSize)
	for offset := uint64(0); offset < totalSize; offset += partSize {
		size := partSize
		if size > (totalSize - offset) {
			size = totalSize - offset
		}
		haveWritten := offsetMap[offset]
		parts = append(parts, &writerAtPart{internal_io.NewOffsetWriter(wad.wr, int64(offset+haveWritten)), offset, size, size - haveWritten})
	}
	return parts, nil
}

func (wad *writerAtDestination) DestinationKey() (string, error) {
	return wad.destinationKey, nil
}

func (wad *writerAtDestination) Close() error {
	return wad.wr.Close()
}

func (w *writerAtPart) Size() uint64 {
	return w.totalSize
}

func (w *writerAtPart) Offset() uint64 {
	return w.offset
}

func (w *writerAtPart) HaveDownloaded() uint64 {
	return w.totalSize - w.restSize
}

func (w *writerAtPart) CopyFrom(r io.Reader, progress func(uint64)) (uint64, error) {
	var newProgress func(uint64)

	if w.restSize == 0 {
		return 0, nil
	}

	haveCopied := w.HaveDownloaded()
	if progress != nil {
		newProgress = func(downloaded uint64) { progress(haveCopied + downloaded) }
	}
	n, err := copyBuffer(w.OffsetWriter, io.LimitReader(r, int64(w.restSize)), newProgress)
	if n > 0 {
		w.restSize -= n
	}
	return n, err
}

func copyBuffer(w io.Writer, r io.Reader, progress func(uint64)) (uint64, error) {
	const BUFSIZE = 32 * 1024
	var (
		buf         = make([]byte, BUFSIZE)
		haveCopied  uint64
		nr, nw      int
		er, ew, err error
	)
	for {
		nr, er = r.Read(buf)
		if nr > 0 {
			nw, ew = w.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errInvalidWrite
				}
			}
			haveCopied += uint64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
			if progress != nil {
				progress(haveCopied)
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return haveCopied, err
}

func NewFileDestination(filePath string) (Destination, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return nil, err
	}
	if !canSeekReally(file) {
		return NewWriteCloserDestination(file, ""), nil
	} else if absFilePath, err := filepath.Abs(filePath); err != nil {
		return nil, err
	} else {
		return NewWriteAtCloserDestination(file, absFilePath), nil
	}
}

func canSeekReally(seeker io.Seeker) bool {
	_, err := seeker.Seek(0, io.SeekCurrent)
	return err == nil
}
