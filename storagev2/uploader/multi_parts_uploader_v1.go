package uploader

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
	"github.com/qiniu/go-sdk/v7/storagev2/apis"
	"github.com/qiniu/go-sdk/v7/storagev2/retrier"
	resumablerecorder "github.com/qiniu/go-sdk/v7/storagev2/uploader/resumable_recorder"
	"github.com/qiniu/go-sdk/v7/storagev2/uploader/source"
)

type (
	multiPartsUploaderV1 struct {
		storage *apis.Storage
		options *MultiPartsUploaderOptions
	}

	multiPartsUploaderV1InitializedParts struct {
		multiPartsObjectOptions *MultiPartsObjectOptions
		records                 map[uint64]resumedMultiPartsUploaderV1Record
		medium                  resumablerecorder.WriteableResumableRecorderMedium
		src                     source.Source
	}

	multiPartsUploaderV1UploadedPart struct {
		ctx                      string
		crc32                    uint32
		partNumber, offset, size uint64
	}

	resumedMultiPartsUploaderV1Record struct {
		ctx          string
		crc32        uint32
		offset, size uint64
		expiredAt    time.Time
	}
)

// 创建分片上传器 V1
func NewMultiPartsUploaderV1(options *MultiPartsUploaderOptions) MultiPartsUploader {
	if options == nil {
		options = &MultiPartsUploaderOptions{}
	}
	return &multiPartsUploaderV1{apis.NewStorage(&options.Options), options}
}

func (uploader *multiPartsUploaderV1) InitializeParts(ctx context.Context, src source.Source, multiPartsObjectOptions *MultiPartsObjectOptions) (InitializedParts, error) {
	if multiPartsObjectOptions == nil {
		multiPartsObjectOptions = &MultiPartsObjectOptions{}
	}
	if multiPartsObjectOptions.PartSize == 0 {
		multiPartsObjectOptions.PartSize = 1 << 22
	}
	medium := tryToOpenResumableRecorderForAppending(ctx, src, multiPartsObjectOptions, uploader.options)
	return &multiPartsUploaderV1InitializedParts{multiPartsObjectOptions, nil, medium, src}, nil
}

func (uploader *multiPartsUploaderV1) TryToResume(ctx context.Context, src source.Source, multiPartsObjectOptions *MultiPartsObjectOptions) InitializedParts {
	if multiPartsObjectOptions == nil {
		multiPartsObjectOptions = &MultiPartsObjectOptions{}
	}
	if multiPartsObjectOptions.PartSize == 0 {
		multiPartsObjectOptions.PartSize = 1 << 22
	}

	readableMedium := tryToOpenResumableRecorderForReading(ctx, src, multiPartsObjectOptions, uploader.options)
	if readableMedium == nil {
		return nil
	}
	defer readableMedium.Close()

	records := make(map[uint64]resumedMultiPartsUploaderV1Record)
	for {
		var record resumablerecorder.ResumableRecord
		if err := readableMedium.Next(&record); err != nil {
			break
		}
		records[record.PartNumber] = resumedMultiPartsUploaderV1Record{
			ctx:       record.PartID,
			crc32:     record.CRC32,
			offset:    record.Offset,
			size:      record.PartSize,
			expiredAt: record.ExpiredAt,
		}
	}
	readableMedium.Close()

	medium := tryToOpenResumableRecorderForAppending(ctx, src, multiPartsObjectOptions, uploader.options)
	return &multiPartsUploaderV1InitializedParts{multiPartsObjectOptions, records, medium, src}
}

func (uploader *multiPartsUploaderV1) UploadPart(ctx context.Context, initialized InitializedParts, part source.Part, options *UploadPartOptions) (UploadedPart, error) {
	initializedParts, ok := initialized.(*multiPartsUploaderV1InitializedParts)
	if !ok {
		return nil, errors.New("unrecognized initialized parts")
	}
	if len(initializedParts.records) > 0 {
		if record, ok := initializedParts.records[part.PartNumber()]; ok {
			if record.offset == part.Offset() && record.size == part.Size() {
				if options != nil && options.OnUploadingProgress != nil {
					options.OnUploadingProgress(&UploadingPartProgress{Uploaded: record.size, PartSize: record.size})
				}
				return multiPartsUploaderV1UploadedPart{
					ctx:        record.ctx,
					crc32:      record.crc32,
					offset:     record.offset,
					size:       record.size,
					partNumber: part.PartNumber(),
				}, nil
			}
		}
	}
	return uploader.uploadPart(ctx, initializedParts, part, options)
}

func (uploader *multiPartsUploaderV1) uploadPart(ctx context.Context, initialized *multiPartsUploaderV1InitializedParts, part source.Part, options *UploadPartOptions) (UploadedPart, error) {
	apisOptions := apis.Options{
		OverwrittenRegion: initialized.multiPartsObjectOptions.RegionsProvider,
	}
	if options != nil && options.OnUploadingProgress != nil {
		apisOptions.OnRequestProgress = func(uploaded, totalSize uint64) {
			options.OnUploadingProgress(&UploadingPartProgress{Uploaded: uploaded, PartSize: totalSize})
		}
	}
	upToken, err := getUpToken(uploader.options.Credentials, &initialized.multiPartsObjectOptions.ObjectOptions, uploader.options.UpTokenProvider)
	if err != nil {
		return nil, err
	}

	response, err := uploader.storage.ResumableUploadV1MakeBlock(ctx, &apis.ResumableUploadV1MakeBlockRequest{
		BlockSize: int64(part.Size()),
		UpToken:   upToken,
		Body:      internal_io.MakeReadSeekCloserFromReader(part),
	}, &apisOptions)
	if err != nil {
		return nil, err
	} else if response.Crc32 > 0 {
		if _, err = part.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
		crc32, err := crc32FromReadSeeker(part)
		if err != nil {
			return nil, err
		}
		if crc32 != uint32(response.Crc32) {
			return nil, errors.New("unexpected crc32")
		}
	}

	if medium := initialized.medium; medium != nil {
		medium.Write(&resumablerecorder.ResumableRecord{
			PartID:     response.Ctx,
			Offset:     part.Offset(),
			PartSize:   part.Size(),
			PartNumber: part.PartNumber(),
			ExpiredAt:  time.Unix(response.ExpiredAt, 0),
			CRC32:      uint32(response.Crc32),
		})
	}

	return multiPartsUploaderV1UploadedPart{
		ctx:        response.Ctx,
		crc32:      uint32(response.Crc32),
		offset:     part.Offset(),
		size:       part.Size(),
		partNumber: part.PartNumber(),
	}, nil
}

func (uploader *multiPartsUploaderV1) CompleteParts(ctx context.Context, initialized InitializedParts, parts []UploadedPart, returnValue interface{}) error {
	initializedParts, ok := initialized.(*multiPartsUploaderV1InitializedParts)
	if !ok {
		return errors.New("unrecognized initialized parts")
	}
	options := apis.Options{
		OverwrittenRegion: initializedParts.multiPartsObjectOptions.RegionsProvider,
	}
	upToken, err := getUpToken(uploader.options.Credentials, &initializedParts.multiPartsObjectOptions.ObjectOptions, uploader.options.UpTokenProvider)
	if err != nil {
		return err
	}

	var (
		ctxs = make([]string, 0, len(parts))
		size uint64
	)
	for _, part := range parts {
		uploadedPart, ok := part.(multiPartsUploaderV1UploadedPart)
		if !ok {
			return errors.New("unrecognized uploaded part")
		}
		ctxs = append(ctxs, uploadedPart.ctx)
		size += uploadedPart.size
	}

	_, err = uploader.storage.ResumableUploadV1MakeFile(ctx, &apis.ResumableUploadV1MakeFileRequest{
		Size:         int64(size),
		ObjectName:   initializedParts.multiPartsObjectOptions.ObjectName,
		FileName:     initializedParts.multiPartsObjectOptions.FileName,
		MimeType:     initializedParts.multiPartsObjectOptions.ContentType,
		CustomData:   mergeCustomVarsAndMetadata(initializedParts.multiPartsObjectOptions.Metadata, initializedParts.multiPartsObjectOptions.CustomVars),
		UpToken:      upToken,
		Body:         internal_io.NewBytesNopCloser([]byte(strings.Join(ctxs, ","))),
		ResponseBody: returnValue,
	}, &options)
	if err == nil || !retrier.IsErrorRetryable(err) {
		if medium := initializedParts.medium; medium != nil {
			medium.Close()
		}
		initializedParts.medium = nil
		tryToDeleteResumableRecorderMedium(ctx, initializedParts.src, initializedParts.multiPartsObjectOptions, uploader.options)
	}
	return err
}

func (uploader *multiPartsUploaderV1) MultiPartsUploaderOptions() *MultiPartsUploaderOptions {
	return uploader.options
}

func (initialized *multiPartsUploaderV1InitializedParts) Close() error {
	if initialized.medium != nil {
		return initialized.medium.Close()
	}
	return nil
}

func (uploadedPart multiPartsUploaderV1UploadedPart) Offset() uint64 {
	return uploadedPart.offset
}

func (uploadedPart multiPartsUploaderV1UploadedPart) PartNumber() uint64 {
	return uploadedPart.partNumber
}

func (uploadedPart multiPartsUploaderV1UploadedPart) PartSize() uint64 {
	return uploadedPart.size
}

func (uploadedPart multiPartsUploaderV1UploadedPart) Crc32() uint32 {
	return uploadedPart.crc32
}

func (uploadedPart multiPartsUploaderV1UploadedPart) Ctx() string {
	return uploadedPart.ctx
}
