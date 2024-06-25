// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	resumableuploadv1bput "github.com/qiniu/go-sdk/v7/storagev2/apis/resumable_upload_v1_bput"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	uplog "github.com/qiniu/go-sdk/v7/storagev2/internal/uplog"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"strconv"
	"strings"
)

type innerResumableUploadV1BputRequest resumableuploadv1bput.Request

func (request *innerResumableUploadV1BputRequest) getBucketName(ctx context.Context) (string, error) {
	if request.UpToken != nil {
		if putPolicy, err := request.UpToken.GetPutPolicy(ctx); err != nil {
			return "", err
		} else {
			return putPolicy.GetBucketName()
		}
	}
	return "", nil
}
func (path *innerResumableUploadV1BputRequest) buildPath() ([]string, error) {
	var allSegments []string
	if path.Ctx != "" {
		allSegments = append(allSegments, path.Ctx)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Ctx"}
	}
	allSegments = append(allSegments, strconv.FormatInt(path.ChunkOffset, 10))
	return allSegments, nil
}
func (request *innerResumableUploadV1BputRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.UpToken != nil {
		return request.UpToken.GetAccessKey(ctx)
	}
	return "", nil
}

type ResumableUploadV1BputRequest = resumableuploadv1bput.Request
type ResumableUploadV1BputResponse = resumableuploadv1bput.Response

// 上传指定块的一片数据，具体数据量可根据现场环境调整，同一块的每片数据必须串行上传
func (storage *Storage) ResumableUploadV1Bput(ctx context.Context, request *ResumableUploadV1BputRequest, options *Options) (*ResumableUploadV1BputResponse, error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerResumableUploadV1BputRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceUp}
	if innerRequest.UpToken == nil {
		return nil, errors.MissingRequiredFieldError{Name: "UpToken"}
	}
	var pathSegments []string
	pathSegments = append(pathSegments, "bput")
	if segments, err := innerRequest.buildPath(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	body := innerRequest.Body
	if body == nil {
		return nil, errors.MissingRequiredFieldError{Name: "Body"}
	}
	bucketName := options.OverwrittenBucketName
	if bucketName == "" {
		var err error
		if bucketName, err = innerRequest.getBucketName(ctx); err != nil {
			return nil, err
		}
	}
	var objectName string
	uplogInterceptor, err := uplog.NewRequestUplog("resumableUploadV1Bput", bucketName, objectName, func() (string, error) {
		return innerRequest.UpToken.GetUpToken(ctx)
	})
	if err != nil {
		return nil, err
	}
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, Endpoints: options.OverwrittenEndpoints, Region: options.OverwrittenRegion, Interceptors: []httpclient.Interceptor{uplogInterceptor}, UpToken: innerRequest.UpToken, BufferResponse: true, RequestBody: httpclient.GetRequestBodyFromReadSeekCloser(body), OnRequestProgress: options.OnRequestProgress}
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
	var respBody ResumableUploadV1BputResponse
	if err := storage.client.DoAndAcceptJSON(ctx, &req, &respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}
