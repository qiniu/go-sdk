package uploader

import (
	"context"
	"io"

	"github.com/qiniu/go-sdk/v7/storagev2/uploader/source"
)

type (
	// 上传对象接口
	Uploader interface {
		// 上传文件
		UploadFile(context.Context, string, *ObjectParams, interface{}) error

		// 上传 io.Reader
		UploadReader(context.Context, io.Reader, *ObjectParams, interface{}) error
	}

	// 分片上传器接口
	MultiPartsUploader interface {
		// 初始化分片上传
		InitializeParts(context.Context, source.Source, *MultiPartsObjectParams) (InitializedParts, error)

		// 尝试恢复分片，如果返回 nil 表示恢复失败
		TryToResume(context.Context, source.Source, *MultiPartsObjectParams) InitializedParts

		// 上传分片
		UploadPart(context.Context, InitializedParts, source.Part, *UploadPartParams) (UploadedPart, error)

		// 完成分片上传，生成对象
		CompleteParts(context.Context, InitializedParts, []UploadedPart, interface{}) error

		// 获取分片上传选项
		MultiPartsUploaderOptions() *MultiPartsUploaderOptions
	}

	// 经过初始化的分片上传
	InitializedParts interface {
		// 关闭分片上传，InitializedParts 一旦使用完毕，无论是否成功，都应该调用该方法关闭
		io.Closer
	}

	// 已经上传的分片
	UploadedPart interface {
		// 分片偏移量
		Offset() uint64
	}

	// 分片上传调度器
	MultiPartsUploaderScheduler interface {
		// 上传数据源的全部分片
		UploadParts(context.Context, InitializedParts, source.Source, *UploadPartsParams) ([]UploadedPart, error)

		// 获取分片上传器实例
		MultiPartsUploader() MultiPartsUploader

		// 获取最大分片大小
		PartSize() uint64
	}
)
