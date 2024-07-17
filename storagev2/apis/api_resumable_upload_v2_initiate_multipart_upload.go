// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/base64"
	uplog "github.com/qiniu/go-sdk/v7/internal/uplog"
	resumableuploadv2initiatemultipartupload "github.com/qiniu/go-sdk/v7/storagev2/apis/resumable_upload_v2_initiate_multipart_upload"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"strings"
)

type innerResumableUploadV2InitiateMultipartUploadRequest resumableuploadv2initiatemultipartupload.Request

func (request *innerResumableUploadV2InitiateMultipartUploadRequest) getBucketName(ctx context.Context) (string, error) {
	if request.UpToken != nil {
		if putPolicy, err := request.UpToken.GetPutPolicy(ctx); err != nil {
			return "", err
		} else {
			return putPolicy.GetBucketName()
		}
	}
	return "", nil
}
func (path *innerResumableUploadV2InitiateMultipartUploadRequest) buildPath() ([]string, error) {
	var allSegments []string
	if path.BucketName != "" {
		allSegments = append(allSegments, path.BucketName)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "BucketName"}
	}
	if path.ObjectName != nil {
		allSegments = append(allSegments, "objects", base64.URLEncoding.EncodeToString([]byte(*path.ObjectName)))
	} else {
		allSegments = append(allSegments, "objects", "~")
	}
	return allSegments, nil
}
func (request *innerResumableUploadV2InitiateMultipartUploadRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.UpToken != nil {
		return request.UpToken.GetAccessKey(ctx)
	}
	return "", nil
}

type ResumableUploadV2InitiateMultipartUploadRequest = resumableuploadv2initiatemultipartupload.Request
type ResumableUploadV2InitiateMultipartUploadResponse = resumableuploadv2initiatemultipartupload.Response

// 使用 Multipart Upload 方式上传数据前，必须先调用 API 来获取一个全局唯一的 UploadId，后续的块数据通过 uploadPart API 上传，整个文件完成 completeMultipartUpload API，已经上传块的删除 abortMultipartUpload API 都依赖该 UploadId
func (storage *Storage) ResumableUploadV2InitiateMultipartUpload(ctx context.Context, request *ResumableUploadV2InitiateMultipartUploadRequest, options *Options) (*ResumableUploadV2InitiateMultipartUploadResponse, error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerResumableUploadV2InitiateMultipartUploadRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceUp}
	if innerRequest.UpToken == nil {
		return nil, errors.MissingRequiredFieldError{Name: "UpToken"}
	}
	var pathSegments []string
	pathSegments = append(pathSegments, "buckets")
	if segments, err := innerRequest.buildPath(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	pathSegments = append(pathSegments, "uploads")
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	bucketName := options.OverwrittenBucketName
	if bucketName == "" {
		var err error
		if bucketName, err = innerRequest.getBucketName(ctx); err != nil {
			return nil, err
		}
	}
	var objectName string
	uplogInterceptor, err := uplog.NewRequestUplog("resumableUploadV2InitiateMultipartUpload", bucketName, objectName, func() (string, error) {
		return innerRequest.UpToken.GetUpToken(ctx)
	})
	if err != nil {
		return nil, err
	}
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, Endpoints: options.OverwrittenEndpoints, Region: options.OverwrittenRegion, Interceptors: []httpclient.Interceptor{uplogInterceptor}, UpToken: innerRequest.UpToken, BufferResponse: true, OnRequestProgress: options.OnRequestProgress}
	if options.OverwrittenEndpoints == nil && options.OverwrittenRegion == nil && storage.client.GetRegions() == nil {
		query := storage.client.GetBucketQuery()
		if query == nil {
			bucketHosts := httpclient.DefaultBucketHosts()
			if options.OverwrittenBucketHosts != nil {
				if bucketHosts, err = options.OverwrittenBucketHosts.GetEndpoints(ctx); err != nil {
					return nil, err
				}
			}
			queryOptions := region.BucketRegionsQueryOptions{UseInsecureProtocol: storage.client.UseInsecureProtocol(), AccelerateUploading: storage.client.AccelerateUploadingEnabled(), HostFreezeDuration: storage.client.GetHostFreezeDuration(), Client: storage.client.GetClient(), Resolver: storage.client.GetResolver(), Chooser: storage.client.GetChooser(), BeforeResolve: storage.client.GetBeforeResolveCallback(), AfterResolve: storage.client.GetAfterResolveCallback(), ResolveError: storage.client.GetResolveErrorCallback(), BeforeBackoff: storage.client.GetBeforeBackoffCallback(), AfterBackoff: storage.client.GetAfterBackoffCallback(), BeforeRequest: storage.client.GetBeforeRequestCallback(), AfterResponse: storage.client.GetAfterResponseCallback()}
			if hostRetryConfig := storage.client.GetHostRetryConfig(); hostRetryConfig != nil {
				queryOptions.RetryMax = hostRetryConfig.RetryMax
				queryOptions.Backoff = hostRetryConfig.Backoff
			}
			if query, err = region.NewBucketRegionsQuery(bucketHosts, &queryOptions); err != nil {
				return nil, err
			}
		}
		if query != nil {
			var accessKey string
			var err error
			if accessKey, err = innerRequest.getAccessKey(ctx); err != nil {
				return nil, err
			}
			if accessKey == "" {
				if credentialsProvider := storage.client.GetCredentials(); credentialsProvider != nil {
					if creds, err := credentialsProvider.Get(ctx); err != nil {
						return nil, err
					} else if creds != nil {
						accessKey = creds.AccessKey
					}
				}
			}
			if accessKey != "" && bucketName != "" {
				req.Region = query.Query(accessKey, bucketName)
			}
		}
	}
	var respBody ResumableUploadV2InitiateMultipartUploadResponse
	if err := storage.client.DoAndAcceptJSON(ctx, &req, &respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}
