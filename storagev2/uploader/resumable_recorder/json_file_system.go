package resumablerecorder

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/gofrs/flock"
	"github.com/qiniu/go-sdk/v7/storagev2/region"
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

func (frr jsonFileSystemResumableRecorder) OpenForReading(options *ResumableRecorderOpenOptions) ReadableResumableRecorderMedium {
	if options == nil {
		options = &ResumableRecorderOpenOptions{}
	}
	if options.SourceID == "" {
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

func (frr jsonFileSystemResumableRecorder) OpenForAppending(options *ResumableRecorderOpenOptions) WriteableResumableRecorderMedium {
	if options == nil {
		options = &ResumableRecorderOpenOptions{}
	}
	if options.SourceID == "" {
		return nil
	}

	file, err := os.OpenFile(frr.getFilePath(options), os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil
	}
	return jsonFileSystemResumableRecorderWritableMedium{file, json.NewEncoder(file)}
}

func (frr jsonFileSystemResumableRecorder) OpenForCreatingNew(options *ResumableRecorderOpenOptions) WriteableResumableRecorderMedium {
	if options == nil {
		options = &ResumableRecorderOpenOptions{}
	}
	if options.SourceID == "" {
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

func (frr jsonFileSystemResumableRecorder) Delete(options *ResumableRecorderOpenOptions) error {
	return os.Remove(frr.getFilePath(options))
}

func (frr jsonFileSystemResumableRecorder) ClearExpired() error {
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
		if err = frr.tryToClearPath(filePath); err != nil {
			os.Remove(filePath)
		}
	}
	return nil
}

func (frr jsonFileSystemResumableRecorder) tryToClearPath(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_ = fileutil.Fadvise(file, 0, 0, fileutil.POSIX_FADV_SEQUENTIAL)
	decoder := json.NewDecoder(file)
	var (
		lineOptions jsonBasedResumableRecorderOpenOptions
		jrr         jsonBasedResumableRecord
	)
	if err = decoder.Decode(&lineOptions); err != nil {
		return nil
	}
	if lineOptions.Version != fileSystemResumableRecorderVersion {
		return nil
	}
	for {
		if err := decoder.Decode(&jrr); err != nil {
			if err == io.EOF {
				break
			}
			return err
		} else if time.Now().Before(time.Unix(jrr.ExpiredAt, 0)) {
			return nil
		}
	}
	return errors.New("no valid resumable record")
}

func (frr jsonFileSystemResumableRecorder) fileName(options *ResumableRecorderOpenOptions) string {
	hasher := sha1.New()
	hasher.Write([]byte(options.SourceID))
	hasher.Write([]byte{0})
	hasher.Write([]byte(options.AccessKey))
	hasher.Write([]byte{0})
	hasher.Write([]byte(options.BucketName))
	hasher.Write([]byte{0})
	hasher.Write([]byte(options.ObjectName))
	hasher.Write([]byte{0})
	for _, endpoint := range options.UpEndpoints.Preferred {
		hasher.Write([]byte(endpoint))
		hasher.Write([]byte{1})
	}
	hasher.Write([]byte{0})
	for _, endpoint := range options.UpEndpoints.Alternative {
		hasher.Write([]byte(endpoint))
		hasher.Write([]byte{1})
	}
	hasher.Write([]byte{0})
	binary.Write(hasher, binary.LittleEndian, options.PartSize)
	binary.Write(hasher, binary.LittleEndian, options.TotalSize)
	return hex.EncodeToString(hasher.Sum(nil))
}

func (frr jsonFileSystemResumableRecorder) getFilePath(options *ResumableRecorderOpenOptions) string {
	return filepath.Join(frr.dirPath, frr.fileName(options))
}

type (
	jsonBasedResumableRecorderOpenOptions struct {
		AccessKey   string           `json:"a,omitempty"`
		BucketName  string           `json:"b,omitempty"`
		ObjectName  string           `json:"o,omitempty"`
		SourceID    string           `json:"s,omitempty"`
		PartSize    uint64           `json:"p,omitempty"`
		TotalSize   uint64           `json:"t,omitempty"`
		UpEndpoints region.Endpoints `json:"u,omitempty"`
		Version     uint32           `json:"v,omitempty"`
	}

	jsonBasedResumableRecord struct {
		UploadId   string `json:"u,omitempty"`
		PartId     string `json:"p,omitempty"`
		Offset     uint64 `json:"o,omitempty"`
		PartNumber uint64 `json:"n,omitempty"`
		PartSize   uint64 `json:"s,omitempty"`
		ExpiredAt  int64  `json:"e,omitempty"`
		Crc32      uint32 `json:"c,omitempty"`
		MD5        string `json:"m,omitempty"`
	}
)

const fileSystemResumableRecorderVersion uint32 = 1

func jsonFileSystemResumableRecorderWriteHeaderLine(encoder *json.Encoder, options *ResumableRecorderOpenOptions) error {
	return encoder.Encode(&jsonBasedResumableRecorderOpenOptions{
		AccessKey:   options.AccessKey,
		BucketName:  options.BucketName,
		ObjectName:  options.ObjectName,
		SourceID:    options.SourceID,
		PartSize:    options.PartSize,
		TotalSize:   options.TotalSize,
		UpEndpoints: options.UpEndpoints,
		Version:     fileSystemResumableRecorderVersion,
	})
}

func jsonFileSystemResumableRecorderVerifyHeaderLine(decoder *json.Decoder, options *ResumableRecorderOpenOptions) (bool, error) {
	var lineOptions jsonBasedResumableRecorderOpenOptions
	err := decoder.Decode(&lineOptions)
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(lineOptions, jsonBasedResumableRecorderOpenOptions{
		AccessKey:   options.AccessKey,
		BucketName:  options.BucketName,
		ObjectName:  options.ObjectName,
		SourceID:    options.SourceID,
		PartSize:    options.PartSize,
		TotalSize:   options.TotalSize,
		UpEndpoints: options.UpEndpoints,
		Version:     fileSystemResumableRecorderVersion,
	}), nil
}

func (medium jsonFileSystemResumableRecorderReadableMedium) Next(rr *ResumableRecord) error {
	var jrr jsonBasedResumableRecord
	for {
		if err := medium.decoder.Decode(&jrr); err != nil {
			return err
		} else if time.Now().Before(time.Unix(jrr.ExpiredAt, 0)) {
			break
		}
	}
	md5Bytes, err := hex.DecodeString(jrr.MD5)
	if err != nil {
		return err
	} else if len(md5Bytes) != md5.Size {
		return errors.New("invalid md5 bytes")
	}

	*rr = ResumableRecord{
		UploadId:   jrr.UploadId,
		PartId:     jrr.PartId,
		Offset:     jrr.Offset,
		PartNumber: jrr.PartNumber,
		PartSize:   jrr.PartSize,
		ExpiredAt:  time.Unix(jrr.ExpiredAt, 0),
		Crc32:      jrr.Crc32,
	}
	copy(rr.MD5[:], md5Bytes)
	return nil
}

func (medium jsonFileSystemResumableRecorderReadableMedium) Close() error {
	return medium.file.Close()
}

func (medium jsonFileSystemResumableRecorderWritableMedium) Write(rr *ResumableRecord) error {
	jrr := jsonBasedResumableRecord{
		UploadId:   rr.UploadId,
		PartId:     rr.PartId,
		Offset:     rr.Offset,
		PartNumber: rr.PartNumber,
		PartSize:   rr.PartSize,
		ExpiredAt:  rr.ExpiredAt.Unix(),
		Crc32:      rr.Crc32,
		MD5:        hex.EncodeToString(rr.MD5[:]),
	}
	return medium.encoder.Encode(&jrr)
}

func (medium jsonFileSystemResumableRecorderWritableMedium) Close() error {
	return medium.file.Close()
}
