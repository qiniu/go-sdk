// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/base64"
	"encoding/json"
	resumableuploadv1makefile "github.com/qiniu/go-sdk/v7/storagev2/apis/resumable_upload_v1_make_file"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"strconv"
	"strings"
)

type innerResumableUploadV1MakeFileRequest resumableuploadv1makefile.Request

func (request *innerResumableUploadV1MakeFileRequest) getBucketName(ctx context.Context) (string, error) {
	if request.UpToken != nil {
		if putPolicy, err := request.UpToken.GetPutPolicy(ctx); err != nil {
			return "", err
		} else {
			return putPolicy.GetBucketName()
		}
	}
	return "", nil
}
func (path *innerResumableUploadV1MakeFileRequest) buildPath() ([]string, error) {
	var allSegments []string
	allSegments = append(allSegments, strconv.FormatInt(path.Size, 10))
	if path.ObjectName != "" {
		allSegments = append(allSegments, "key", base64.URLEncoding.EncodeToString([]byte(path.ObjectName)))
	}
	if path.FileName != "" {
		allSegments = append(allSegments, "fname", base64.URLEncoding.EncodeToString([]byte(path.FileName)))
	}
	if path.MimeType != "" {
		allSegments = append(allSegments, "mimeType", base64.URLEncoding.EncodeToString([]byte(path.MimeType)))
	}
	for key, value := range path.CustomData {
		allSegments = append(allSegments, key)
		allSegments = append(allSegments, base64.URLEncoding.EncodeToString([]byte(value)))
	}
	return allSegments, nil
}
func (j *innerResumableUploadV1MakeFileRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*resumableuploadv1makefile.Request)(j))
}
func (j *innerResumableUploadV1MakeFileRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*resumableuploadv1makefile.Request)(j))
}
func (request *innerResumableUploadV1MakeFileRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.UpToken != nil {
		return request.UpToken.GetAccessKey(ctx)
	}
	return "", nil
}

type ResumableUploadV1MakeFileRequest = resumableuploadv1makefile.Request
type ResumableUploadV1MakeFileResponse = resumableuploadv1makefile.Response

// 将上传好的所有数据块按指定顺序合并成一个资源文件
func (storage *Storage) ResumableUploadV1MakeFile(ctx context.Context, request *ResumableUploadV1MakeFileRequest, options *Options) (*ResumableUploadV1MakeFileResponse, error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerResumableUploadV1MakeFileRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceUp}
	var pathSegments []string
	pathSegments = append(pathSegments, "mkfile")
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
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, Endpoints: options.OverwrittenEndpoints, Region: options.OverwrittenRegion, UpToken: innerRequest.UpToken, BufferResponse: true, RequestBody: httpclient.GetRequestBodyFromReadSeekCloser(body)}
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
	respBody := ResumableUploadV1MakeFileResponse{Body: innerRequest.ResponseBody}
	if err := storage.client.DoAndAcceptJSON(ctx, &req, &respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}
