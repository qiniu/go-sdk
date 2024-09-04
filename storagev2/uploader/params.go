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
		// 但如果不传值，则必须给出 BucketName，并且配合 Uploader 的 Credentials 自动生成 UpToken
		UpToken uptoken.Provider

		// 空间名称，可选，但如果不传值，则必须给出 UpToken
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
		OnUploadingProgress func(*UploadingProgress)
	}

	// 分片上传对象上传选项
	MultiPartsObjectOptions struct {
		// 对象上传选项
		ObjectOptions

		// 分片大小，如果不填写，默认为 4 MB
		PartSize uint64
	}

	// 分片上传进度
	UploadingPartProgress struct {
		Uploaded uint64 // 已经上传的数据量，单位为字节
		PartSize uint64 // 分片大小，单位为字节
	}

	// 对象上传进度
	UploadingProgress struct {
		Uploaded  uint64 // 已经上传的数据量，单位为字节
		TotalSize uint64 // 总数据量，单位为字节
	}

	// 上传分片列表选项
	UploadPartsOptions struct {
		// 分片上传进度
		OnUploadingProgress func(partNumber uint64, progress *UploadingPartProgress)
		// 分片上传成功后回调函数
		OnPartUploaded func(UploadedPart) error
	}

	// 上传分片选项
	UploadPartOptions struct {
		// 分片上传进度
		OnUploadingProgress func(*UploadingPartProgress)
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
		OnUploadingProgress func(filePath string, progress *UploadingProgress)

		// 对象上传成功后回调
		OnObjectUploaded func(filePath string, info *UploadedObjectInfo)

		// 是否在空间内创建目录
		ShouldCreateDirectory bool

		// 是否上传指定对象，如果 objectOptions 为 nil 则表示是目录
		ShouldUploadObject func(filePath string, objectOptions *ObjectOptions) bool

		// 更改对象名称
		UpdateObjectName func(string) string

		// 分隔符，默认为 /
		PathSeparator string
	}

	// 已经上传的对象信息
	UploadedObjectInfo struct {
		Size uint64 // 对象大小
	}
)
