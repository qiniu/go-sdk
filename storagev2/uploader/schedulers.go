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

	// 串行分片上传调度器选项
	serialMultiPartsUploaderSchedulerOptions struct {
		PartSize uint64 // 分片大小
	}

	concurrentMultiPartsUploaderScheduler struct {
		uploader    MultiPartsUploader
		partSize    uint64
		concurrency int
	}

	// 并行分片上传调度器选项
	concurrentMultiPartsUploaderSchedulerOptions struct {
		PartSize    uint64 // 分片大小
		Concurrency int    // 并发度
	}
)

// 创建串行分片上传调度器
func newSerialMultiPartsUploaderScheduler(uploader MultiPartsUploader, options *serialMultiPartsUploaderSchedulerOptions) multiPartsUploaderScheduler {
	if options == nil {
		options = &serialMultiPartsUploaderSchedulerOptions{}
	}
	partSize := options.PartSize
	if partSize == 0 {
		partSize = 1 << 22
	} else if partSize < (1 << 20) {
		partSize = 1 << 20
	} else if partSize > (1 << 30) {
		partSize = 1 << 30
	}
	return serialMultiPartsUploaderScheduler{uploader, partSize}
}

// 创建并行分片上传调度器
func newConcurrentMultiPartsUploaderScheduler(uploader MultiPartsUploader, options *concurrentMultiPartsUploaderSchedulerOptions) multiPartsUploaderScheduler {
	if options == nil {
		options = &concurrentMultiPartsUploaderSchedulerOptions{}
	}
	partSize := options.PartSize
	if partSize == 0 {
		partSize = 1 << 22
	} else if partSize < (1 << 20) {
		partSize = 1 << 20
	} else if partSize > (1 << 30) {
		partSize = 1 << 30
	}
	concurrency := options.Concurrency
	if concurrency <= 0 {
		concurrency = 4
	}

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
			uploadPartParam.OnUploadingProgress = func(progress *UploadingPartProgress) {
				options.OnUploadingProgress(part.PartNumber(), &UploadingPartProgress{Uploaded: progress.Uploaded, PartSize: part.Size()})
			}
		}
		uploadedPart, err := scheduler.uploader.UploadPart(ctx, initialized, part, &uploadPartParam)
		if err != nil {
			return nil, err
		}
		if options != nil && options.OnPartUploaded != nil {
			if err = options.OnPartUploaded(uploadedPart); err != nil {
				return nil, err
			}
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
				uploadPartParam.OnUploadingProgress = func(progress *UploadingPartProgress) {
					onUploadingProgressMutex.Lock()
					defer onUploadingProgressMutex.Unlock()
					options.OnUploadingProgress(part.PartNumber(), progress)
				}
			}
			uploadedPart, err := scheduler.uploader.UploadPart(ctx, initialized, part, &uploadPartParam)
			if err != nil {
				return err
			}
			if options != nil && options.OnPartUploaded != nil {
				if err = options.OnPartUploaded(uploadedPart); err != nil {
					return err
				}
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
