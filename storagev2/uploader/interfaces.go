package uploader

import (
	"context"
	"io"

	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/uploader/source"
)

type (
	Uploader interface {
		UploadPath(context.Context, string, *ObjectParams, interface{}) error
		UploadReader(context.Context, io.Reader, *ObjectParams, interface{}) error
	}

	MultiPartsUploader interface {
		InitializeParts(context.Context, source.Source, *ObjectParams) (InitializedParts, error)
		TryToResume(context.Context, source.Source, *ObjectParams) (ResumedInitializedParts, error)
		UploadPart(context.Context, InitializedParts, source.Part, *UploadPartParams) (UploadedPart, error)
		CompleteParts(context.Context, InitializedParts, []UploadedPart, interface{}) error
		HttpClientOptions() *httpclient.Options
	}

	InitializedParts interface {
	}

	ResumedInitializedParts interface {
		InitializedParts
		ResumedBytes() uint64
	}

	UploadedPart interface {
		Offset() uint64
	}

	MultiPartsUploaderScheduler interface {
		UploadParts(context.Context, InitializedParts, source.Source, *UploadPartsParams) ([]UploadedPart, error)
		MultiPartsUploader() MultiPartsUploader
	}
)
