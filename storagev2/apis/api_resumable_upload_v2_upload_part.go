// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/base64"
	uplog "github.com/qiniu/go-sdk/v7/internal/uplog"
	resumableuploadv2uploadpart "github.com/qiniu/go-sdk/v7/storagev2/apis/resumable_upload_v2_upload_part"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	utils "github.com/qiniu/go-sdk/v7/storagev2/internal/utils"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"net/http"
	"strconv"
	"strings"
)

type innerResumableUploadV2UploadPartRequest resumableuploadv2uploadpart.Request

func (request *innerResumableUploadV2UploadPartRequest) getBucketName(ctx context.Context) (string, error) {
	if request.UpToken != nil {
		if putPolicy, err := request.UpToken.GetPutPolicy(ctx); err != nil {
			return "", err
		} else {
			return putPolicy.GetBucketName()
		}
	}
	return "", nil
}
func (path *innerResumableUploadV2UploadPartRequest) buildPath() ([]string, error) {
	allSegments := make([]string, 0, 6)
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
	if path.UploadId != "" {
		allSegments = append(allSegments, "uploads", path.UploadId)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "UploadId"}
	}
	if path.PartNumber != 0 {
		allSegments = append(allSegments, strconv.FormatInt(path.PartNumber, 10))
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "PartNumber"}
	}
	return allSegments, nil
}
func (headers *innerResumableUploadV2UploadPartRequest) buildHeaders() (http.Header, error) {
	allHeaders := make(http.Header)
	if headers.Md5 != "" {
		allHeaders.Set("Content-MD5", headers.Md5)
	}
	return allHeaders, nil
}
func (request *innerResumableUploadV2UploadPartRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.UpToken != nil {
		return request.UpToken.GetAccessKey(ctx)
	}
	return "", nil
}

type ResumableUploadV2UploadPartRequest = resumableuploadv2uploadpart.Request
type ResumableUploadV2UploadPartResponse = resumableuploadv2uploadpart.Response

// 初始化一个 Multipart Upload 任务之后，可以根据指定的对象名称和 UploadId 来分片上传数据
func (storage *Storage) ResumableUploadV2UploadPart(ctx context.Context, request *ResumableUploadV2UploadPartRequest, options *Options) (*ResumableUploadV2UploadPartResponse, error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerResumableUploadV2UploadPartRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceUp}
	if innerRequest.UpToken == nil {
		return nil, errors.MissingRequiredFieldError{Name: "UpToken"}
	}
	pathSegments := make([]string, 0, 7)
	pathSegments = append(pathSegments, "buckets")
	if segments, err := innerRequest.buildPath(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	headers, err := innerRequest.buildHeaders()
	if err != nil {
		return nil, err
	}
	body := innerRequest.Body
	if body == nil {
		return nil, errors.MissingRequiredFieldError{Name: "Body"}
	}
	hErr := utils.HttpHeadAddContentLength(headers, body)
	if hErr != nil {
		return nil, hErr
	}
	bucketName := options.OverwrittenBucketName
	if bucketName == "" {
		var err error
		if bucketName, err = innerRequest.getBucketName(ctx); err != nil {
			return nil, err
		}
	}
	uplogInterceptor, err := uplog.NewRequestUplog("resumableUploadV2UploadPart", bucketName, "", func() (string, error) {
		return innerRequest.UpToken.GetUpToken(ctx)
	})
	if err != nil {
		return nil, err
	}
	req := httpclient.Request{Method: "PUT", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, Endpoints: options.OverwrittenEndpoints, Region: options.OverwrittenRegion, Interceptors: []httpclient.Interceptor{uplogInterceptor}, Header: headers, UpToken: innerRequest.UpToken, BufferResponse: true, RequestBody: httpclient.GetRequestBodyFromReadSeekCloser(body), OnRequestProgress: options.OnRequestProgress}
	if options.OverwrittenEndpoints == nil && options.OverwrittenRegion == nil && storage.client.GetRegions() == nil {
		bucketHosts := httpclient.DefaultBucketHosts()
		if bucketName != "" {
			query := storage.client.GetBucketQuery()
			if query == nil {
				if options.OverwrittenBucketHosts != nil {
					if bucketHosts, err = options.OverwrittenBucketHosts.GetEndpoints(ctx); err != nil {
						return nil, err
					}
				}
				queryOptions := region.BucketRegionsQueryOptions{UseInsecureProtocol: storage.client.UseInsecureProtocol(), AccelerateUploading: storage.client.AccelerateUploadingEnabled(), HostFreezeDuration: storage.client.GetHostFreezeDuration(), Resolver: storage.client.GetResolver(), Chooser: storage.client.GetChooser(), BeforeResolve: storage.client.GetBeforeResolveCallback(), AfterResolve: storage.client.GetAfterResolveCallback(), ResolveError: storage.client.GetResolveErrorCallback(), BeforeBackoff: storage.client.GetBeforeBackoffCallback(), AfterBackoff: storage.client.GetAfterBackoffCallback(), BeforeRequest: storage.client.GetBeforeRequestCallback(), AfterResponse: storage.client.GetAfterResponseCallback()}
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
				if accessKey != "" {
					req.Region = query.Query(accessKey, bucketName)
				}
			}
		}
	}
	var respBody ResumableUploadV2UploadPartResponse
	if err := storage.client.DoAndAcceptJSON(ctx, &req, &respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}
