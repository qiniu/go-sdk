// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/base64"
	"encoding/json"
	auth "github.com/qiniu/go-sdk/v7/auth"
	modifyobjectstatus "github.com/qiniu/go-sdk/v7/storagev2/apis/modify_object_status"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"strconv"
	"strings"
)

type innerModifyObjectStatusRequest modifyobjectstatus.Request

func (pp *innerModifyObjectStatusRequest) getBucketName(ctx context.Context) (string, error) {
	return strings.SplitN(pp.Entry, ":", 2)[0], nil
}
func (path *innerModifyObjectStatusRequest) buildPath() ([]string, error) {
	var allSegments []string
	if path.Entry != "" {
		allSegments = append(allSegments, base64.URLEncoding.EncodeToString([]byte(path.Entry)))
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Entry"}
	}
	allSegments = append(allSegments, "status", strconv.FormatInt(path.Status, 10))
	return allSegments, nil
}
func (j *innerModifyObjectStatusRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*modifyobjectstatus.Request)(j))
}
func (j *innerModifyObjectStatusRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*modifyobjectstatus.Request)(j))
}
func (request *innerModifyObjectStatusRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

type ModifyObjectStatusRequest = modifyobjectstatus.Request
type ModifyObjectStatusResponse = modifyobjectstatus.Response

// 修改文件的存储状态，即禁用状态和启用状态间的的互相转换
func (storage *Storage) ModifyObjectStatus(ctx context.Context, request *ModifyObjectStatusRequest, options *Options) (response *ModifyObjectStatusResponse, err error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerModifyObjectStatusRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceRs}
	var pathSegments []string
	pathSegments = append(pathSegments, "chstatus")
	if segments, err := innerRequest.buildPath(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, AuthType: auth.TokenQiniu, Credentials: innerRequest.Credentials}
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
	resp, err := storage.client.Do(ctx, &req)
	if err != nil {
		return nil, err
	}
	return &ModifyObjectStatusResponse{}, resp.Body.Close()
}
