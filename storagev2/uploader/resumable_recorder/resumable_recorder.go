package resumablerecorder

import (
	"crypto/md5"
	"io"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/region"
)

type (
	// 可恢复记录仪选项
	ResumableRecorderOpenArgs struct {
		// AccessKey
		AccessKey string

		// 空间名称
		BucketName string

		// 对象名称
		ObjectName string

		// 数据源 ID
		SourceID string

		// 分片大小
		PartSize uint64

		// 数据源大小
		TotalSize uint64

		// 上传服务 URL
		UpEndpoints region.Endpoints
	}

	// 可恢复记录仪接口
	ResumableRecorder interface {
		// 打开记录仪介质以读取记录
		OpenForReading(*ResumableRecorderOpenArgs) ReadableResumableRecorderMedium

		// 打开记录仪介质以追加记录
		OpenForAppending(*ResumableRecorderOpenArgs) WriteableResumableRecorderMedium

		// 新建记录仪介质以追加记录
		OpenForCreatingNew(*ResumableRecorderOpenArgs) WriteableResumableRecorderMedium

		// 删除记录仪介质
		Delete(*ResumableRecorderOpenArgs) error

		// 清理过期的记录仪介质
		ClearExpired() error
	}

	// 只读的可恢复记录仪介质接口
	ReadableResumableRecorderMedium interface {
		io.Closer

		// 读取下一条记录
		Next(*ResumableRecord) error
	}

	// 只追加的可恢复记录仪介质接口
	WriteableResumableRecorderMedium interface {
		io.Closer

		// 写入下一条记录
		Write(*ResumableRecord) error
	}

	// 可恢复记录
	ResumableRecord struct {
		// 上传对象 ID
		UploadID string

		// 上传分片 ID
		PartID string

		// 分片偏移量
		Offset uint64

		// 分片大小
		PartSize uint64

		// 分片编号
		PartNumber uint64

		// 分片过期时间
		ExpiredAt time.Time

		// 分片内容 CRC32
		CRC32 uint32

		// 分片内容 MD5
		MD5 [md5.Size]byte
	}
)
