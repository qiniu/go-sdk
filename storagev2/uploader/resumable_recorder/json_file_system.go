package resumablerecorder

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/region"
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

func NewJsonFileSystemResumableRecorder(dirPath string) ResumableRecorder {
	return jsonFileSystemResumableRecorder{dirPath}
}

func (frr jsonFileSystemResumableRecorder) OpenForReading(options *ResumableRecorderOpenOptions) ReadableResumableRecorderMedium {
	file, err := os.Open(frr.getFilePath(options))
	if err != nil {
		return nil
	}
	decoder := json.NewDecoder(file)
	if verified, err := jsonFileSystemResumableRecorderVerifyHeaderLine(decoder, options); err != nil || !verified {
		return nil
	}
	return jsonFileSystemResumableRecorderReadableMedium{file, decoder}
}

func (frr jsonFileSystemResumableRecorder) OpenForAppending(options *ResumableRecorderOpenOptions) WriteableResumableRecorderMedium {
	file, err := os.OpenFile(frr.getFilePath(options), os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil
	}
	return jsonFileSystemResumableRecorderWritableMedium{file, json.NewEncoder(file)}
}

func (frr jsonFileSystemResumableRecorder) OpenForCreatingNew(options *ResumableRecorderOpenOptions) WriteableResumableRecorderMedium {
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

func (frr jsonFileSystemResumableRecorder) fileName(options *ResumableRecorderOpenOptions) string {
	hasher := sha1.New()
	hasher.Write([]byte(options.SourceKey))
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
		BucketName  string           `json:"b,omitempty"`
		ObjectName  string           `json:"o,omitempty"`
		SourceKey   string           `json:"s,omitempty"`
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
		ExpiredAt  int64  `json:"e,omitempty"`
		Crc32      uint32 `json:"c,omitempty"`
		MD5        string `json:"m,omitempty"`
	}
)

const fileSystemResumableRecorderVersion uint32 = 1

func jsonFileSystemResumableRecorderWriteHeaderLine(encoder *json.Encoder, options *ResumableRecorderOpenOptions) error {
	return encoder.Encode(&jsonBasedResumableRecorderOpenOptions{
		BucketName:  options.BucketName,
		ObjectName:  options.ObjectName,
		SourceKey:   options.SourceKey,
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
		BucketName:  options.BucketName,
		ObjectName:  options.ObjectName,
		SourceKey:   options.SourceKey,
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
		} else if time.Now().Before(time.UnixMicro(jrr.ExpiredAt)) {
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
		ExpiredAt:  time.UnixMicro(jrr.ExpiredAt),
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
		ExpiredAt:  rr.ExpiredAt.UnixMicro(),
		Crc32:      rr.Crc32,
		MD5:        hex.EncodeToString(rr.MD5[:]),
	}
	return medium.encoder.Encode(&jrr)
}

func (medium jsonFileSystemResumableRecorderWritableMedium) Close() error {
	return medium.file.Close()
}
