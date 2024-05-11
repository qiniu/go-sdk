package uploader

import (
	"bytes"
	"context"
	"hash/crc32"
	"io"
	"net/http"
	"os"
	"strings"
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
	formUploader struct {
		storage *apis.Storage
		options *httpclient.Options
	}

	multipartsUploader struct {
		scheduler MultiPartsUploaderScheduler
	}
)

func NewFormUploader(options *httpclient.Options) Uploader {
	if options == nil {
		options = &httpclient.Options{}
	}
	return formUploader{apis.NewStorage(options), options}
}

func (uploader formUploader) UploadPath(ctx context.Context, path string, objectParams *ObjectParams, returnValue interface{}) error {
	if objectParams == nil {
		objectParams = &ObjectParams{}
	}
	upToken, err := getUpToken(uploader.options.Credentials, objectParams)
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
	return uploader.upload(ctx, file, fileSize, upToken, objectParams.ObjectName, objectParams.FileName, objectParams.ContentType,
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
	upToken, err := getUpToken(uploader.options.Credentials, objectParams)
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
	return uploader.upload(ctx, rsc, size, upToken, objectParams.ObjectName, objectParams.FileName, objectParams.ContentType,
		crc32, mergeCustomVarsAndMetadata(objectParams.Metadata, objectParams.CustomVars), objectParams.OnUploadingProgress, returnValue)
}

func (uploader formUploader) upload(
	ctx context.Context, reader io.ReadSeeker, size uint64, upToken uptoken.Provider,
	objectName *string, fileName, contentType string, crc32 uint32, customData map[string]string,
	onRequestProgress func(uint64, uint64), returnValue interface{},
) error {
	return forEachRegion(ctx, upToken, uploader.options, func(region *region.Region) (bool, error) {
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
		options.OnRequestProgress = func(_ context.Context, _ *http.Request, uploadedInt64, _ int64) {
			uploaded := uint64(uploadedInt64)
			if uploaded > size {
				uploaded = size
			}
			onRequestProgress(uploaded, size)
		}
	}
	_, err := uploader.storage.PostObject(ctx, &request, &options)
	return err
}

func NewMultipartsUploader(scheduler MultiPartsUploaderScheduler) Uploader {
	return multipartsUploader{scheduler}
}

func (uploader multipartsUploader) UploadPath(ctx context.Context, path string, objectParams *ObjectParams, returnValue interface{}) error {
	if objectParams == nil {
		objectParams = &ObjectParams{}
	}
	httpClientOptions := uploader.scheduler.MultiPartsUploader().HttpClientOptions()
	if httpClientOptions == nil {
		httpClientOptions = &httpclient.Options{}
	}

	upToken, err := getUpToken(httpClientOptions.Credentials, objectParams)
	if err != nil {
		return err
	}

	src, err := source.NewFileSource(path)
	if err != nil {
		return err
	}
	defer src.Close()

	return uploader.upload(ctx, src, upToken, httpClientOptions, objectParams, returnValue)
}

func (uploader multipartsUploader) UploadReader(ctx context.Context, reader io.Reader, objectParams *ObjectParams, returnValue interface{}) error {
	if objectParams == nil {
		objectParams = &ObjectParams{}
	}

	httpClientOptions := uploader.scheduler.MultiPartsUploader().HttpClientOptions()
	if httpClientOptions == nil {
		httpClientOptions = &httpclient.Options{}
	}

	upToken, err := getUpToken(httpClientOptions.Credentials, objectParams)
	if err != nil {
		return err
	}

	var src source.Source
	if rscs, ok := reader.(io.ReadSeeker); ok && canSeekReally(rscs) {
		src = source.NewReadSeekCloserSource(internal_io.MakeReadSeekCloserFromReader(reader), "")
	} else {
		src = source.NewReadCloserSource(io.NopCloser(reader), "")
	}

	return uploader.upload(ctx, src, upToken, httpClientOptions, objectParams, returnValue)
}

func (uploader multipartsUploader) upload(ctx context.Context, src source.Source, upToken uptoken.Provider, httpClientOptions *httpclient.Options, objectParams *ObjectParams, returnValue interface{}) error {
	resumed, err := uploader.uploadResumedParts(ctx, src, upToken, httpClientOptions, objectParams, returnValue)
	if err == nil {
		if resumed {
			return nil
		} else {
			return uploader.tryToUploadToEachRegion(ctx, src, upToken, httpClientOptions, objectParams, returnValue)
		}
	}
	if resumed {
		if rsrc, ok := src.(source.ResetableSource); ok {
			if resetErr := rsrc.Reset(); resetErr == nil {
				return err
			}
		}
	}
	return uploader.tryToUploadToEachRegion(ctx, src, upToken, httpClientOptions, objectParams, returnValue)
}

func (uploader multipartsUploader) uploadResumedParts(ctx context.Context, src source.Source, upToken uptoken.Provider, httpClientOptions *httpclient.Options, objectParams *ObjectParams, returnValue interface{}) (bool, error) {
	if initializedParts, err := uploader.scheduler.MultiPartsUploader().TryToResume(ctx, src, objectParams); err != nil {
		return false, err
	} else if initializedParts == nil {
		return false, nil
	} else if err = uploader.uploadPartsAndComplete(ctx, src, initializedParts, objectParams, returnValue); err != nil {
		return true, err
	} else {
		return true, nil
	}
}

func (uploader multipartsUploader) tryToUploadToEachRegion(ctx context.Context, src source.Source, upToken uptoken.Provider, httpClientOptions *httpclient.Options, objectParams *ObjectParams, returnValue interface{}) error {
	return forEachRegion(ctx, upToken, httpClientOptions, func(region *region.Region) (bool, error) {
		objectParams.RegionsProvider = region
		initializedParts, err := uploader.scheduler.MultiPartsUploader().InitializeParts(ctx, src, objectParams)
		if err == nil {
			if err = uploader.uploadPartsAndComplete(ctx, src, initializedParts, objectParams, returnValue); err == nil {
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

func (uploader multipartsUploader) uploadPartsAndComplete(ctx context.Context, src source.Source, initializedParts InitializedParts, objectParams *ObjectParams, returnValue interface{}) error {
	uploadParts, err := uploader.scheduler.UploadParts(ctx, initializedParts, src)
	if err != nil {
		return err
	}
	return uploader.scheduler.MultiPartsUploader().CompleteParts(ctx, initializedParts, uploadParts, returnValue)
}

func getUpToken(credentials creds.CredentialsProvider, objectParams *ObjectParams) (uptoken.Provider, error) {
	if objectParams.UpToken != nil {
		return objectParams.UpToken, nil
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
		if !strings.HasPrefix(k, "x-qn-meta-") {
			k = "x-qn-meta-" + k
		}
		result[k] = v
	}
	for k, v := range customVars {
		if !strings.HasPrefix(k, "x:") {
			k = "x:" + k
		}
		result[k] = v
	}
	return result
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

func forEachRegion(ctx context.Context, upToken uptoken.Provider, options *httpclient.Options, fn func(*region.Region) (bool, error)) (err error) {
	var (
		regions         []*region.Region
		regionsProvider = options.Regions
		retryable       bool
	)
	if regionsProvider == nil {
		var (
			accessKey, bucketName string
			putPolicy             uptoken.PutPolicy
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
		} else if putPolicy, err = upToken.GetPutPolicy(ctx); err != nil {
			return
		} else if bucketName, err = putPolicy.GetBucketName(); err != nil {
			return
		}
		regionsProvider = query.Query(accessKey, bucketName)
	}
	if regions, err = regionsProvider.GetRegions(ctx); err != nil {
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
