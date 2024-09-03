package uploader

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"time"

	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
	"github.com/qiniu/go-sdk/v7/storagev2/apis"
	"github.com/qiniu/go-sdk/v7/storagev2/apis/resumable_upload_v2_complete_multipart_upload"
	"github.com/qiniu/go-sdk/v7/storagev2/retrier"
	resumablerecorder "github.com/qiniu/go-sdk/v7/storagev2/uploader/resumable_recorder"
	"github.com/qiniu/go-sdk/v7/storagev2/uploader/source"
)

type (
	multiPartsUploaderV2 struct {
		storage *apis.Storage
		options *MultiPartsUploaderOptions
	}

	multiPartsUploaderV2InitializedParts struct {
		bucketName, uploadID    string
		multiPartsObjectOptions *MultiPartsObjectOptions
		expiredAt               time.Time
		records                 map[uint64]resumedMultiPartsUploaderV2Record
		medium                  resumablerecorder.WriteableResumableRecorderMedium
		src                     source.Source
	}

	multiPartsUploaderV2UploadedPart struct {
		partNumber, offset, size uint64
		etag                     string
		md5                      [md5.Size]byte
	}

	resumedMultiPartsUploaderV2Record struct {
		uploadID, etag           string
		md5                      [md5.Size]byte
		partNumber, offset, size uint64
		expiredAt                time.Time
	}
)

// 创建分片上传器 V2
func NewMultiPartsUploaderV2(options *MultiPartsUploaderOptions) MultiPartsUploader {
	if options == nil {
		options = &MultiPartsUploaderOptions{}
	}
	return &multiPartsUploaderV2{apis.NewStorage(&options.Options), options}
}

func (uploader *multiPartsUploaderV2) InitializeParts(ctx context.Context, src source.Source, multiPartsObjectOptions *MultiPartsObjectOptions) (InitializedParts, error) {
	if multiPartsObjectOptions == nil {
		multiPartsObjectOptions = &MultiPartsObjectOptions{}
	}
	if multiPartsObjectOptions.PartSize == 0 {
		multiPartsObjectOptions.PartSize = 1 << 22
	}

	upToken, err := getUpToken(uploader.options.Credentials, &multiPartsObjectOptions.ObjectOptions, uploader.options.UpTokenProvider)
	if err != nil {
		return nil, err
	}

	bucketName, err := guessBucketName(ctx, multiPartsObjectOptions.BucketName, upToken)
	if err != nil {
		return nil, err
	} else if multiPartsObjectOptions.BucketName == "" {
		multiPartsObjectOptions.BucketName = bucketName
	}

	response, err := uploader.storage.ResumableUploadV2InitiateMultipartUpload(ctx, &apis.ResumableUploadV2InitiateMultipartUploadRequest{
		BucketName: bucketName,
		ObjectName: multiPartsObjectOptions.ObjectName,
		UpToken:    upToken,
	}, &apis.Options{
		OverwrittenRegion: multiPartsObjectOptions.RegionsProvider,
	})
	if err != nil {
		return nil, err
	}

	medium := tryToOpenResumableRecorderForAppending(ctx, src, multiPartsObjectOptions, uploader.options)
	return &multiPartsUploaderV2InitializedParts{
		bucketName:              bucketName,
		uploadID:                response.UploadId,
		multiPartsObjectOptions: multiPartsObjectOptions,
		expiredAt:               time.Unix(response.ExpiredAt, 0),
		medium:                  medium,
		src:                     src,
	}, nil
}

func (uploader *multiPartsUploaderV2) TryToResume(ctx context.Context, src source.Source, multiPartsObjectOptions *MultiPartsObjectOptions) InitializedParts {
	if multiPartsObjectOptions == nil {
		multiPartsObjectOptions = &MultiPartsObjectOptions{}
	}
	if multiPartsObjectOptions.PartSize == 0 {
		multiPartsObjectOptions.PartSize = 1 << 22
	}

	upToken, err := getUpToken(uploader.options.Credentials, &multiPartsObjectOptions.ObjectOptions, uploader.options.UpTokenProvider)
	if err != nil {
		return nil
	}

	bucketName, err := guessBucketName(ctx, multiPartsObjectOptions.BucketName, upToken)
	if err != nil {
		return nil
	} else if multiPartsObjectOptions.BucketName == "" {
		multiPartsObjectOptions.BucketName = bucketName
	}

	readableMedium := tryToOpenResumableRecorderForReading(ctx, src, multiPartsObjectOptions, uploader.options)
	if readableMedium == nil {
		return nil
	}
	defer readableMedium.Close()

	var (
		records   = make(map[uint64]resumedMultiPartsUploaderV2Record)
		uploadID  string
		expiredAt time.Time
	)
	for {
		var record resumablerecorder.ResumableRecord
		if err := readableMedium.Next(&record); err != nil {
			break
		}
		records[record.PartNumber] = resumedMultiPartsUploaderV2Record{
			uploadID:   record.UploadID,
			etag:       record.PartID,
			md5:        record.MD5,
			partNumber: record.PartNumber,
			offset:     record.Offset,
			size:       record.PartSize,
			expiredAt:  record.ExpiredAt,
		}
		if uploadID == "" {
			uploadID = record.UploadID
			expiredAt = record.ExpiredAt
		}
	}
	readableMedium.Close()
	if uploadID == "" {
		return nil
	}

	medium := tryToOpenResumableRecorderForAppending(ctx, src, multiPartsObjectOptions, uploader.options)
	return &multiPartsUploaderV2InitializedParts{
		bucketName:              bucketName,
		uploadID:                uploadID,
		multiPartsObjectOptions: multiPartsObjectOptions,
		expiredAt:               expiredAt,
		records:                 records,
		medium:                  medium,
		src:                     src,
	}
}

func (uploader *multiPartsUploaderV2) UploadPart(ctx context.Context, initialized InitializedParts, part source.Part, options *UploadPartOptions) (UploadedPart, error) {
	initializedParts, ok := initialized.(*multiPartsUploaderV2InitializedParts)
	if !ok {
		return nil, errors.New("unrecognized initialized parts")
	}
	if len(initializedParts.records) > 0 {
		if record, ok := initializedParts.records[part.PartNumber()]; ok {
			if record.offset == part.Offset() && record.size == part.Size() {
				if options != nil && options.OnUploadingProgress != nil {
					options.OnUploadingProgress(&UploadingPartProgress{Uploaded: record.size, PartSize: record.size})
				}
				return multiPartsUploaderV2UploadedPart{
					partNumber: record.partNumber,
					offset:     record.offset,
					size:       record.size,
					etag:       record.etag,
					md5:        record.md5,
				}, nil
			}
		}
	}
	return uploader.uploadPart(ctx, initializedParts, part, options)
}

func (uploader *multiPartsUploaderV2) uploadPart(ctx context.Context, initialized *multiPartsUploaderV2InitializedParts, part source.Part, options *UploadPartOptions) (UploadedPart, error) {
	apisOptions := apis.Options{
		OverwrittenRegion: initialized.multiPartsObjectOptions.RegionsProvider,
	}
	if options != nil && options.OnUploadingProgress != nil {
		apisOptions.OnRequestProgress = func(uploaded, partSize uint64) {
			options.OnUploadingProgress(&UploadingPartProgress{Uploaded: uploaded, PartSize: partSize})
		}
	}
	upToken, err := getUpToken(uploader.options.Credentials, &initialized.multiPartsObjectOptions.ObjectOptions, uploader.options.UpTokenProvider)
	if err != nil {
		return nil, err
	}

	md5, err := md5FromReadSeeker(part)
	if err != nil {
		return nil, err
	}

	response, err := uploader.storage.ResumableUploadV2UploadPart(ctx, &apis.ResumableUploadV2UploadPartRequest{
		BucketName: initialized.bucketName,
		ObjectName: initialized.multiPartsObjectOptions.ObjectName,
		UploadId:   initialized.uploadID,
		PartNumber: int64(part.PartNumber()),
		Md5:        hex.EncodeToString(md5[:]),
		UpToken:    upToken,
		Body:       internal_io.MakeReadSeekCloserFromReader(part),
	}, &apisOptions)
	if err != nil {
		return nil, err
	}

	if medium := initialized.medium; medium != nil {
		medium.Write(&resumablerecorder.ResumableRecord{
			UploadID:   initialized.uploadID,
			PartID:     response.Etag,
			Offset:     part.Offset(),
			PartSize:   part.Size(),
			PartNumber: part.PartNumber(),
			ExpiredAt:  initialized.expiredAt,
			MD5:        md5,
		})
	}

	return multiPartsUploaderV2UploadedPart{
		partNumber: part.PartNumber(),
		offset:     part.Offset(),
		size:       part.Size(),
		etag:       response.Etag,
		md5:        md5,
	}, nil
}

func (uploader *multiPartsUploaderV2) CompleteParts(ctx context.Context, initialized InitializedParts, parts []UploadedPart, returnValue interface{}) error {
	initializedParts, ok := initialized.(*multiPartsUploaderV2InitializedParts)
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

	completedParts := make(resumable_upload_v2_complete_multipart_upload.Parts, 0, len(parts))
	for _, part := range parts {
		uploadedPart, ok := part.(multiPartsUploaderV2UploadedPart)
		if !ok {
			return errors.New("unrecognized uploaded part")
		}
		completedParts = append(completedParts, resumable_upload_v2_complete_multipart_upload.PartInfo{
			PartNumber: int64(uploadedPart.partNumber),
			Etag:       uploadedPart.etag,
		})
	}

	metadata := make(map[string]string)
	for k, v := range initializedParts.multiPartsObjectOptions.Metadata {
		metadata[normalizeMetadataKey(k)] = v
	}

	customVars := make(map[string]string)
	for k, v := range initializedParts.multiPartsObjectOptions.CustomVars {
		customVars[normalizeCustomVarKey(k)] = v
	}

	_, err = uploader.storage.ResumableUploadV2CompleteMultipartUpload(ctx, &apis.ResumableUploadV2CompleteMultipartUploadRequest{
		BucketName:   initializedParts.bucketName,
		ObjectName:   initializedParts.multiPartsObjectOptions.ObjectName,
		UploadId:     initializedParts.uploadID,
		UpToken:      upToken,
		Parts:        completedParts,
		FileName:     initializedParts.multiPartsObjectOptions.FileName,
		MimeType:     initializedParts.multiPartsObjectOptions.ContentType,
		Metadata:     metadata,
		CustomVars:   customVars,
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

func (uploader *multiPartsUploaderV2) MultiPartsUploaderOptions() *MultiPartsUploaderOptions {
	return uploader.options
}

func (initialized *multiPartsUploaderV2InitializedParts) Close() error {
	if initialized.medium != nil {
		return initialized.medium.Close()
	}
	return nil
}

func (uploadedPart multiPartsUploaderV2UploadedPart) Offset() uint64 {
	return uploadedPart.offset
}

func (uploadedPart multiPartsUploaderV2UploadedPart) PartNumber() uint64 {
	return uploadedPart.partNumber
}

func (uploadedPart multiPartsUploaderV2UploadedPart) PartSize() uint64 {
	return uploadedPart.size
}

func (uploadedPart multiPartsUploaderV2UploadedPart) Etag() string {
	return uploadedPart.etag
}

func (uploadedPart multiPartsUploaderV2UploadedPart) MD5() [md5.Size]byte {
	return uploadedPart.md5
}

func md5FromReadSeeker(r io.ReadSeeker) (md5Result [md5.Size]byte, err error) {
	var offset int64

	offset, err = r.Seek(0, io.SeekCurrent)
	if err != nil {
		return
	}
	hasher := md5.New()
	if _, err = io.Copy(hasher, r); err != nil {
		return
	}
	if _, err = r.Seek(offset, io.SeekStart); err != nil {
		return
	}
	copy(md5Result[:], hasher.Sum(nil))
	return
}
