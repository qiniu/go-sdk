package uploader

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
	"github.com/qiniu/go-sdk/v7/storagev2/apis"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/retrier"
	resumablerecorder "github.com/qiniu/go-sdk/v7/storagev2/uploader/resumable_recorder"
	"github.com/qiniu/go-sdk/v7/storagev2/uploader/source"
	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"
)

type (
	multiPartsUploaderV1 struct {
		storage *apis.Storage
		options *MultiPartsUploaderOptions
	}

	multiPartsUploaderV1InitializedParts struct {
		resumableObjectParams ResumableObjectParams
		records               map[uint64]resumedMultiPartsUploaderV1Record
		medium                resumablerecorder.WriteableResumableRecorderMedium
		src                   source.Source
	}

	multiPartsUploaderV1UploadedPart struct {
		ctx          string
		crc32        uint32
		offset, size uint64
	}

	resumedMultiPartsUploaderV1Record struct {
		ctx          string
		crc32        uint32
		offset, size uint64
		expiredAt    time.Time
	}

	MultiPartsUploaderOptions struct {
		*httpclient.Options
		UpTokenProvider   uptoken.Provider
		ResumableRecorder resumablerecorder.ResumableRecorder
	}
)

func NewMultiPartsUploaderV1(options *MultiPartsUploaderOptions) MultiPartsUploader {
	if options == nil {
		options = &MultiPartsUploaderOptions{}
	}
	return &multiPartsUploaderV1{apis.NewStorage(options.Options), options}
}

func (uploader *multiPartsUploaderV1) InitializeParts(ctx context.Context, src source.Source, resumableObjectParams ResumableObjectParams) (InitializedParts, error) {
	if resumableObjectParams.ObjectParams == nil {
		resumableObjectParams.ObjectParams = &ObjectParams{}
	}
	medium := tryToOpenResumableRecorderForAppending(ctx, src, resumableObjectParams, uploader.options)
	return &multiPartsUploaderV1InitializedParts{resumableObjectParams, nil, medium, src}, nil
}

func (uploader *multiPartsUploaderV1) TryToResume(ctx context.Context, src source.Source, resumableObjectParams ResumableObjectParams) InitializedParts {
	if resumableObjectParams.ObjectParams == nil {
		resumableObjectParams.ObjectParams = &ObjectParams{}
	}

	readableMedium := tryToOpenResumableRecorderForReading(ctx, src, resumableObjectParams, uploader.options)
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
			ctx:       record.PartId,
			crc32:     record.Crc32,
			offset:    record.Offset,
			size:      record.PartSize,
			expiredAt: record.ExpiredAt,
		}
	}
	readableMedium.Close()

	medium := tryToOpenResumableRecorderForAppending(ctx, src, resumableObjectParams, uploader.options)
	return &multiPartsUploaderV1InitializedParts{resumableObjectParams, records, medium, src}
}

func (uploader *multiPartsUploaderV1) UploadPart(ctx context.Context, initialized InitializedParts, part source.Part, params *UploadPartParams) (UploadedPart, error) {
	initializedParts, ok := initialized.(*multiPartsUploaderV1InitializedParts)
	if !ok {
		return nil, errors.New("unrecognized initialized parts")
	}
	if len(initializedParts.records) > 0 {
		if record, ok := initializedParts.records[part.PartNumber()]; ok {
			if record.offset == part.Offset() && record.size == part.Size() {
				if params != nil && params.OnUploadingProgress != nil {
					params.OnUploadingProgress(record.size, record.size)
				}
				return multiPartsUploaderV1UploadedPart{
					ctx:    record.ctx,
					crc32:  record.crc32,
					offset: record.offset,
					size:   record.size,
				}, nil
			}
		}
	}
	return uploader.uploadPart(ctx, initializedParts, part, params)
}

func (uploader *multiPartsUploaderV1) uploadPart(ctx context.Context, initialized *multiPartsUploaderV1InitializedParts, part source.Part, params *UploadPartParams) (UploadedPart, error) {
	options := apis.Options{
		OverwrittenRegion: initialized.resumableObjectParams.RegionsProvider,
	}
	if params != nil && params.OnUploadingProgress != nil {
		options.OnRequestProgress = params.OnUploadingProgress
	}
	upToken, err := getUpToken(uploader.options.Credentials, initialized.resumableObjectParams.ObjectParams, uploader.options.UpTokenProvider)
	if err != nil {
		return nil, err
	}

	response, err := uploader.storage.ResumableUploadV1MakeBlock(ctx, &apis.ResumableUploadV1MakeBlockRequest{
		BlockSize: int64(part.Size()),
		UpToken:   upToken,
		Body:      internal_io.MakeReadSeekCloserFromReader(part),
	}, &options)
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
			PartId:     response.Ctx,
			Offset:     part.Offset(),
			PartSize:   part.Size(),
			PartNumber: part.PartNumber(),
			ExpiredAt:  time.Unix(response.ExpiredAt, 0),
			Crc32:      uint32(response.Crc32),
		})
	}

	return multiPartsUploaderV1UploadedPart{
		ctx:    response.Ctx,
		crc32:  uint32(response.Crc32),
		offset: part.Offset(),
		size:   part.Size(),
	}, nil
}

func (uploader *multiPartsUploaderV1) CompleteParts(ctx context.Context, initialized InitializedParts, parts []UploadedPart, returnValue interface{}) error {
	initializedParts, ok := initialized.(*multiPartsUploaderV1InitializedParts)
	if !ok {
		return errors.New("unrecognized initialized parts")
	}
	options := apis.Options{
		OverwrittenRegion: initializedParts.resumableObjectParams.RegionsProvider,
	}
	upToken, err := getUpToken(uploader.options.Credentials, initializedParts.resumableObjectParams.ObjectParams, uploader.options.UpTokenProvider)
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
		ObjectName:   initializedParts.resumableObjectParams.ObjectName,
		FileName:     initializedParts.resumableObjectParams.FileName,
		MimeType:     initializedParts.resumableObjectParams.ContentType,
		CustomData:   mergeCustomVarsAndMetadata(initializedParts.resumableObjectParams.Metadata, initializedParts.resumableObjectParams.CustomVars),
		UpToken:      upToken,
		Body:         internal_io.NewBytesNopCloser([]byte(strings.Join(ctxs, ","))),
		ResponseBody: returnValue,
	}, &options)
	if err == nil || !retrier.IsErrorRetryable(err) {
		if medium := initializedParts.medium; medium != nil {
			medium.Close()
		}
		initializedParts.medium = nil
		tryToDeleteResumableRecorderMedium(ctx, initializedParts.src, initializedParts.resumableObjectParams, uploader.options)
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
