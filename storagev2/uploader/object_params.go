package uploader

import "github.com/qiniu/go-sdk/v7/storagev2/region"

type ObjectParams struct {
	RegionsProvider region.RegionsProvider
	ObjectName      string
	FileName        string
	ContentType     string
	Metadata        map[string]string
	CustomVars      map[string]string
}
