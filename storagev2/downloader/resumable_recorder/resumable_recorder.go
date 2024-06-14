package resumablerecorder

import "io"

type (
	// 可恢复记录仪选项
	ResumableRecorderOpenOptions struct {
		// 数据源 ETag
		ETag string

		// 数据目标 KEY
		DestinationKey string

		// 分片大小
		PartSize uint64

		// 数据源大小
		TotalSize uint64
	}

	// 可恢复记录仪接口
	ResumableRecorder interface {
		// 打开记录仪介质以读取记录
		OpenForReading(*ResumableRecorderOpenOptions) ReadableResumableRecorderMedium

		// 打开记录仪介质以追加记录
		OpenForAppending(*ResumableRecorderOpenOptions) WriteableResumableRecorderMedium

		// 新建记录仪介质以追加记录
		OpenForCreatingNew(*ResumableRecorderOpenOptions) WriteableResumableRecorderMedium

		// 删除记录仪介质
		Delete(*ResumableRecorderOpenOptions) error
	}

	// 只读的可恢复记录仪介质接口
	ReadableResumableRecorderMedium interface {
		io.Closer

		// 读取下一条记录
		Next(*ResumableRecord) error
	}

	// 只追家的可恢复记录仪介质接口
	WriteableResumableRecorderMedium interface {
		io.Closer

		// 写入下一条记录
		Write(*ResumableRecord) error
	}

	// 可恢复记录
	ResumableRecord struct {
		// 分片偏移量
		Offset uint64

		// 分片大小
		PartSize uint64

		// 分片写入量
		PartWritten uint64
	}
)
