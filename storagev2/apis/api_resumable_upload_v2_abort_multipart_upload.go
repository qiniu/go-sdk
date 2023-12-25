// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/base64"
	"encoding/json"
	resumableuploadv2abortmultipartupload "github.com/qiniu/go-sdk/v7/storagev2/apis/resumable_upload_v2_abort_multipart_upload"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"strings"
)

type innerResumableUploadV2AbortMultipartUploadRequest resumableuploadv2abortmultipartupload.Request

func (request *innerResumableUploadV2AbortMultipartUploadRequest) getBucketName(ctx context.Context) (string, error) {
	if request.UpToken != nil {
		if putPolicy, err := request.UpToken.GetPutPolicy(ctx); err != nil {
			return "", err
		} else {
			return putPolicy.GetBucketName()
		}
	}
	return "", nil
}
func (path *innerResumableUploadV2AbortMultipartUploadRequest) buildPath() ([]string, error) {
	var allSegments []string
	if path.BucketName != "" {
		allSegments = append(allSegments, path.BucketName)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "BucketName"}
	}
	if path.ObjectName != "" {
		allSegments = append(allSegments, "objects", base64.URLEncoding.EncodeToString([]byte(path.ObjectName)))
	} else {
		allSegments = append(allSegments, "objects", "~")
	}
	if path.UploadId != "" {
		allSegments = append(allSegments, "uploads", path.UploadId)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "UploadId"}
	}
	return allSegments, nil
}
func (j *innerResumableUploadV2AbortMultipartUploadRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*resumableuploadv2abortmultipartupload.Request)(j))
}
func (j *innerResumableUploadV2AbortMultipartUploadRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*resumableuploadv2abortmultipartupload.Request)(j))
}
func (request *innerResumableUploadV2AbortMultipartUploadRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.UpToken != nil {
		return request.UpToken.GetAccessKey(ctx)
	}
	return "", nil
}

type ResumableUploadV2AbortMultipartUploadRequest = resumableuploadv2abortmultipartupload.Request
type ResumableUploadV2AbortMultipartUploadResponse = resumableuploadv2abortmultipartupload.Response

// 根据 UploadId 终止 Multipart Upload
func (storage *Storage) ResumableUploadV2AbortMultipartUpload(ctx context.Context, request *ResumableUploadV2AbortMultipartUploadRequest, options *Options) (response *ResumableUploadV2AbortMultipartUploadResponse, err error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerResumableUploadV2AbortMultipartUploadRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceUp}
	var pathSegments []string
	pathSegments = append(pathSegments, "buckets")
	if segments, err := innerRequest.buildPath(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	req := httpclient.Request{Method: "DELETE", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, Endpoints: options.OverwrittenEndpoints, Region: options.OverwrittenRegion, UpToken: innerRequest.UpToken}
	if options.OverwrittenEndpoints == nil && options.OverwrittenRegion == nil && storage.client.GetRegions() == nil {
		queryer := storage.client.GetBucketQueryer()
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
				req.Region = queryer.Query(accessKey, bucketName)
			}
		}
	}
	resp, err := storage.client.Do(ctx, &req)
	if err != nil {
		return nil, err
	}
	return &ResumableUploadV2AbortMultipartUploadResponse{}, resp.Body.Close()
}
