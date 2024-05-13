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

func NewSerialMultiPartsUploaderScheduler(uploader MultiPartsUploader, partSize uint64) MultiPartsUploaderScheduler {
	return serialMultiPartsUploaderScheduler{uploader, partSize}
}

func NewConcurrentMultiPartsUploaderScheduler(uploader MultiPartsUploader, partSize uint64, concurrency int) MultiPartsUploaderScheduler {
	return concurrentMultiPartsUploaderScheduler{uploader, partSize, concurrency}
}

func (scheduler serialMultiPartsUploaderScheduler) UploadParts(ctx context.Context, initialized InitializedParts, src source.Source, params *UploadPartsParams) ([]UploadedPart, error) {
	parts := make([]UploadedPart, 0)
	for {
		part, err := src.Slice(scheduler.partSize)
		if err != nil {
			return nil, err
		}
		if part == nil {
			break
		}
		var uploadPartParam UploadPartParams
		if params != nil && params.OnUploadingProgress != nil {
			uploadPartParam.OnUploadingProgress = func(uploaded, partSize uint64) {
				params.OnUploadingProgress(part.PartNumber(), uploaded, scheduler.partSize)
			}
		}
		uploadedPart, err := scheduler.uploader.UploadPart(ctx, initialized, part, &uploadPartParam)
		if err != nil {
			return nil, err
		}
		parts = append(parts, uploadedPart)
	}
	return parts, nil
}

func (scheduler serialMultiPartsUploaderScheduler) MultiPartsUploader() MultiPartsUploader {
	return scheduler.uploader
}

func (scheduler concurrentMultiPartsUploaderScheduler) UploadParts(ctx context.Context, initialized InitializedParts, src source.Source, params *UploadPartsParams) ([]UploadedPart, error) {
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
	for {
		part, err := src.Slice(scheduler.partSize)
		if err != nil {
			return nil, err
		}
		if part == nil {
			break
		}
		g.Go(func() error {
			var uploadPartParam UploadPartParams
			if params != nil && params.OnUploadingProgress != nil {
				uploadPartParam.OnUploadingProgress = func(uploaded, partSize uint64) {
					params.OnUploadingProgress(part.PartNumber(), uploaded, scheduler.partSize)
				}
			}
			uploadedPart, err := scheduler.uploader.UploadPart(ctx, initialized, part, &uploadPartParam)
			if err != nil {
				return err
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
