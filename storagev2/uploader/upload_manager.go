package uploader

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	resumablerecorder "github.com/qiniu/go-sdk/v7/storagev2/uploader/resumable_recorder"
	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"
	"golang.org/x/sync/errgroup"
)

type (
	// 上传器
	UploadManager struct {
		options     *UploadManagerOptions
		optionsInit sync.Once
	}

	// 上传器选项
	UploadManagerOptions struct {
		// HTTP 客户端选项
		*httpclient.Options

		// HTTP 客户端选项
		UpTokenProvider uptoken.Provider

		// 可恢复记录，如果不设置，则无法进行断点续传
		ResumableRecorder resumablerecorder.ResumableRecorder

		// 分片大小，如果不填写，默认为 4 MB
		PartSize uint64

		// 分片上传阈值，如果不填写，默认为 4 MB
		MultiPartsThreshold uint64

		// 分片上传并行度，如果不填写，默认为 1
		Concurrency int

		// 分片上传版本，如果不填写，默认为 V2
		MultiPartsUploaderVersion MultiPartsUploaderVersion
	}

	// 分片上传版本
	MultiPartsUploaderVersion uint8
)

const (
	// 分片上传 V1
	MultiPartsUploaderVersionV1 MultiPartsUploaderVersion = 1

	// 分片上传 V2
	MultiPartsUploaderVersionV2 MultiPartsUploaderVersion = 2
)

// 创建上传器
func NewUploadManager(options *UploadManagerOptions) *UploadManager {
	uploadManager := UploadManager{options: options}
	uploadManager.init()
	return &uploadManager
}

// 上传目录
func (uploadManager *UploadManager) UploadDirectory(ctx context.Context, directoryPath string, directoryParams *DirectoryParams) error {
	uploadManager.init()

	if directoryParams == nil {
		directoryParams = &DirectoryParams{}
	}
	if directoryParams.FileConcurrency == 0 {
		directoryParams.FileConcurrency = 1
	}

	if !strings.HasSuffix(directoryPath, "/") {
		directoryPath += "/"
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(directoryParams.FileConcurrency)

	err := filepath.Walk(directoryPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		objectName := filepath.Join(directoryParams.ObjectPrefix, strings.TrimPrefix(path, directoryPath))
		if info.Mode().IsRegular() {
			objectParams := ObjectParams{
				RegionsProvider: directoryParams.RegionsProvider,
				UpToken:         directoryParams.UpToken,
				BucketName:      directoryParams.BucketName,
				ObjectName:      &objectName,
				FileName:        filepath.Base(path),
			}
			if directoryParams.ShouldUploadFile != nil && !directoryParams.ShouldUploadFile(path) {
				return nil
			}
			if directoryParams.BeforeFileUpload != nil {
				directoryParams.BeforeFileUpload(path, &objectParams)
			}
			if directoryParams.OnUploadingProgress != nil {
				objectParams.OnUploadingProgress = func(uploaded, totalSize uint64) {
					directoryParams.OnUploadingProgress(path, uploaded, totalSize)
				}
			}
			err = uploadManager.UploadFile(ctx, path, &objectParams, nil)
			if err == nil && directoryParams.OnFileUploaded != nil {
				directoryParams.OnFileUploaded(path, uint64(info.Size()))
			}
		} else if directoryParams.ShouldCreateDirectory && info.IsDir() {
			objectName += string(os.PathSeparator)
			objectParams := ObjectParams{
				RegionsProvider: directoryParams.RegionsProvider,
				UpToken:         directoryParams.UpToken,
				BucketName:      directoryParams.BucketName,
				ObjectName:      &objectName,
				FileName:        filepath.Base(path),
			}
			err = uploadManager.UploadReader(ctx, http.NoBody, &objectParams, nil)
		}
		return err
	})
	if err != nil {
		return err
	}

	return g.Wait()

}

// 上传文件
func (uploadManager *UploadManager) UploadFile(ctx context.Context, path string, objectParams *ObjectParams, returnValue interface{}) error {
	uploadManager.init()

	if objectParams == nil {
		objectParams = &ObjectParams{}
	}

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

// 上传 io.Reader
func (uploadManager *UploadManager) UploadReader(ctx context.Context, reader io.Reader, objectParams *ObjectParams, returnValue interface{}) error {
	var uploader Uploader

	uploadManager.init()

	if objectParams == nil {
		objectParams = &ObjectParams{}
	}

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

func (uploadManager *UploadManager) getScheduler() MultiPartsUploaderScheduler {
	if uploadManager.options.Concurrency > 1 {
		return NewConcurrentMultiPartsUploaderScheduler(uploadManager.getMultiPartsUploader(), uploadManager.options.PartSize, uploadManager.options.Concurrency)
	} else {
		return NewSerialMultiPartsUploaderScheduler(uploadManager.getMultiPartsUploader(), uploadManager.options.PartSize)
	}
}

func (uploadManager *UploadManager) getMultiPartsUploader() MultiPartsUploader {
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

func (uploadManager *UploadManager) getFormUploader() Uploader {
	return NewFormUploader(&FormUploaderOptions{
		Options:         uploadManager.options.Options,
		UpTokenProvider: uploadManager.options.UpTokenProvider,
	})
}

func (uploadManager *UploadManager) init() {
	uploadManager.optionsInit.Do(func() {
		if uploadManager.options == nil {
			uploadManager.options = &UploadManagerOptions{}
		}
		if uploadManager.options.PartSize == 0 {
			uploadManager.options.PartSize = 1 << 22
		} else if uploadManager.options.PartSize < (1 << 20) {
			uploadManager.options.PartSize = 1 << 20
		} else if uploadManager.options.PartSize > (1 << 30) {
			uploadManager.options.PartSize = 1 << 30
		}
		if uploadManager.options.MultiPartsThreshold == 0 {
			uploadManager.options.MultiPartsThreshold = uploadManager.options.PartSize
		}
	})
}
