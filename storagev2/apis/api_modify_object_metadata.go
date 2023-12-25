// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/base64"
	"encoding/json"
	auth "github.com/qiniu/go-sdk/v7/auth"
	modifyobjectmetadata "github.com/qiniu/go-sdk/v7/storagev2/apis/modify_object_metadata"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"strings"
)

type innerModifyObjectMetadataRequest modifyobjectmetadata.Request

func (pp *innerModifyObjectMetadataRequest) getBucketName(ctx context.Context) (string, error) {
	return strings.SplitN(pp.Entry, ":", 2)[0], nil
}
func (path *innerModifyObjectMetadataRequest) buildPath() ([]string, error) {
	var allSegments []string
	if path.Entry != "" {
		allSegments = append(allSegments, base64.URLEncoding.EncodeToString([]byte(path.Entry)))
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Entry"}
	}
	if path.MimeType != "" {
		allSegments = append(allSegments, "mime", base64.URLEncoding.EncodeToString([]byte(path.MimeType)))
	}
	if path.Condition != "" {
		allSegments = append(allSegments, "cond", base64.URLEncoding.EncodeToString([]byte(path.Condition)))
	}
	for key, value := range path.MetaData {
		allSegments = append(allSegments, key)
		allSegments = append(allSegments, base64.URLEncoding.EncodeToString([]byte(value)))
	}
	return allSegments, nil
}
func (j *innerModifyObjectMetadataRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*modifyobjectmetadata.Request)(j))
}
func (j *innerModifyObjectMetadataRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*modifyobjectmetadata.Request)(j))
}
func (request *innerModifyObjectMetadataRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

type ModifyObjectMetadataRequest = modifyobjectmetadata.Request
type ModifyObjectMetadataResponse = modifyobjectmetadata.Response

// 修改文件元信息
func (storage *Storage) ModifyObjectMetadata(ctx context.Context, request *ModifyObjectMetadataRequest, options *Options) (response *ModifyObjectMetadataResponse, err error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerModifyObjectMetadataRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceRs}
	var pathSegments []string
	pathSegments = append(pathSegments, "chgm")
	if segments, err := innerRequest.buildPath(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, AuthType: auth.TokenQiniu, Credentials: innerRequest.Credentials}
	var queryer region.BucketRegionsQueryer
	if storage.client.GetRegions() == nil {
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
	resp, err := storage.client.Do(ctx, &req)
	if err != nil {
		return nil, err
	}
	return &ModifyObjectMetadataResponse{}, resp.Body.Close()
}
