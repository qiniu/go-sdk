package uploader

import (
	"github.com/qiniu/go-sdk/v7/storagev2/region"
	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"
)

type ObjectParams struct {
	RegionsProvider     region.RegionsProvider
	UpToken             uptoken.Provider
	BucketName          string
	ObjectName          *string
	FileName            string
	ContentType         string
	Metadata            map[string]string
	CustomVars          map[string]string
	OnUploadingProgress func(uploaded uint64, fileSize uint64)
}
