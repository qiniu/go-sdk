package uploader

import (
	"context"
	"io"

	"github.com/qiniu/go-sdk/v7/storagev2/uploader/source"
)

type (
	Uploader interface {
		UploadPath(context.Context, string, *ObjectParams, interface{}) error
		UploadReader(context.Context, io.Reader, *ObjectParams, interface{}) error
	}

	MultiPartsUploader interface {
		InitializeParts(context.Context, source.Source, ResumableObjectParams) (InitializedParts, error)
		TryToResume(context.Context, source.Source, ResumableObjectParams) InitializedParts
		UploadPart(context.Context, InitializedParts, source.Part, *UploadPartParams) (UploadedPart, error)
		CompleteParts(context.Context, InitializedParts, []UploadedPart, interface{}) error
		MultiPartsUploaderOptions() *MultiPartsUploaderOptions
	}

	InitializedParts interface {
		io.Closer
	}

	UploadedPart interface {
		Offset() uint64
	}

	MultiPartsUploaderScheduler interface {
		UploadParts(context.Context, InitializedParts, source.Source, *UploadPartsParams) ([]UploadedPart, error)
		MultiPartsUploader() MultiPartsUploader
		PartSize() uint64
	}
)
