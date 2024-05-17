package uploader

import (
	"bytes"
	"context"
	stderrors "errors"
	"hash/crc32"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
	"github.com/qiniu/go-sdk/v7/storagev2/apis"
	creds "github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/region"
	"github.com/qiniu/go-sdk/v7/storagev2/retrier"
	"github.com/qiniu/go-sdk/v7/storagev2/uploader/source"
	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"
	"modernc.org/fileutil"
)

type (
	FormUploaderOptions struct {
		*httpclient.Options
		UpTokenProvider uptoken.Provider
	}

	formUploader struct {
		storage *apis.Storage
		options *FormUploaderOptions
	}

	multiPartsUploader struct {
		scheduler MultiPartsUploaderScheduler
	}
)

func NewFormUploader(options *FormUploaderOptions) Uploader {
	if options == nil {
		options = &FormUploaderOptions{}
	}
	return formUploader{apis.NewStorage(options.Options), options}
}

func (uploader formUploader) UploadFile(ctx context.Context, path string, objectParams *ObjectParams, returnValue interface{}) error {
	if objectParams == nil {
		objectParams = &ObjectParams{}
	}
	upToken, err := getUpToken(uploader.options.Credentials, objectParams, uploader.options.UpTokenProvider)
	if err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	fileSize := uint64(fileInfo.Size())
	_ = fileutil.Fadvise(file, 0, 0, fileutil.POSIX_FADV_SEQUENTIAL)

	crc32, err := crc32FromReadSeeker(file)
	if err != nil {
		return err
	}
	return uploader.upload(ctx, file, fileSize, upToken, objectParams.BucketName, objectParams.ObjectName, objectParams.FileName, objectParams.ContentType,
		crc32, mergeCustomVarsAndMetadata(objectParams.Metadata, objectParams.CustomVars), objectParams.OnUploadingProgress, returnValue)
}

func (uploader formUploader) UploadReader(ctx context.Context, reader io.Reader, objectParams *ObjectParams, returnValue interface{}) error {
	var (
		rsc  io.ReadSeeker
		size uint64
		ok   bool
	)
	if objectParams == nil {
		objectParams = &ObjectParams{}
	}
	upToken, err := getUpToken(uploader.options.Credentials, objectParams, uploader.options.UpTokenProvider)
	if err != nil {
		return err
	}
	if rsc, ok = reader.(io.ReadSeeker); ok && canSeekReally(rsc) {
		if size, err = getSeekerSize(rsc); err != nil {
			return err
		}
	} else {
		dataBytes, err := internal_io.ReadAll(reader)
		if err != nil {
			return err
		}
		size = uint64(len(dataBytes))
		rsc = bytes.NewReader(dataBytes)
	}
	crc32, err := crc32FromReadSeeker(rsc)
	if err != nil {
		return err
	}
	return uploader.upload(ctx, rsc, size, upToken, objectParams.BucketName, objectParams.ObjectName, objectParams.FileName, objectParams.ContentType,
		crc32, mergeCustomVarsAndMetadata(objectParams.Metadata, objectParams.CustomVars), objectParams.OnUploadingProgress, returnValue)
}

func (uploader formUploader) upload(
	ctx context.Context, reader io.ReadSeeker, size uint64, upToken uptoken.Provider, bucketName string,
	objectName *string, fileName, contentType string, crc32 uint32, customData map[string]string,
	onRequestProgress func(uint64, uint64), returnValue interface{},
) error {
	return forEachRegion(ctx, upToken, bucketName, uploader.options.Options, func(region *region.Region) (bool, error) {
		err := uploader.uploadToRegion(ctx, region, reader, size, upToken, objectName, fileName, contentType,
			crc32, customData, onRequestProgress, returnValue)
		return true, err
	})
}

func (uploader formUploader) uploadToRegion(
	ctx context.Context, region *region.Region, reader io.ReadSeeker, size uint64, upToken uptoken.Provider,
	objectName *string, fileName, contentType string, crc32 uint32, customData map[string]string,
	onRequestProgress func(uint64, uint64), returnValue interface{},
) error {
	options := apis.Options{OverwrittenRegion: region}
	request := apis.PostObjectRequest{
		ObjectName:  objectName,
		UploadToken: upToken,
		Crc32:       int64(crc32),
		File: httpclient.MultipartFormBinaryData{
			Data:        internal_io.NewReadSeekableNopCloser(reader),
			Name:        fileName,
			ContentType: contentType,
		},
		CustomData:   customData,
		ResponseBody: returnValue,
	}
	if onRequestProgress != nil {
		options.OnRequestProgress = func(uploaded, _ uint64) {
			if uploaded > size {
				uploaded = size
			}
			onRequestProgress(uploaded, size)
		}
	}
	_, err := uploader.storage.PostObject(ctx, &request, &options)
	return err
}

func NewMultiPartsUploader(scheduler MultiPartsUploaderScheduler) Uploader {
	return multiPartsUploader{scheduler}
}

func (uploader multiPartsUploader) UploadFile(ctx context.Context, path string, objectParams *ObjectParams, returnValue interface{}) error {
	if objectParams == nil {
		objectParams = &ObjectParams{}
	}
	options := uploader.scheduler.MultiPartsUploader().MultiPartsUploaderOptions()
	if options == nil {
		options = &MultiPartsUploaderOptions{}
	}

	upToken, err := getUpToken(options.Credentials, objectParams, options.UpTokenProvider)
	if err != nil {
		return err
	}

	src, err := source.NewFileSource(path)
	if err != nil {
		return err
	}
	defer src.Close()

	if file := src.GetFile(); file != nil {
		_ = fileutil.Fadvise(file, 0, 0, fileutil.POSIX_FADV_SEQUENTIAL)
	}

	return uploader.upload(ctx, src, upToken, options.Options, objectParams, returnValue)
}

func (uploader multiPartsUploader) UploadReader(ctx context.Context, reader io.Reader, objectParams *ObjectParams, returnValue interface{}) error {
	if objectParams == nil {
		objectParams = &ObjectParams{}
	}

	options := uploader.scheduler.MultiPartsUploader().MultiPartsUploaderOptions()
	if options == nil {
		options = &MultiPartsUploaderOptions{}
	}

	upToken, err := getUpToken(options.Credentials, objectParams, options.UpTokenProvider)
	if err != nil {
		return err
	}

	var src source.Source
	if rss, ok := reader.(io.ReadSeeker); ok && canSeekReally(rss) {
		if rasc, ok := rss.(source.ReadAtSeekCloser); ok {
			src = source.NewReadAtSeekCloserSource(rasc, "")
		} else if rscs, ok := rss.(internal_io.ReadSeekCloser); ok {
			src = source.NewReadSeekCloserSource(rscs, "")
		} else {
			src = source.NewReadSeekCloserSource(internal_io.MakeReadSeekCloserFromReader(rss), "")
		}
	} else {
		src = source.NewReadCloserSource(ioutil.NopCloser(reader), "")
	}

	return uploader.upload(ctx, src, upToken, options.Options, objectParams, returnValue)
}

func (uploader multiPartsUploader) upload(ctx context.Context, src source.Source, upToken uptoken.Provider, httpClientOptions *httpclient.Options, objectParams *ObjectParams, returnValue interface{}) error {
	resumed, err := uploader.uploadResumedParts(ctx, src, upToken, httpClientOptions, objectParams, returnValue)
	if err == nil && resumed {
		return nil
	} else if resumed {
		if rsrc, ok := src.(source.ResetableSource); ok {
			if resetErr := rsrc.Reset(); resetErr == nil {
				return err
			}
		}
	}
	return uploader.tryToUploadToEachRegion(ctx, src, upToken, httpClientOptions, objectParams, returnValue)
}

func (uploader multiPartsUploader) uploadResumedParts(ctx context.Context, src source.Source, upToken uptoken.Provider, httpClientOptions *httpclient.Options, objectParams *ObjectParams, returnValue interface{}) (bool, error) {
	multiPartsObjectParams := MultiPartsObjectParams{objectParams, uploader.scheduler.PartSize()}
	if initializedParts := uploader.scheduler.MultiPartsUploader().TryToResume(ctx, src, &multiPartsObjectParams); initializedParts == nil {
		return false, nil
	} else {
		defer initializedParts.Close()
		var size uint64
		if ssrc, ok := src.(source.SizedSource); ok {
			if totalSize, sizeErr := ssrc.TotalSize(); sizeErr == nil {
				size = totalSize
			}
		}
		if err := uploader.uploadPartsAndComplete(ctx, src, size, initializedParts, objectParams, returnValue); err != nil {
			return true, err
		} else {
			return true, nil
		}
	}
}

func (uploader multiPartsUploader) tryToUploadToEachRegion(ctx context.Context, src source.Source, upToken uptoken.Provider, httpClientOptions *httpclient.Options, objectParams *ObjectParams, returnValue interface{}) error {
	return forEachRegion(ctx, upToken, objectParams.BucketName, httpClientOptions, func(region *region.Region) (bool, error) {
		objectParams.RegionsProvider = region
		multiPartsObjectParams := MultiPartsObjectParams{objectParams, uploader.scheduler.PartSize()}
		initializedParts, err := uploader.scheduler.MultiPartsUploader().InitializeParts(ctx, src, &multiPartsObjectParams)
		var size uint64
		if ssrc, ok := src.(source.SizedSource); ok {
			if totalSize, sizeErr := ssrc.TotalSize(); sizeErr == nil {
				size = totalSize
			}
		}
		if err == nil {
			defer initializedParts.Close()
			if err = uploader.uploadPartsAndComplete(ctx, src, size, initializedParts, objectParams, returnValue); err == nil {
				return true, nil
			}
		}
		if rsrc, ok := src.(source.ResetableSource); ok {
			if resetErr := rsrc.Reset(); resetErr == nil {
				return true, err
			}
		}
		return false, err
	})
}

func (uploader multiPartsUploader) uploadPartsAndComplete(ctx context.Context, src source.Source, size uint64, initializedParts InitializedParts, objectParams *ObjectParams, returnValue interface{}) error {
	var uploadPartsParams UploadPartsParams
	if objectParams.OnUploadingProgress != nil {
		progress := newUploadingPartsProgress()
		uploadPartsParams.OnUploadingProgress = func(partNumber, uploaded, _ uint64) {
			progress.setPartUploadingProgress(partNumber, uploaded)
			objectParams.OnUploadingProgress(progress.totalUploaded(), size)
		}
		uploadPartsParams.OnPartUploaded = func(partNumber, partSize uint64) {
			progress.partUploaded(partNumber, partSize)
			objectParams.OnUploadingProgress(progress.totalUploaded(), size)
		}
	}
	uploadParts, err := uploader.scheduler.UploadParts(ctx, initializedParts, src, &uploadPartsParams)
	if err != nil {
		return err
	}
	return uploader.scheduler.MultiPartsUploader().CompleteParts(ctx, initializedParts, uploadParts, returnValue)
}

func getUpToken(credentials creds.CredentialsProvider, objectParams *ObjectParams, upTokenProvider uptoken.Provider) (uptoken.Provider, error) {
	if objectParams.UpToken != nil {
		return objectParams.UpToken, nil
	} else if upTokenProvider != nil {
		return upTokenProvider, nil
	} else if credentials != nil && objectParams.BucketName != "" {
		return newCredentialsUpTokenSigner(credentials, objectParams.BucketName, 1*time.Hour, 10*time.Minute), nil
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "UpToken"}
	}
}

func crc32FromReadSeeker(r io.ReadSeeker) (uint32, error) {
	offset, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	hasher := crc32.NewIEEE()
	if _, err = io.Copy(hasher, r); err != nil {
		return 0, err
	}
	if _, err = r.Seek(offset, io.SeekStart); err != nil {
		return 0, err
	}
	return hasher.Sum32(), nil
}

func mergeCustomVarsAndMetadata(metadata, customVars map[string]string) map[string]string {
	result := make(map[string]string, len(metadata)+len(customVars))
	for k, v := range metadata {
		result[normalizeMetadataKey(k)] = v
	}
	for k, v := range customVars {
		result[normalizeCustomVarKey(k)] = v
	}
	return result
}

func normalizeMetadataKey(k string) string {
	if !strings.HasPrefix(k, "x-qn-meta-") {
		k = "x-qn-meta-" + k
	}
	return k
}

func normalizeCustomVarKey(k string) string {
	if !strings.HasPrefix(k, "x:") {
		k = "x:" + k
	}
	return k
}

func canSeekReally(seeker io.Seeker) bool {
	_, err := seeker.Seek(0, io.SeekCurrent)
	return err == nil
}

func getSeekerSize(seeker io.Seeker) (uint64, error) {
	currentOffset, err := seeker.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	endOffset, err := seeker.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}
	_, err = seeker.Seek(currentOffset, io.SeekStart)
	if err != nil {
		return 0, err
	}
	return uint64(endOffset - currentOffset), nil
}

func getRegions(ctx context.Context, upToken uptoken.Provider, bucketName string, options *httpclient.Options) (regions []*region.Region, err error) {
	regionsProvider := options.Regions
	if regionsProvider == nil {
		var (
			accessKey string
			putPolicy uptoken.PutPolicy
		)
		query := options.BucketQuery
		if query == nil {
			bucketHosts := httpclient.DefaultBucketHosts()
			queryOptions := region.BucketRegionsQueryOptions{
				UseInsecureProtocol: options.UseInsecureProtocol,
				HostFreezeDuration:  options.HostFreezeDuration,
				Client:              options.BasicHTTPClient,
			}
			if hostRetryConfig := options.HostRetryConfig; hostRetryConfig != nil {
				queryOptions.RetryMax = hostRetryConfig.RetryMax
			}
			if query, err = region.NewBucketRegionsQuery(bucketHosts, &queryOptions); err != nil {
				return
			}
		}
		if accessKey, err = upToken.GetAccessKey(ctx); err != nil {
			return
		}
		if bucketName == "" {
			if putPolicy, err = upToken.GetPutPolicy(ctx); err != nil {
				return
			} else if bucketName, err = putPolicy.GetBucketName(); err != nil {
				return
			}
		}
		regionsProvider = query.Query(accessKey, bucketName)
	}
	regions, err = regionsProvider.GetRegions(ctx)
	return
}

func forEachRegion(ctx context.Context, upToken uptoken.Provider, bucketName string, options *httpclient.Options, fn func(*region.Region) (bool, error)) (err error) {
	var (
		regions   []*region.Region
		retryable bool
	)

	regions, err = getRegions(ctx, upToken, bucketName, options)
	if err != nil {
		return
	}
	if len(regions) == 0 {
		err = stderrors.New("none of regions got")
		return
	}
	for _, region := range regions {
		if retryable, err = fn(region); err != nil {
			if !retryable || !retrier.IsErrorRetryable(err) {
				break
			}
		} else {
			break
		}
	}
	return
}

type uploadingPartsProgress struct {
	uploaded  uint64
	uploading map[uint64]uint64
	lock      sync.Mutex
}

func newUploadingPartsProgress() *uploadingPartsProgress {
	return &uploadingPartsProgress{
		uploading: make(map[uint64]uint64),
	}
}

func (progress *uploadingPartsProgress) setPartUploadingProgress(partNumber, uploaded uint64) {
	progress.lock.Lock()
	defer progress.lock.Unlock()

	progress.uploading[partNumber] = uploaded
}

func (progress *uploadingPartsProgress) partUploaded(partNumber, partSize uint64) {
	progress.lock.Lock()
	defer progress.lock.Unlock()

	delete(progress.uploading, partNumber)
	progress.uploaded += partSize
}

func (progress *uploadingPartsProgress) totalUploaded() uint64 {
	progress.lock.Lock()
	defer progress.lock.Unlock()

	uploaded := progress.uploaded
	for _, b := range progress.uploading {
		uploaded += b
	}
	return uploaded
}
