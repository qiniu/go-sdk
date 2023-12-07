// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/json"
	resumableuploadv1makeblock "github.com/qiniu/go-sdk/v7/storagev2/apis/resumable_upload_v1_make_block"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"strconv"
	"strings"
)

type innerResumableUploadV1MakeBlockRequest resumableuploadv1makeblock.Request

func (request *innerResumableUploadV1MakeBlockRequest) getBucketName(ctx context.Context) (string, error) {
	if request.UpToken != nil {
		if putPolicy, err := request.UpToken.GetPutPolicy(ctx); err != nil {
			return "", err
		} else {
			return putPolicy.GetBucketName()
		}
	}
	return "", nil
}
func (path *innerResumableUploadV1MakeBlockRequest) buildPath() ([]string, error) {
	var allSegments []string
	if path.BlockSize != 0 {
		allSegments = append(allSegments, strconv.FormatInt(path.BlockSize, 10))
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "BlockSize"}
	}
	return allSegments, nil
}
func (j *innerResumableUploadV1MakeBlockRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*resumableuploadv1makeblock.Request)(j))
}
func (j *innerResumableUploadV1MakeBlockRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*resumableuploadv1makeblock.Request)(j))
}
func (request *innerResumableUploadV1MakeBlockRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.UpToken != nil {
		return request.UpToken.GetAccessKey(ctx)
	}
	return "", nil
}

type ResumableUploadV1MakeBlockRequest = resumableuploadv1makeblock.Request
type ResumableUploadV1MakeBlockResponse = resumableuploadv1makeblock.Response

// 为后续分片上传创建一个新的块，同时上传第一片数据
func (storage *Storage) ResumableUploadV1MakeBlock(ctx context.Context, request *ResumableUploadV1MakeBlockRequest, options *Options) (response *ResumableUploadV1MakeBlockResponse, err error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerResumableUploadV1MakeBlockRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceUp}
	var pathSegments []string
	pathSegments = append(pathSegments, "mkblk")
	if segments, err := innerRequest.buildPath(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, UpToken: innerRequest.UpToken, BufferResponse: true, RequestBody: httpclient.GetRequestBodyFromReadSeekCloser(innerRequest.Body)}
	var queryer region.BucketRegionsQueryer
	if storage.client.GetRegions() == nil && storage.client.GetEndpoints() == nil {
		queryer = storage.client.GetBucketQueryer()
		if queryer == nil {
			bucketHosts := httpclient.DefaultBucketHosts()
			var err error
			if options.OverwrittenBucketHosts != nil {
				if bucketHosts, err = options.OverwrittenBucketHosts.GetEndpoints(ctx); err != nil {
					return nil, err
				}
			}
			queryerOptions := region.BucketRegionsQueryerOptions{UseInsecureProtocol: storage.client.UseInsecureProtocol(), HostFreezeDuration: storage.client.GetHostFreezeDuration(), Client: storage.client.GetClient()}
			if hostRetryConfig := storage.client.GetHostRetryConfig(); hostRetryConfig != nil {
				queryerOptions.RetryMax = hostRetryConfig.RetryMax
			}
			if queryer, err = region.NewBucketRegionsQueryer(bucketHosts, &queryerOptions); err != nil {
				return nil, err
			}
		}
	}
	if queryer != nil {
		bucketName := options.OverwrittenBucketName
		var accessKey string
		var err error
		if bucketName == "" {
			if bucketName, err = innerRequest.getBucketName(ctx); err != nil {
				return nil, err
			}
		}
		if accessKey, err = innerRequest.getAccessKey(ctx); err != nil {
			return nil, err
		} else if accessKey == "" {
			if credentialsProvider := storage.client.GetCredentials(); credentialsProvider != nil {
				if creds, err := credentialsProvider.Get(ctx); err != nil {
					return nil, err
				} else if creds != nil {
					accessKey = creds.AccessKey
				}
			}
		}
		if accessKey != "" && bucketName != "" {
			req.Region = queryer.Query(accessKey, bucketName)
		}
	}
	var respBody ResumableUploadV1MakeBlockResponse
	if _, err := storage.client.DoAndAcceptJSON(ctx, &req, &respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}
