package uploader

import (
	"context"
	"sort"
	"sync"

	"github.com/qiniu/go-sdk/v7/storagev2/uploader/source"
	"golang.org/x/sync/errgroup"
)

type (
	serialMultiPartsUploaderScheduler struct {
		uploader MultiPartsUploader
		partSize uint64
	}

	concurrentMultiPartsUploaderScheduler struct {
		uploader    MultiPartsUploader
		partSize    uint64
		concurrency int
	}
)

// 创建串行分片上传调度器
func NewSerialMultiPartsUploaderScheduler(uploader MultiPartsUploader, partSize uint64) MultiPartsUploaderScheduler {
	return serialMultiPartsUploaderScheduler{uploader, partSize}
}

// 创建并行分片上传调度器
func NewConcurrentMultiPartsUploaderScheduler(uploader MultiPartsUploader, partSize uint64, concurrency int) MultiPartsUploaderScheduler {
	return concurrentMultiPartsUploaderScheduler{uploader, partSize, concurrency}
}

func (scheduler serialMultiPartsUploaderScheduler) UploadParts(ctx context.Context, initialized InitializedParts, src source.Source, options *UploadPartsOptions) ([]UploadedPart, error) {
	parts := make([]UploadedPart, 0)
	for {
		part, err := src.Slice(scheduler.partSize)
		if err != nil {
			return nil, err
		}
		if part == nil {
			break
		}
		var uploadPartParam UploadPartOptions
		if options != nil && options.OnUploadingProgress != nil {
			uploadPartParam.OnUploadingProgress = func(uploaded, partSize uint64) {
				options.OnUploadingProgress(part.PartNumber(), uploaded, part.Size())
			}
		}
		uploadedPart, err := scheduler.uploader.UploadPart(ctx, initialized, part, &uploadPartParam)
		if err != nil {
			return nil, err
		}
		if options != nil && options.OnPartUploaded != nil {
			options.OnPartUploaded(part.PartNumber(), part.Size())
		}
		parts = append(parts, uploadedPart)
	}
	return parts, nil
}

func (scheduler serialMultiPartsUploaderScheduler) MultiPartsUploader() MultiPartsUploader {
	return scheduler.uploader
}

func (scheduler serialMultiPartsUploaderScheduler) PartSize() uint64 {
	return scheduler.partSize
}

func (scheduler concurrentMultiPartsUploaderScheduler) UploadParts(ctx context.Context, initialized InitializedParts, src source.Source, options *UploadPartsOptions) ([]UploadedPart, error) {
	var (
		parts     []UploadedPart
		partsLock sync.Mutex
	)
	if ss, ok := src.(source.SizedSource); ok {
		totalSize, err := ss.TotalSize()
		if err != nil {
			return nil, err
		}
		partsCount := (totalSize + scheduler.partSize - 1) / scheduler.partSize
		parts = make([]UploadedPart, 0, partsCount)
	}
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(scheduler.concurrency)
	var onUploadingProgressMutex sync.Mutex
	for {
		part, err := src.Slice(scheduler.partSize)
		if err != nil {
			return nil, err
		}
		if part == nil {
			break
		}
		g.Go(func() error {
			var uploadPartParam UploadPartOptions
			if options != nil && options.OnUploadingProgress != nil {
				uploadPartParam.OnUploadingProgress = func(uploaded, partSize uint64) {
					onUploadingProgressMutex.Lock()
					defer onUploadingProgressMutex.Unlock()
					options.OnUploadingProgress(part.PartNumber(), uploaded, partSize)
				}
			}
			uploadedPart, err := scheduler.uploader.UploadPart(ctx, initialized, part, &uploadPartParam)
			if err != nil {
				return err
			}
			if options != nil && options.OnPartUploaded != nil {
				options.OnPartUploaded(part.PartNumber(), part.Size())
			}

			partsLock.Lock()
			defer partsLock.Unlock()
			parts = append(parts, uploadedPart)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	sort.Slice(parts, func(i, j int) bool {
		return parts[i].Offset() < parts[j].Offset()
	})
	return parts, nil
}

func (scheduler concurrentMultiPartsUploaderScheduler) MultiPartsUploader() MultiPartsUploader {
	return scheduler.uploader
}

func (scheduler concurrentMultiPartsUploaderScheduler) PartSize() uint64 {
	return scheduler.partSize
}
