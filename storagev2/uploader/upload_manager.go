package uploader

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	resumablerecorder "github.com/qiniu/go-sdk/v7/storagev2/uploader/resumable_recorder"
	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"
	"golang.org/x/sync/errgroup"
)

type (
	// 上传器
	UploadManager struct {
		options                   httpclient.Options
		upTokenProvider           uptoken.Provider
		resumableRecorder         resumablerecorder.ResumableRecorder
		partSize                  uint64
		multiPartsThreshold       uint64
		concurrency               int
		multiPartsUploaderVersion MultiPartsUploaderVersion
	}

	// 上传器选项
	UploadManagerOptions struct {
		// HTTP 客户端选项
		httpclient.Options

		// 上传凭证接口
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
	if options == nil {
		options = &UploadManagerOptions{}
	}
	partSize := options.PartSize
	if partSize == 0 {
		partSize = 1 << 22
	} else if partSize < (1 << 20) {
		partSize = 1 << 20
	} else if partSize > (1 << 30) {
		partSize = 1 << 30
	}
	multiPartsThreshold := options.MultiPartsThreshold
	if multiPartsThreshold == 0 {
		multiPartsThreshold = partSize
	}
	concurrency := options.Concurrency
	if concurrency == 0 {
		concurrency = 4
	}
	uploadManager := UploadManager{
		options:                   options.Options,
		upTokenProvider:           options.UpTokenProvider,
		resumableRecorder:         options.ResumableRecorder,
		partSize:                  partSize,
		multiPartsThreshold:       multiPartsThreshold,
		concurrency:               concurrency,
		multiPartsUploaderVersion: options.MultiPartsUploaderVersion,
	}
	return &uploadManager
}

// 上传目录
func (uploadManager *UploadManager) UploadDirectory(ctx context.Context, directoryPath string, directoryOptions *DirectoryOptions) error {
	if directoryOptions == nil {
		directoryOptions = &DirectoryOptions{}
	}
	objectConcurrency := directoryOptions.ObjectConcurrency
	if objectConcurrency == 0 {
		objectConcurrency = 4
	}
	pathSeparator := directoryOptions.PathSeparator
	if pathSeparator == "" {
		pathSeparator = "/"
	}

	if !strings.HasSuffix(directoryPath, string(filepath.Separator)) {
		directoryPath += string(filepath.Separator)
	}

	updateObjectName := directoryOptions.UpdateObjectName
	if updateObjectName == nil {
		updateObjectName = func(path string) string { return path }
	}
	generateObjectName := func(path string) string {
		path = strings.TrimPrefix(path, directoryPath)
		if pathSeparator != string(filepath.Separator) {
			path = strings.Replace(path, string(filepath.Separator), pathSeparator, -1)
		}
		return updateObjectName(path)
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(objectConcurrency)

	err := filepath.Walk(directoryPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		g.Go(func() error {
			objectName := generateObjectName(path)
			if info.Mode().IsRegular() {
				objectOptions := ObjectOptions{
					RegionsProvider: directoryOptions.RegionsProvider,
					UpToken:         directoryOptions.UpToken,
					BucketName:      directoryOptions.BucketName,
					ObjectName:      &objectName,
					FileName:        filepath.Base(path),
				}
				if directoryOptions.ShouldUploadObject != nil && !directoryOptions.ShouldUploadObject(path, &objectOptions) {
					return nil
				}
				if directoryOptions.BeforeObjectUpload != nil {
					directoryOptions.BeforeObjectUpload(path, &objectOptions)
				}
				if directoryOptions.OnUploadingProgress != nil {
					objectOptions.OnUploadingProgress = func(progress *UploadingProgress) {
						directoryOptions.OnUploadingProgress(path, progress)
					}
				}
				err = uploadManager.UploadFile(ctx, path, &objectOptions, nil)
				if err == nil && directoryOptions.OnObjectUploaded != nil {
					directoryOptions.OnObjectUploaded(path, &UploadedObjectInfo{Size: uint64(info.Size())})
				}
			} else if directoryOptions.ShouldCreateDirectory && info.IsDir() {
				if directoryOptions.ShouldUploadObject != nil && !directoryOptions.ShouldUploadObject(path, nil) {
					return nil
				}
				if !strings.HasSuffix(objectName, pathSeparator) {
					objectName += pathSeparator
				}
				objectOptions := ObjectOptions{
					RegionsProvider: directoryOptions.RegionsProvider,
					UpToken:         directoryOptions.UpToken,
					BucketName:      directoryOptions.BucketName,
					ObjectName:      &objectName,
					FileName:        filepath.Base(path),
				}
				err = uploadManager.UploadReader(ctx, http.NoBody, &objectOptions, nil)
			}
			return err
		})
		return nil
	})
	if err != nil {
		return err
	}

	return g.Wait()

}

// 上传文件
func (uploadManager *UploadManager) UploadFile(ctx context.Context, path string, objectOptions *ObjectOptions, returnValue interface{}) error {
	if objectOptions == nil {
		objectOptions = &ObjectOptions{}
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	var uploader Uploader
	if fileInfo.Size() > int64(uploadManager.multiPartsThreshold) {
		uploader = newMultiPartsUploader(uploadManager.getScheduler())
	} else {
		uploader = uploadManager.getFormUploader()
	}

	return uploader.UploadFile(ctx, path, objectOptions, returnValue)
}

// 上传 io.Reader
func (uploadManager *UploadManager) UploadReader(ctx context.Context, reader io.Reader, objectOptions *ObjectOptions, returnValue interface{}) error {
	var uploader Uploader

	if objectOptions == nil {
		objectOptions = &ObjectOptions{}
	}

	if rscs, ok := reader.(io.ReadSeeker); ok && canSeekReally(rscs) {
		size, err := getSeekerSize(rscs)
		if err == nil && size > uploadManager.multiPartsThreshold {
			uploader = newMultiPartsUploader(uploadManager.getScheduler())
		}
	}
	if uploader == nil {
		firstPartBytes, err := internal_io.ReadAll(io.LimitReader(reader, int64(uploadManager.multiPartsThreshold+1)))
		if err != nil {
			return err
		}
		reader = io.MultiReader(bytes.NewReader(firstPartBytes), reader)
		if len(firstPartBytes) > int(uploadManager.multiPartsThreshold) {
			uploader = newMultiPartsUploader(uploadManager.getScheduler())
		} else {
			uploader = uploadManager.getFormUploader()
		}
	}

	return uploader.UploadReader(ctx, reader, objectOptions, returnValue)
}

func (uploadManager *UploadManager) getScheduler() multiPartsUploaderScheduler {
	if uploadManager.concurrency > 1 {
		return newConcurrentMultiPartsUploaderScheduler(uploadManager.getMultiPartsUploader(), &concurrentMultiPartsUploaderSchedulerOptions{
			PartSize: uploadManager.partSize, Concurrency: uploadManager.concurrency,
		})
	} else {
		return newSerialMultiPartsUploaderScheduler(uploadManager.getMultiPartsUploader(), &serialMultiPartsUploaderSchedulerOptions{
			PartSize: uploadManager.partSize,
		})
	}
}

func (uploadManager *UploadManager) getMultiPartsUploader() MultiPartsUploader {
	multiPartsUploaderOptions := MultiPartsUploaderOptions{
		Options:           uploadManager.options,
		UpTokenProvider:   uploadManager.upTokenProvider,
		ResumableRecorder: uploadManager.resumableRecorder,
	}
	if uploadManager.multiPartsUploaderVersion == MultiPartsUploaderVersionV1 {
		return NewMultiPartsUploaderV1(&multiPartsUploaderOptions)
	} else {
		return NewMultiPartsUploaderV2(&multiPartsUploaderOptions)
	}
}

func (uploadManager *UploadManager) getFormUploader() Uploader {
	return NewFormUploader(&FormUploaderOptions{
		Options: uploadManager.options,
		UpToken: uploadManager.upTokenProvider,
	})
}
