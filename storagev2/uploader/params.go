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
		ObjectOptions

		// 分片大小，如果不填写，默认为 4 MB
		PartSize uint64
	}

	// 上传分片列表选项
	UploadPartsOptions struct {
		// 分片上传进度
		OnUploadingProgress func(partNumber, uploaded, partSize uint64)
		// 分片上传成功后回调函数
		OnPartUploaded func(partNumber, partSize uint64)
	}

	// 上传分片选项
	UploadPartOptions struct {
		// 分片上传进度
		OnUploadingProgress func(uploaded, partSize uint64)
	}

	DirectoryOptions struct {
		// 区域提供者
		RegionsProvider region.RegionsProvider

		// 上传凭证
		UpToken uptoken.Provider

		// 空间名称
		BucketName string

		// 上传并发度
		ObjectConcurrency int

		// 上传前回调函数
		BeforeObjectUpload func(filePath string, objectOptions *ObjectOptions)

		// 上传进度
		OnUploadingProgress func(filePath string, uploaded, totalSize uint64)

		// 对象上传成功后回调
		OnObjectUploaded func(filePath string, size uint64)

		// 是否在空间内创建目录
		ShouldCreateDirectory bool

		// 是否上传指定对象
		ShouldUploadObject func(filePath string, objectOptions *ObjectOptions) bool

		// 更改对象名称
		UpdateObjectName func(string) string

		// 分隔符，默认为 /
		PathSeparator string
	}
)
