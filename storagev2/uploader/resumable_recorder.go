package uploader

import (
	"context"

	resumablerecorder "github.com/qiniu/go-sdk/v7/storagev2/uploader/resumable_recorder"
	"github.com/qiniu/go-sdk/v7/storagev2/uploader/source"
	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"
)

func tryToOpenResumableRecorderForReading(ctx context.Context, src source.Source, resumableObjectParams ResumableObjectParams, multiPartsUploaderOptions *MultiPartsUploaderOptions) resumablerecorder.ReadableResumableRecorderMedium {
	if options := makeResumableRecorderOpenOptions(ctx, src, resumableObjectParams, multiPartsUploaderOptions); options != nil {
		if resumableRecorder := multiPartsUploaderOptions.ResumableRecorder; resumableRecorder != nil {
			return resumableRecorder.OpenForReading(options)
		}
	}
	return nil
}

func tryToOpenResumableRecorderForAppending(ctx context.Context, src source.Source, resumableObjectParams ResumableObjectParams, multiPartsUploaderOptions *MultiPartsUploaderOptions) resumablerecorder.WriteableResumableRecorderMedium {
	if options := makeResumableRecorderOpenOptions(ctx, src, resumableObjectParams, multiPartsUploaderOptions); options != nil {
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

func tryToDeleteResumableRecorderMedium(ctx context.Context, src source.Source, resumableObjectParams ResumableObjectParams, multiPartsUploaderOptions *MultiPartsUploaderOptions) {
	if options := makeResumableRecorderOpenOptions(ctx, src, resumableObjectParams, multiPartsUploaderOptions); options != nil {
		if resumableRecorder := multiPartsUploaderOptions.ResumableRecorder; resumableRecorder != nil {
			resumableRecorder.Delete(options)
		}
	}
}

func makeResumableRecorderOpenOptions(ctx context.Context, src source.Source, resumableObjectParams ResumableObjectParams, multiPartsUploaderOptions *MultiPartsUploaderOptions) *resumablerecorder.ResumableRecorderOpenOptions {
	guessBucketName := func(ctx context.Context, bucketName string, upTokenProvider uptoken.Provider) (string, error) {
		if bucketName == "" {
			if putPolicy, err := upTokenProvider.GetPutPolicy(ctx); err != nil {
				return "", err
			} else if bucketName, err = putPolicy.GetBucketName(); err != nil {
				return "", err
			}
		}
		return bucketName, nil
	}

	sourceKey, err := src.SourceKey()
	if err != nil || sourceKey == "" {
		return nil
	}

	upToken, err := getUpToken(multiPartsUploaderOptions.Credentials, resumableObjectParams.ObjectParams, multiPartsUploaderOptions.UpTokenProvider)
	if err != nil {
		return nil
	}

	bucketName, err := guessBucketName(ctx, resumableObjectParams.BucketName, upToken)
	if err != nil {
		return nil
	}

	var objectName string
	if resumableObjectParams.ObjectName != nil {
		objectName = *resumableObjectParams.ObjectName
	}

	var totalSize uint64
	if sizedSource, ok := src.(source.SizedSource); ok {
		if ts, err := sizedSource.TotalSize(); err == nil {
			totalSize = ts
		}
	}

	regions, err := getRegions(ctx, upToken, bucketName, multiPartsUploaderOptions.Options)
	if err != nil || len(regions) == 0 {
		return nil
	}

	return &resumablerecorder.ResumableRecorderOpenOptions{
		BucketName:  bucketName,
		ObjectName:  objectName,
		SourceKey:   sourceKey,
		PartSize:    resumableObjectParams.PartSize,
		TotalSize:   totalSize,
		UpEndpoints: regions[0].Up,
	}
}
