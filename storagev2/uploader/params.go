package uploader

import (
	"github.com/qiniu/go-sdk/v7/storagev2/region"
	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"
)

type (
	ObjectParams struct {
		RegionsProvider     region.RegionsProvider
		UpToken             uptoken.Provider
		BucketName          string
		ObjectName          *string
		FileName            string
		ContentType         string
		Metadata            map[string]string
		CustomVars          map[string]string
		OnUploadingProgress func(uploaded, totalSize uint64)
	}

	ResumableObjectParams struct {
		*ObjectParams
		PartSize uint64
	}

	UploadPartsParams struct {
		OnUploadingProgress func(partNumber, uploaded, partSize uint64)
		OnPartUploaded      func(partNumber, partSize uint64)
	}

	UploadPartParams struct {
		OnUploadingProgress func(uploaded, partSize uint64)
	}

	DirectoryParams struct {
		RegionsProvider       region.RegionsProvider
		UpToken               uptoken.Provider
		BucketName            string
		ObjectPrefix          string
		FileConcurrency       int
		BeforeFileUpload      func(filePath string, objectParams *ObjectParams)
		OnUploadingProgress   func(filePath string, uploaded, totalSize uint64)
		OnFileUploaded        func(filePath string, size uint64)
		ShouldCreateDirectory bool
		ShouldUploadFile      func(filePath string) bool
	}
)
