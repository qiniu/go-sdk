package resumablerecorder

import (
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"

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

// 创建记录文件系统的可恢复记录仪
func NewJsonFileSystemResumableRecorder(dirPath string) ResumableRecorder {
	return jsonFileSystemResumableRecorder{dirPath}
}

func (frr jsonFileSystemResumableRecorder) OpenForReading(options *ResumableRecorderOpenOptions) ReadableResumableRecorderMedium {
	if options == nil {
		options = &ResumableRecorderOpenOptions{}
	}
	if options.DestinationKey == "" {
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
	if options.DestinationKey == "" {
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
	if options.DestinationKey == "" {
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

func (frr jsonFileSystemResumableRecorder) fileName(options *ResumableRecorderOpenOptions) string {
	hasher := sha1.New()
	hasher.Write([]byte(options.DestinationKey))
	hasher.Write([]byte{0})
	hasher.Write([]byte(options.ETag))
	hasher.Write([]byte{0})
	for _, downloadURL := range options.DownloadURLs {
		hasher.Write([]byte(downloadURL))
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
		ETag           string   `json:"e,omitempty"`
		DestinationKey string   `json:"d,omitempty"`
		PartSize       uint64   `json:"p,omitempty"`
		TotalSize      uint64   `json:"t,omitempty"`
		DownloadURLs   []string `json:"u,omitempty"`
		Version        uint32   `json:"v,omitempty"`
	}

	jsonBasedResumableRecord struct {
		Offset      uint64 `json:"o,omitempty"`
		PartSize    uint64 `json:"s,omitempty"`
		PartWritten uint64 `json:"w,omitempty"`
	}
)

const fileSystemResumableRecorderVersion uint32 = 1

func jsonFileSystemResumableRecorderWriteHeaderLine(encoder *json.Encoder, options *ResumableRecorderOpenOptions) error {
	return encoder.Encode(&jsonBasedResumableRecorderOpenOptions{
		ETag:           options.ETag,
		DestinationKey: options.DestinationKey,
		PartSize:       options.PartSize,
		TotalSize:      options.TotalSize,
		DownloadURLs:   options.DownloadURLs,
		Version:        fileSystemResumableRecorderVersion,
	})
}

func jsonFileSystemResumableRecorderVerifyHeaderLine(decoder *json.Decoder, options *ResumableRecorderOpenOptions) (bool, error) {
	var lineOptions jsonBasedResumableRecorderOpenOptions
	err := decoder.Decode(&lineOptions)
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(lineOptions, jsonBasedResumableRecorderOpenOptions{
		ETag:           options.ETag,
		DestinationKey: options.DestinationKey,
		PartSize:       options.PartSize,
		TotalSize:      options.TotalSize,
		DownloadURLs:   options.DownloadURLs,
		Version:        fileSystemResumableRecorderVersion,
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
