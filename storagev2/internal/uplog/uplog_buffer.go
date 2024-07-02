package uplog

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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
	uplogFileBufferThreshold      uint64 = 4 * 1024 * 1024
	uplogMaxStorageBytes          uint64 = 100 * 1024 * 1024
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

func GetUplogMaxStorageBytes() uint64 {
	return atomic.LoadUint64(&uplogMaxStorageBytes)
}

func SetUplogMaxStorageBytes(max uint64) {
	atomic.StoreUint64(&uplogMaxStorageBytes, max)
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
		uplogFileBufferPath = uplogFileBufferPath + "." + time.Now().UTC().Format("20060102150405.999999999")
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
	return withUploadFileBuffer(func(w io.WriteCloser) error {
		return w.Close()
	})
}

func writeMemoryBufferToFileBuffer(data []byte) (n int, err error) {
	if err = withUploadFileBuffer(func(w io.WriteCloser) (e error) {
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

	if fi, serr := os.Stat(getUplogFileBufferPath(true)); serr == nil && uint64(fi.Size()) >= uplogFileBufferThreshold {
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

	if err = withUploadFileBuffer(func(w io.WriteCloser) error {
		currentFilePath := getUplogFileBufferPath(true)
		if fileInfo, fileInfoErr := os.Stat(currentFilePath); fileInfoErr == nil && fileInfo.Size() == 0 {
			return nil
		}
		archivedFilePath := getUplogFileBufferPath(false)
		w.Close()
		return os.Rename(currentFilePath, archivedFilePath)
	}); err != nil {
		return
	}

	resetWriteFileBufferInterval()
	go uploadAllClosedFileBuffers()
}

type uplogFileBufferWrapper struct{}

func (uplogFileBufferWrapper) Write(p []byte) (int, error) {
	return uplogFileBuffer.Write(p)
}

func (uplogFileBufferWrapper) Close() error {
	return closeUplogFileBufferWithoutLock()
}

func withUploadFileBuffer(fn func(io.WriteCloser) error) (err error) {
	uplogFileBufferLock.Lock()
	defer uplogFileBufferLock.Unlock()

	uplogFileBufferPath := getUplogFileBufferPath(true)

	if uplogFileBuffer != nil {
		if uplogFileBuffer.Name() != uplogFileBufferPath {
			closeUplogFileBufferWithoutLock()
		} else if _, err := os.Stat(uplogFileBufferPath); err != nil && os.IsNotExist(err) {
			closeUplogFileBufferWithoutLock()
		}
	}

	if uplogFileBuffer == nil {
		if err = os.MkdirAll(filepath.Dir(uplogFileBufferPath), 0755); err != nil {
			return
		} else if uplogFileBuffer, err = os.OpenFile(uplogFileBufferPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err != nil {
			return
		}
		uplogFileBufferLockPath := uplogFileBufferPath + ".lock"
		uplogFileBufferFileLocker = flock.New(uplogFileBufferLockPath)
	}

	if err = uplogFileBufferFileLocker.Lock(); err != nil {
		return
	}
	err = fn(uplogFileBufferWrapper{})
	if uplogFileBufferFileLocker != nil && uplogFileBufferFileLocker.Locked() {
		_ = uplogFileBufferFileLocker.Unlock()
	}
	return
}

func closeUplogFileBufferWithoutLock() error {
	var err1, err2 error
	if uplogFileBuffer != nil {
		err1 = uplogFileBuffer.Close()
		uplogFileBuffer = nil
	}
	if uplogFileBufferFileLocker != nil {
		err2 = uplogFileBufferFileLocker.Close()
		uplogFileBufferFileLocker = nil
	}
	if err1 != nil {
		return err1
	}
	return err2
}

func SetWriteFileBufferInterval(d time.Duration) {
	uplogWriteFileBufferTimerLock.Lock()
	defer uplogWriteFileBufferTimerLock.Unlock()
	if d == 0 {
		d = 1 * time.Minute
	}
	uplogWriteFileBufferInterval = d
	uplogWriteFileBufferTicker.Stop()
	uplogWriteFileBufferTicker = time.NewTicker(d)
}

func GetWriteFileBufferInterval() time.Duration {
	uplogWriteFileBufferTimerLock.Lock()
	defer uplogWriteFileBufferTimerLock.Unlock()

	return uplogWriteFileBufferInterval
}

func resetWriteFileBufferInterval() {
	uplogWriteFileBufferTimerLock.Lock()
	defer uplogWriteFileBufferTimerLock.Unlock()
	uplogWriteFileBufferTicker.Stop()
	uplogWriteFileBufferTicker = time.NewTicker(uplogWriteFileBufferInterval)
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
		if err := r.readAllForPathAsync(path); err != nil {
			r.setError(err)
			return
		}
	}
}

func (r *multipleFileReader) readAllForPathAsync(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err = io.Copy(r.compressor, file); err != nil && err != io.EOF {
		return err
	}
	return nil
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
	dirEntries, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	archivedPaths := make([]string, 0, len(dirEntries))
	for _, dirEntry := range dirEntries {
		if !dirEntry.Mode().IsRegular() ||
			!strings.HasPrefix(dirEntry.Name(), UPLOG_FILE_BUFFER_NAME+".") ||
			strings.HasSuffix(dirEntry.Name(), ".lock") {
			continue
		}
		archivedPaths = append(archivedPaths, filepath.Join(dirPath, dirEntry.Name()))
	}
	return archivedPaths, nil
}
