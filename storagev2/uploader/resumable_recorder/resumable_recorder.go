package resumablerecorder

import (
	"crypto/md5"
	"io"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/region"
)

type (
	ResumableRecorderOpenOptions struct {
		BucketName, ObjectName, SourceKey string
		PartSize, TotalSize               uint64
		UpEndpoints                       region.Endpoints
	}

	ResumableRecorder interface {
		OpenForReading(*ResumableRecorderOpenOptions) ReadableResumableRecorderMedium
		OpenForAppending(*ResumableRecorderOpenOptions) WriteableResumableRecorderMedium
		OpenForCreatingNew(*ResumableRecorderOpenOptions) WriteableResumableRecorderMedium
		Delete(*ResumableRecorderOpenOptions) error
	}

	ReadableResumableRecorderMedium interface {
		io.Closer
		Next(*ResumableRecord) error
	}

	WriteableResumableRecorderMedium interface {
		io.Closer
		Write(*ResumableRecord) error
	}

	ResumableRecord struct {
		UploadId   string
		PartId     string
		Offset     uint64
		PartNumber uint64
		ExpiredAt  time.Time
		Crc32      uint32
		MD5        [md5.Size]byte
	}
)
