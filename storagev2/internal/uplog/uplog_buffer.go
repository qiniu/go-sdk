package uplog

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/flock"
)

var (
	uplogDisabled bool
	uplogChan     chan<- uplogSerializedEntry

	uplogFileBufferDirPath      string
	uplogFileBufferDirPathMutex sync.Mutex

	uplogFileBuffer               *os.File
	uplogFileBufferFileLocker     *flock.Flock
	uplogFileBufferLock           sync.Mutex
	uplogFileBufferThreshold      int64 = 4 * 1024 * 1024
	uplogWriteFileBufferTicker    *time.Ticker
	uplogWriteFileBufferInterval  time.Duration = 1 * time.Minute
	uplogWriteFileBufferTimerLock sync.Mutex

	getUpToken GetUpToken
)

const (
	UPLOG_CHANNEL_SIZE       = 1024
	UPLOG_MEMORY_BUFFER_SIZE = 4 * 1024
	UPLOG_FILE_BUFFER_NAME   = "uplog_v4_01.buffer"
	UPLOG_FILE_LOCK_NAME     = "uplog_v4_01.lock"
)

type uplogSerializedEntry struct {
	serializedUplog []byte
	getUpToken      GetUpToken
}

func init() {
	uplogChannel := make(chan uplogSerializedEntry, UPLOG_CHANNEL_SIZE)
	uplogWriteFileBufferTicker = time.NewTicker(uplogWriteFileBufferInterval)
	go func() {
		for {
			select {
			case serializedEntry := <-uplogChannel:
				if uplogDisabled {
					continue
				}
				if gut := serializedEntry.getUpToken; gut != nil {
					getUpToken = gut
				}
				uplogBuffer := bytes.NewBuffer(make([]byte, 0, UPLOG_MEMORY_BUFFER_SIZE))
				uplogBuffer.Write(serializedEntry.serializedUplog)
				uplogBuffer.WriteString("\n")
				for uplogBuffer.Len() < (UPLOG_MEMORY_BUFFER_SIZE / 2) {
					select {
					case serializedEntry := <-uplogChannel:
						uplogBuffer.Write(serializedEntry.serializedUplog)
						uplogBuffer.WriteString("\n")
					default:
						goto finishReading
					}
				}
			finishReading:
				_, _ = writeMemoryBufferToFileBuffer(uplogBuffer.Bytes())
			case <-uplogWriteFileBufferTicker.C:
				if fi, err := os.Stat(getUplogFileBufferPath(true)); err == nil && fi.Size() > 0 {
					tryToArchiveFileBuffer(false)
				}
			}
		}
	}()
	uplogChan = uplogChannel
}

func DisableUplog() {
	uplogDisabled = true
}

func EnableUplog() {
	uplogDisabled = false
}

func IsUplogEnabled() bool {
	return !uplogDisabled
}

func SetUplogFileBufferDirPath(path string) {
	uplogFileBufferDirPathMutex.Lock()
	defer uplogFileBufferDirPathMutex.Unlock()
	uplogFileBufferDirPath = path
}

func getUplogFileBufferPath(current bool) string {
	var uplogFileBufferPath string

	uplogFileBufferDirPathMutex.Lock()
	defer uplogFileBufferDirPathMutex.Unlock()

	if uplogFileBufferDirPath == "" {
		uplogFileBufferPath = filepath.Join(os.TempDir(), "qiniu-golang-sdk", UPLOG_FILE_BUFFER_NAME)
	} else {
		uplogFileBufferPath = filepath.Join(uplogFileBufferDirPath, UPLOG_FILE_BUFFER_NAME)
	}
	if !current {
		uplogFileBufferPath = uplogFileBufferPath + "." + time.Now().UTC().Format(time.RFC3339Nano)
	}
	return uplogFileBufferPath
}

func getUplogFileDirectoryLock() *flock.Flock {
	uplogFileBufferDirPathMutex.Lock()
	defer uplogFileBufferDirPathMutex.Unlock()
	var uplogFileLockPath string
	if uplogFileBufferDirPath == "" {
		uplogFileLockPath = filepath.Join(os.TempDir(), "qiniu-golang-sdk", UPLOG_FILE_LOCK_NAME)
	} else {
		uplogFileLockPath = filepath.Join(uplogFileBufferDirPath, UPLOG_FILE_LOCK_NAME)
	}
	return flock.New(uplogFileLockPath)
}

func FlushBuffer() error {
	return withUploadFileBuffer(func(io.Writer) (bool, error) {
		return true, nil
	})
}

func writeMemoryBufferToFileBuffer(data []byte) (n int, err error) {
	if err = withUploadFileBuffer(func(w io.Writer) (shouldClose bool, e error) {
		for len(data) > 0 {
			n, e = w.Write(data)
			if e != nil {
				return
			}
			data = data[n:]
		}
		return
	}); err != nil {
		return
	}

	if fi, serr := os.Stat(getUplogFileBufferPath(true)); serr == nil && fi.Size() >= uplogFileBufferThreshold {
		tryToArchiveFileBuffer(false)
	}
	return
}

func tryToArchiveFileBuffer(force bool) {
	var (
		locked bool
		err    error
	)

	if fileInfo, fileInfoErr := os.Stat(getUplogFileBufferPath(true)); fileInfoErr == nil && fileInfo.Size() == 0 {
		return
	}

	locker := getUplogFileDirectoryLock()
	if force {
		if err = locker.Lock(); err != nil {
			return
		}
	} else {
		if locked, err = locker.TryLock(); err != nil || !locked {
			return
		}
	}
	defer locker.Close()

	if err = withUploadFileBuffer(func(io.Writer) (shouldClose bool, renameErr error) {
		currentFilePath := getUplogFileBufferPath(true)
		if fileInfo, fileInfoErr := os.Stat(currentFilePath); fileInfoErr == nil && fileInfo.Size() == 0 {
			return
		}
		archivedFilePath := getUplogFileBufferPath(false)
		if renameErr = os.Rename(currentFilePath, archivedFilePath); renameErr == nil {
			shouldClose = true
		}
		return
	}); err != nil {
		return
	}

	resetWriteFileBufferInterval()
	go uploadAllClosedFileBuffers()
}

func closeUplogFileBufferWithoutLock() {
	uplogFileBuffer.Close()
	uplogFileBuffer = nil
	uplogFileBufferFileLocker.Close()
	uplogFileBufferFileLocker = nil
}

func withUploadFileBuffer(fn func(io.Writer) (bool, error)) (err error) {
	var shouldClose bool

	uplogFileBufferLock.Lock()
	defer uplogFileBufferLock.Unlock()

	if uplogFileBuffer != nil {
		if _, err := os.Stat(uplogFileBuffer.Name()); err != nil && os.IsNotExist(err) {
			closeUplogFileBufferWithoutLock()
		}
	}

	if uplogFileBuffer == nil {
		uplogFileBufferPath := getUplogFileBufferPath(true)
		if err = os.MkdirAll(filepath.Dir(uplogFileBufferPath), 0755); err != nil {
			return
		} else if uplogFileBuffer, err = os.OpenFile(uplogFileBufferPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err != nil {
			return
		}
		uplogFileBufferFileLocker = flock.New(uplogFileBuffer.Name())
	}

	if err = uplogFileBufferFileLocker.Lock(); err != nil {
		return
	}
	shouldClose, err = fn(uplogFileBuffer)
	_ = uplogFileBufferFileLocker.Unlock()
	if shouldClose {
		closeUplogFileBufferWithoutLock()
	}
	return
}

func ResetWriteFileBufferInterval(d time.Duration) {
	uplogWriteFileBufferTimerLock.Lock()
	defer uplogWriteFileBufferTimerLock.Unlock()
	uplogWriteFileBufferInterval = d
	uplogWriteFileBufferTicker.Reset(d)
}

func resetWriteFileBufferInterval() {
	uplogWriteFileBufferTimerLock.Lock()
	defer uplogWriteFileBufferTimerLock.Unlock()
	uplogWriteFileBufferTicker.Reset(uplogWriteFileBufferInterval)
}

type multipleFileReader struct {
	paths []string

	compressor *gzip.Writer
	w          *io.PipeWriter
	r          *io.PipeReader
	err        error
	errLock    sync.Mutex
	wg         sync.WaitGroup
}

func newMultipleFileReader(paths []string) (*multipleFileReader, error) {
	r, w := io.Pipe()
	compressor, err := gzip.NewWriterLevel(w, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	mfr := &multipleFileReader{
		paths:      paths,
		r:          r,
		w:          w,
		compressor: compressor,
	}
	mfr.wg.Add(1)
	go mfr.readAllAsync()
	return mfr, nil
}

func (r *multipleFileReader) readAllAsync() {
	defer r.wg.Done()
	defer r.w.CloseWithError(io.EOF)
	defer r.compressor.Close()
	for _, path := range r.paths {
		file, err := os.Open(path)
		if err != nil {
			r.setError(err)
			return
		}
		if _, err = io.Copy(r.compressor, file); err != nil && err != io.EOF {
			r.setError(err)
			return
		}
	}
}

func (r *multipleFileReader) getError() error {
	r.errLock.Lock()
	defer r.errLock.Unlock()
	return r.err
}

func (r *multipleFileReader) setError(err error) {
	r.errLock.Lock()
	defer r.errLock.Unlock()
	r.err = err
}

func (r *multipleFileReader) Read(p []byte) (int, error) {
	if err := r.getError(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}

func (r *multipleFileReader) Close() error {
	err := r.r.CloseWithError(io.EOF)
	r.wg.Wait()
	return err
}

func getArchivedUplogFileBufferPaths(dirPath string) ([]string, error) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	archivedPaths := make([]string, 0, len(dirEntries))
	for _, dirEntry := range dirEntries {
		if !dirEntry.Type().IsRegular() {
			continue
		}
		if !strings.HasPrefix(dirEntry.Name(), UPLOG_FILE_BUFFER_NAME+".") {
			continue
		}
		archivedPaths = append(archivedPaths, filepath.Join(dirPath, dirEntry.Name()))
	}
	return archivedPaths, nil
}
