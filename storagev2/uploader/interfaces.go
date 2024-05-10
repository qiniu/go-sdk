package uploader

import (
	"context"
	"io"

	"github.com/qiniu/go-sdk/v7/storagev2/region"
	"github.com/qiniu/go-sdk/v7/storagev2/uploader/source"
)

type (
	Uploader interface {
		UploadPath(ctx context.Context, path string, objectParams *ObjectParams, returnValue interface{}) error
		UploadReader(ctx context.Context, reader io.Reader, objectParams *ObjectParams, returnValue interface{}) error
	}

	MultiPartsUploader interface {
		InitializeParts(ctx context.Context, source source.Source, objectParams *ObjectParams) (InitializedParts, error)
		UploadPart(ctx context.Context, initializedParts InitializedParts) (UploadedPart, error)
		CompleteParts(ctx context.Context, initializedParts InitializedParts, parts []UploadedPart, returnValue interface{}) error
		ReinitializeParts(ctx context.Context, initializedParts InitializedParts, options *ReinitializeOptions) error
		TryToResumeParts(ctx context.Context, source source.Source, objectParams *ObjectParams) InitializedParts
	}

	InitializedParts interface {
		ObjectParams() *ObjectParams
		UpEndpoints() region.Endpoints
	}

	UploadedPart interface {
		PartSize() uint64
		Offset() uint64
		Resumed() bool
	}

	MultiPartsUploaderScheduler interface {
		Upload(ctx context.Context, source source.Source, objectParams *ObjectParams, returnValue interface{}) error
	}
)
