package uploader

import (
	"github.com/qiniu/go-sdk/v7/storagev2/region"
	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"
)

type (
	// 对象上传选项
	ObjectOptions struct {
		// 区域提供者，可选
		RegionsProvider region.RegionsProvider

		// 上传凭证接口，可选
		UpToken uptoken.Provider

		// 空间名称，可选
		BucketName string

		// 对象名称
		ObjectName *string

		// 文件名称
		FileName string

		// 文件 MIME 类型
		ContentType string

		// 自定义元数据
		Metadata map[string]string

		// 自定义变量
		CustomVars map[string]string

		// 对象上传进度
		OnUploadingProgress func(uploaded, totalSize uint64)
	}

	// 分片上传对象上传选项
	MultiPartsObjectOptions struct {
		// 对象上传选项
		*ObjectOptions

		// 分片大小，如果不填写，默认为 4 MB
		PartSize uint64
	}

	UploadPartsOptions struct {
		OnUploadingProgress func(partNumber, uploaded, partSize uint64)
		OnPartUploaded      func(partNumber, partSize uint64)
	}

	UploadPartOptions struct {
		OnUploadingProgress func(uploaded, partSize uint64)
	}

	DirectoryOptions struct {
		RegionsProvider       region.RegionsProvider
		UpToken               uptoken.Provider
		BucketName            string
		ObjectPrefix          string
		FileConcurrency       int
		BeforeFileUpload      func(filePath string, objectOptions *ObjectOptions)
		OnUploadingProgress   func(filePath string, uploaded, totalSize uint64)
		OnFileUploaded        func(filePath string, size uint64)
		ShouldCreateDirectory bool
		ShouldUploadFile      func(filePath string) bool
	}
)
