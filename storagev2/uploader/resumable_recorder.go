package uploader

import (
	"context"

	resumablerecorder "github.com/qiniu/go-sdk/v7/storagev2/uploader/resumable_recorder"
	"github.com/qiniu/go-sdk/v7/storagev2/uploader/source"
	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"
)

func tryToOpenResumableRecorderForReading(ctx context.Context, src source.Source, multiPartsObjectOptions *MultiPartsObjectOptions, multiPartsUploaderOptions *MultiPartsUploaderOptions) resumablerecorder.ReadableResumableRecorderMedium {
	if options := makeResumableRecorderOpenArgs(ctx, src, multiPartsObjectOptions, multiPartsUploaderOptions); options != nil {
		if resumableRecorder := multiPartsUploaderOptions.ResumableRecorder; resumableRecorder != nil {
			return resumableRecorder.OpenForReading(options)
		}
	}
	return nil
}

func tryToOpenResumableRecorderForAppending(ctx context.Context, src source.Source, multiPartsObjectOptions *MultiPartsObjectOptions, multiPartsUploaderOptions *MultiPartsUploaderOptions) resumablerecorder.WriteableResumableRecorderMedium {
	if options := makeResumableRecorderOpenArgs(ctx, src, multiPartsObjectOptions, multiPartsUploaderOptions); options != nil {
		if resumableRecorder := multiPartsUploaderOptions.ResumableRecorder; resumableRecorder != nil {
			medium := resumableRecorder.OpenForAppending(options)
			if medium == nil {
				medium = resumableRecorder.OpenForCreatingNew(options)
			}
			return medium
		}
	}
	return nil
}

func tryToDeleteResumableRecorderMedium(ctx context.Context, src source.Source, multiPartsObjectOptions *MultiPartsObjectOptions, multiPartsUploaderOptions *MultiPartsUploaderOptions) {
	if options := makeResumableRecorderOpenArgs(ctx, src, multiPartsObjectOptions, multiPartsUploaderOptions); options != nil {
		if resumableRecorder := multiPartsUploaderOptions.ResumableRecorder; resumableRecorder != nil {
			resumableRecorder.Delete(options)
		}
	}
}

func makeResumableRecorderOpenArgs(ctx context.Context, src source.Source, multiPartsObjectOptions *MultiPartsObjectOptions, multiPartsUploaderOptions *MultiPartsUploaderOptions) *resumablerecorder.ResumableRecorderOpenArgs {
	sourceID, err := src.SourceID()
	if err != nil || sourceID == "" {
		return nil
	}

	upToken, err := getUpToken(multiPartsUploaderOptions.Credentials, &multiPartsObjectOptions.ObjectOptions, multiPartsUploaderOptions.UpTokenProvider)
	if err != nil {
		return nil
	}
	accessKey, err := upToken.GetAccessKey(ctx)
	if err != nil {
		return nil
	}

	bucketName, err := guessBucketName(ctx, multiPartsObjectOptions.BucketName, upToken)
	if err != nil {
		return nil
	}

	var objectName string
	if multiPartsObjectOptions.ObjectName != nil {
		objectName = *multiPartsObjectOptions.ObjectName
	}

	var totalSize uint64
	if sizedSource, ok := src.(source.SizedSource); ok {
		if ts, err := sizedSource.TotalSize(); err == nil {
			totalSize = ts
		}
	}

	regions, err := getRegions(ctx, upToken, bucketName, &multiPartsUploaderOptions.Options)
	if err != nil || len(regions) == 0 {
		return nil
	}

	return &resumablerecorder.ResumableRecorderOpenArgs{
		AccessKey:   accessKey,
		BucketName:  bucketName,
		ObjectName:  objectName,
		SourceID:    sourceID,
		PartSize:    multiPartsObjectOptions.PartSize,
		TotalSize:   totalSize,
		UpEndpoints: regions[0].Up,
	}
}

func guessBucketName(ctx context.Context, bucketName string, upTokenProvider uptoken.Provider) (string, error) {
	if bucketName == "" {
		if putPolicy, err := upTokenProvider.GetPutPolicy(ctx); err != nil {
			return "", err
		} else if bucketName, err = putPolicy.GetBucketName(); err != nil {
			return "", err
		}
	}
	return bucketName, nil
}
