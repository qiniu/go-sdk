package uploader

import (
	"bytes"
	"context"
	"io"
	"os"

	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	resumablerecorder "github.com/qiniu/go-sdk/v7/storagev2/uploader/resumable_recorder"
	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"
)

type (
	uploadManager struct {
		options *UploadManagerOptions
	}

	UploadManagerOptions struct {
		*httpclient.Options
		UpTokenProvider           uptoken.Provider
		ResumableRecorder         resumablerecorder.ResumableRecorder
		PartSize                  uint64
		MultiPartsThreshold       uint64
		Concurrency               int
		MultiPartsUploaderVersion MultiPartsUploaderVersion
	}

	MultiPartsUploaderVersion uint8
)

const (
	MultiPartsUploaderVersionV1 MultiPartsUploaderVersion = 1
	MultiPartsUploaderVersionV2 MultiPartsUploaderVersion = 2
)

func NewUploadManager(options *UploadManagerOptions) Uploader {
	if options == nil {
		options = &UploadManagerOptions{}
	}
	if options.PartSize == 0 {
		options.PartSize = 1 << 22
	} else if options.PartSize < (1 << 20) {
		options.PartSize = 1 << 20
	} else if options.PartSize > (1 << 30) {
		options.PartSize = 1 << 30
	}
	if options.MultiPartsThreshold == 0 {
		options.MultiPartsThreshold = options.PartSize
	}
	if options.MultiPartsUploaderVersion != MultiPartsUploaderVersionV1 {
		options.MultiPartsUploaderVersion = MultiPartsUploaderVersionV2
	}
	return &uploadManager{options}
}

func (uploadManager *uploadManager) UploadFile(ctx context.Context, path string, objectParams *ObjectParams, returnValue interface{}) error {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	var uploader Uploader
	if fileInfo.Size() > int64(uploadManager.options.MultiPartsThreshold) {
		uploader = NewMultiPartsUploader(uploadManager.getScheduler())
	} else {
		uploader = uploadManager.getFormUploader()
	}

	return uploader.UploadFile(ctx, path, objectParams, returnValue)
}

func (uploadManager *uploadManager) UploadReader(ctx context.Context, reader io.Reader, objectParams *ObjectParams, returnValue interface{}) error {
	var uploader Uploader

	if rscs, ok := reader.(io.ReadSeeker); ok && canSeekReally(rscs) {
		size, err := getSeekerSize(rscs)
		if err == nil && size > uploadManager.options.MultiPartsThreshold {
			uploader = NewMultiPartsUploader(uploadManager.getScheduler())
		}
	}
	if uploader == nil {
		firstPartBytes, err := internal_io.ReadAll(io.LimitReader(reader, int64(uploadManager.options.MultiPartsThreshold+1)))
		if err != nil {
			return err
		}
		reader = io.MultiReader(bytes.NewReader(firstPartBytes), reader)
		if len(firstPartBytes) > int(uploadManager.options.MultiPartsThreshold) {
			uploader = NewMultiPartsUploader(uploadManager.getScheduler())
		} else {
			uploader = uploadManager.getFormUploader()
		}
	}

	return uploader.UploadReader(ctx, reader, objectParams, returnValue)
}

func (uploadManager *uploadManager) getScheduler() MultiPartsUploaderScheduler {
	if uploadManager.options.Concurrency > 1 {
		return NewConcurrentMultiPartsUploaderScheduler(uploadManager.getMultiPartsUploader(), uploadManager.options.PartSize, uploadManager.options.Concurrency)
	} else {
		return NewSerialMultiPartsUploaderScheduler(uploadManager.getMultiPartsUploader(), uploadManager.options.PartSize)
	}
}

func (uploadManager *uploadManager) getMultiPartsUploader() MultiPartsUploader {
	multiPartsUploaderOptions := MultiPartsUploaderOptions{
		Options:         uploadManager.options.Options,
		UpTokenProvider: uploadManager.options.UpTokenProvider,
	}
	if uploadManager.options.MultiPartsUploaderVersion == MultiPartsUploaderVersionV1 {
		return NewMultiPartsUploaderV1(&multiPartsUploaderOptions)
	} else {
		return NewMultiPartsUploaderV2(&multiPartsUploaderOptions)
	}
}

func (uploadManager *uploadManager) getFormUploader() Uploader {
	return NewFormUploader(&FormUploaderOptions{
		Options:         uploadManager.options.Options,
		UpTokenProvider: uploadManager.options.UpTokenProvider,
	})
}
