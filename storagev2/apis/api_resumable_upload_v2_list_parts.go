// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/base64"
	"encoding/json"
	resumableuploadv2listparts "github.com/qiniu/go-sdk/v7/storagev2/apis/resumable_upload_v2_list_parts"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"net/url"
	"strconv"
	"strings"
)

type innerResumableUploadV2ListPartsRequest resumableuploadv2listparts.Request

func (request *innerResumableUploadV2ListPartsRequest) getBucketName(ctx context.Context) (string, error) {
	if request.UpToken != nil {
		if putPolicy, err := request.UpToken.GetPutPolicy(ctx); err != nil {
			return "", err
		} else {
			return putPolicy.GetBucketName()
		}
	}
	return "", nil
}
func (path *innerResumableUploadV2ListPartsRequest) buildPath() ([]string, error) {
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
func (query *innerResumableUploadV2ListPartsRequest) buildQuery() (url.Values, error) {
	allQuery := make(url.Values)
	if query.MaxParts != 0 {
		allQuery.Set("max-parts", strconv.FormatInt(query.MaxParts, 10))
	}
	if query.PartNumberMarker != 0 {
		allQuery.Set("part-number_marker", strconv.FormatInt(query.PartNumberMarker, 10))
	}
	return allQuery, nil
}
func (j *innerResumableUploadV2ListPartsRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*resumableuploadv2listparts.Request)(j))
}
func (j *innerResumableUploadV2ListPartsRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*resumableuploadv2listparts.Request)(j))
}
func (request *innerResumableUploadV2ListPartsRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.UpToken != nil {
		return request.UpToken.GetAccessKey(ctx)
	}
	return "", nil
}

type ResumableUploadV2ListPartsRequest = resumableuploadv2listparts.Request
type ResumableUploadV2ListPartsResponse = resumableuploadv2listparts.Response

// 列举出指定 UploadId 所属任务所有已经上传成功的分片
func (storage *Storage) ResumableUploadV2ListParts(ctx context.Context, request *ResumableUploadV2ListPartsRequest, options *Options) (response *ResumableUploadV2ListPartsResponse, err error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerResumableUploadV2ListPartsRequest)(request)
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
	if query, err := innerRequest.buildQuery(); err != nil {
		return nil, err
	} else {
		rawQuery += query.Encode()
	}
	req := httpclient.Request{Method: "GET", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, UpToken: innerRequest.UpToken, BufferResponse: true}
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
	var respBody ResumableUploadV2ListPartsResponse
	if _, err := storage.client.DoAndAcceptJSON(ctx, &req, &respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}
