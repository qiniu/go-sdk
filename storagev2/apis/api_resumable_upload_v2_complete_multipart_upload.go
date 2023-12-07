// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/base64"
	"encoding/json"
	resumableuploadv2completemultipartupload "github.com/qiniu/go-sdk/v7/storagev2/apis/resumable_upload_v2_complete_multipart_upload"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"strings"
)

type innerResumableUploadV2CompleteMultipartUploadRequest resumableuploadv2completemultipartupload.Request

func (request *innerResumableUploadV2CompleteMultipartUploadRequest) getBucketName(ctx context.Context) (string, error) {
	if request.UpToken != nil {
		if putPolicy, err := request.UpToken.GetPutPolicy(ctx); err != nil {
			return "", err
		} else {
			return putPolicy.GetBucketName()
		}
	}
	return "", nil
}
func (path *innerResumableUploadV2CompleteMultipartUploadRequest) buildPath() ([]string, error) {
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
func (j *innerResumableUploadV2CompleteMultipartUploadRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*resumableuploadv2completemultipartupload.Request)(j))
}
func (j *innerResumableUploadV2CompleteMultipartUploadRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*resumableuploadv2completemultipartupload.Request)(j))
}
func (request *innerResumableUploadV2CompleteMultipartUploadRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.UpToken != nil {
		return request.UpToken.GetAccessKey(ctx)
	}
	return "", nil
}

type ResumableUploadV2CompleteMultipartUploadRequest = resumableuploadv2completemultipartupload.Request
type ResumableUploadV2CompleteMultipartUploadResponse = resumableuploadv2completemultipartupload.Response

// 在将所有数据分片都上传完成后，必须调用 completeMultipartUpload API 来完成整个文件的 Multipart Upload。用户需要提供有效数据的分片列表（包括 PartNumber 和调用 uploadPart API 服务端返回的 Etag）。服务端收到用户提交的分片列表后，会逐一验证每个数据分片的有效性。当所有的数据分片验证通过后，会把这些数据分片组合成一个完整的对象
func (storage *Storage) ResumableUploadV2CompleteMultipartUpload(ctx context.Context, request *ResumableUploadV2CompleteMultipartUploadRequest, options *Options) (response *ResumableUploadV2CompleteMultipartUploadResponse, err error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerResumableUploadV2CompleteMultipartUploadRequest)(request)
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
	body, err := httpclient.GetJsonRequestBody(&innerRequest)
	if err != nil {
		return nil, err
	}
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, UpToken: innerRequest.UpToken, BufferResponse: true, RequestBody: body}
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
		if bucketName == "" {
			if upTokenProvider := storage.client.GetUpToken(); upTokenProvider != nil {
				if putPolicy, err := upTokenProvider.GetPutPolicy(ctx); err != nil {
					return nil, err
				} else if bucketName, err = putPolicy.GetBucketName(); err != nil {
					return nil, err
				}
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
		if accessKey == "" {
			if upTokenProvider := storage.client.GetUpToken(); upTokenProvider != nil {
				if accessKey, err = upTokenProvider.GetAccessKey(ctx); err != nil {
					return nil, err
				}
			}
		}
		if accessKey != "" && bucketName != "" {
			req.Region = queryer.Query(accessKey, bucketName)
		}
	}
	var respBody ResumableUploadV2CompleteMultipartUploadResponse
	if _, err := storage.client.DoAndAcceptJSON(ctx, &req, &respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}
