package resumablerecorder

import (
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/gofrs/flock"
	"modernc.org/fileutil"
)

type (
	jsonFileSystemResumableRecorder struct {
		dirPath string
	}
	jsonFileSystemResumableRecorderReadableMedium struct {
		file    *os.File
		decoder *json.Decoder
	}
	jsonFileSystemResumableRecorderWritableMedium struct {
		file    *os.File
		encoder *json.Encoder
	}
)

const jsonFileSystemResumableRecorderLock = "json_file_system_resumable_recorder_01.lock"

// 创建记录文件系统的可恢复记录仪
func NewJsonFileSystemResumableRecorder(dirPath string) ResumableRecorder {
	return jsonFileSystemResumableRecorder{dirPath}
}

func (frr jsonFileSystemResumableRecorder) OpenForReading(options *ResumableRecorderOpenArgs) ReadableResumableRecorderMedium {
	if options == nil {
		options = &ResumableRecorderOpenArgs{}
	}
	if options.DestinationID == "" {
		return nil
	}

	err := os.MkdirAll(frr.dirPath, 0700)
	if err != nil {
		return nil
	}
	file, err := os.Open(frr.getFilePath(options))
	if err != nil {
		return nil
	}
	_ = fileutil.Fadvise(file, 0, 0, fileutil.POSIX_FADV_SEQUENTIAL)
	decoder := json.NewDecoder(file)
	if verified, err := jsonFileSystemResumableRecorderVerifyHeaderLine(decoder, options); err != nil || !verified {
		return nil
	}
	return jsonFileSystemResumableRecorderReadableMedium{file, decoder}
}

func (frr jsonFileSystemResumableRecorder) OpenForAppending(options *ResumableRecorderOpenArgs) WriteableResumableRecorderMedium {
	if options == nil {
		options = &ResumableRecorderOpenArgs{}
	}
	if options.DestinationID == "" {
		return nil
	}

	file, err := os.OpenFile(frr.getFilePath(options), os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil
	}
	return jsonFileSystemResumableRecorderWritableMedium{file, json.NewEncoder(file)}
}

func (frr jsonFileSystemResumableRecorder) OpenForCreatingNew(options *ResumableRecorderOpenArgs) WriteableResumableRecorderMedium {
	if options == nil {
		options = &ResumableRecorderOpenArgs{}
	}
	if options.DestinationID == "" {
		return nil
	}

	file, err := os.OpenFile(frr.getFilePath(options), os.O_WRONLY|os.O_APPEND|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil
	}
	encoder := json.NewEncoder(file)
	if err := jsonFileSystemResumableRecorderWriteHeaderLine(encoder, options); err != nil {
		return nil
	}
	return jsonFileSystemResumableRecorderWritableMedium{file, encoder}
}

func (frr jsonFileSystemResumableRecorder) Delete(options *ResumableRecorderOpenArgs) error {
	return os.Remove(frr.getFilePath(options))
}

func (frr jsonFileSystemResumableRecorder) ClearOutdated(createdBefore time.Duration) error {
	jsonFileSystemResumableRecorderLockFilePath := filepath.Join(frr.dirPath, jsonFileSystemResumableRecorderLock)
	lock := flock.New(jsonFileSystemResumableRecorderLockFilePath)
	locked, err := lock.TryLock()
	if err != nil {
		return err
	} else if !locked {
		return nil
	}
	defer lock.Unlock()

	fileInfos, err := ioutil.ReadDir(frr.dirPath)
	if err != nil {
		return err
	}
	for _, fileInfo := range fileInfos {
		if !fileInfo.Mode().IsRegular() {
			continue
		}
		if fileInfo.Name() == jsonFileSystemResumableRecorderLock {
			continue
		}
		filePath := filepath.Join(frr.dirPath, fileInfo.Name())
		if err = frr.tryToClearPath(createdBefore, filePath); err != nil {
			os.Remove(filePath)
		}
	}
	return nil
}

func (frr jsonFileSystemResumableRecorder) tryToClearPath(createdBefore time.Duration, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var lineOptions jsonBasedResumableRecorderOpenArgs
	if err = decoder.Decode(&lineOptions); err != nil {
		return nil
	}
	if lineOptions.Version != fileSystemResumableRecorderVersion {
		return nil
	}
	if time.Now().Before(time.Unix(lineOptions.CreatedAt, 0).Add(createdBefore)) {
		return nil
	}
	return errors.New("resumable recorder is expired")
}

func (frr jsonFileSystemResumableRecorder) fileName(options *ResumableRecorderOpenArgs) string {
	hasher := sha1.New()
	hasher.Write([]byte(options.DestinationID))
	hasher.Write([]byte{0})
	hasher.Write([]byte(options.ETag))
	hasher.Write([]byte{0})
	hasher.Write([]byte{0})
	binary.Write(hasher, binary.LittleEndian, options.Offset)
	binary.Write(hasher, binary.LittleEndian, options.PartSize)
	binary.Write(hasher, binary.LittleEndian, options.TotalSize)
	return hex.EncodeToString(hasher.Sum(nil))
}

func (frr jsonFileSystemResumableRecorder) getFilePath(options *ResumableRecorderOpenArgs) string {
	return filepath.Join(frr.dirPath, frr.fileName(options))
}

type (
	jsonBasedResumableRecorderOpenArgs struct {
		ETag          string `json:"e,omitempty"`
		DestinationID string `json:"d,omitempty"`
		PartSize      uint64 `json:"p,omitempty"`
		TotalSize     uint64 `json:"t,omitempty"`
		Offset        uint64 `json:"o,omitempty"`
		CreatedAt     int64  `json:"c,omitempty"`
		Version       uint32 `json:"v,omitempty"`
	}

	jsonBasedResumableRecord struct {
		Offset      uint64 `json:"o,omitempty"`
		PartSize    uint64 `json:"s,omitempty"`
		PartWritten uint64 `json:"w,omitempty"`
	}
)

const fileSystemResumableRecorderVersion uint32 = 1

func jsonFileSystemResumableRecorderWriteHeaderLine(encoder *json.Encoder, options *ResumableRecorderOpenArgs) error {
	return encoder.Encode(&jsonBasedResumableRecorderOpenArgs{
		ETag:          options.ETag,
		DestinationID: options.DestinationID,
		PartSize:      options.PartSize,
		TotalSize:     options.TotalSize,
		Offset:        options.Offset,
		CreatedAt:     time.Now().Unix(),
		Version:       fileSystemResumableRecorderVersion,
	})
}

func jsonFileSystemResumableRecorderVerifyHeaderLine(decoder *json.Decoder, options *ResumableRecorderOpenArgs) (bool, error) {
	var lineOptions jsonBasedResumableRecorderOpenArgs
	err := decoder.Decode(&lineOptions)
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(lineOptions, jsonBasedResumableRecorderOpenArgs{
		ETag:          options.ETag,
		DestinationID: options.DestinationID,
		PartSize:      options.PartSize,
		TotalSize:     options.TotalSize,
		Offset:        options.Offset,
		CreatedAt:     lineOptions.CreatedAt,
		Version:       fileSystemResumableRecorderVersion,
	}), nil
}

func (medium jsonFileSystemResumableRecorderReadableMedium) Next(rr *ResumableRecord) error {
	var jrr jsonBasedResumableRecord
	for {
		if err := medium.decoder.Decode(&jrr); err != nil {
			return err
		} else {
			break
		}
	}

	*rr = ResumableRecord(jrr)
	return nil
}

func (medium jsonFileSystemResumableRecorderReadableMedium) Close() error {
	return medium.file.Close()
}

func (medium jsonFileSystemResumableRecorderWritableMedium) Write(rr *ResumableRecord) error {
	return medium.encoder.Encode(jsonBasedResumableRecord(*rr))
}

func (medium jsonFileSystemResumableRecorderWritableMedium) Close() error {
	return medium.file.Close()
}
