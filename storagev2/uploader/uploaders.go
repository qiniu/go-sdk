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
	"github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/region"
	"github.com/qiniu/go-sdk/v7/storagev2/retrier"
	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"
	"modernc.org/fileutil"
)

type (
	formUploader struct {
		storage *apis.Storage
		options *httpclient.Options
	}
)

func NewFormUploader(options *httpclient.Options) Uploader {
	return &formUploader{apis.NewStorage(options), options}
}

func (uploader *formUploader) UploadPath(ctx context.Context, path string, objectParams *ObjectParams, returnValue interface{}) error {
	if objectParams == nil {
		objectParams = &ObjectParams{}
	}
	upToken, err := uploader.getUpToken(objectParams)
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
	return uploader.upload(ctx, file, fileSize, upToken, objectParams.ObjectName, objectParams.FileName,
		crc32, mergeCustomVarsAndMetadata(objectParams.Metadata, objectParams.CustomVars), objectParams.OnUploadingProgress, returnValue)
}

func (uploader *formUploader) UploadReader(ctx context.Context, reader io.Reader, objectParams *ObjectParams, returnValue interface{}) error {
	var (
		rsc  io.ReadSeeker
		size uint64
		ok   bool
	)
	if objectParams == nil {
		objectParams = &ObjectParams{}
	}
	upToken, err := uploader.getUpToken(objectParams)
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
	return uploader.upload(ctx, rsc, size, upToken, objectParams.ObjectName, objectParams.FileName,
		crc32, mergeCustomVarsAndMetadata(objectParams.Metadata, objectParams.CustomVars), objectParams.OnUploadingProgress, returnValue)
}

func (uploader *formUploader) upload(
	ctx context.Context, reader io.ReadSeeker, size uint64, upToken uptoken.Provider,
	objectName *string, fileName string, crc32 uint32, customData map[string]string,
	onRequestProgress func(uint64, uint64), returnValue interface{},
) error {
	return forEachRegions(ctx, upToken, uploader.options, func(region *region.Region) error {
		return uploader.uploadToRegion(ctx, region, reader, size, upToken, objectName, fileName,
			crc32, customData, onRequestProgress, returnValue)
	})
}

func (uploader *formUploader) uploadToRegion(
	ctx context.Context, region *region.Region, reader io.ReadSeeker, size uint64, upToken uptoken.Provider,
	objectName *string, fileName string, crc32 uint32, customData map[string]string,
	onRequestProgress func(uint64, uint64), returnValue interface{},
) error {
	options := apis.Options{OverwrittenRegion: region}
	request := apis.PostObjectRequest{
		ObjectName:  objectName,
		UploadToken: upToken,
		Crc32:       int64(crc32),
		File: httpclient.MultipartFormBinaryData{
			Data: internal_io.NewReadSeekableNopCloser(reader),
			Name: fileName,
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

func (uploader *formUploader) getUpToken(objectParams *ObjectParams) (uptoken.Provider, error) {
	if objectParams.UpToken != nil {
		return objectParams.UpToken, nil
	} else if uploader.options.Credentials != nil && objectParams.BucketName != "" {
		return newCredentialsUpTokenSigner(uploader.options.Credentials, objectParams.BucketName, 7*24*time.Hour, time.Hour), nil
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

func forEachRegions(ctx context.Context, upToken uptoken.Provider, options *httpclient.Options, fn func(*region.Region) error) (err error) {
	var regions []*region.Region
	regionsProvider := options.Regions
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
		if err = fn(region); err != nil {
			if !retrier.IsErrorRetryable(err) {
				break
			}
		} else {
			break
		}
	}
	return
}
