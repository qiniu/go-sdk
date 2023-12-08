// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/base64"
	"encoding/json"
	auth "github.com/qiniu/go-sdk/v7/auth"
	prefetchobject "github.com/qiniu/go-sdk/v7/storagev2/apis/prefetch_object"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"strings"
)

type innerPrefetchObjectRequest prefetchobject.Request

func (pp *innerPrefetchObjectRequest) getBucketName(ctx context.Context) (string, error) {
	return strings.SplitN(pp.Entry, ":", 2)[0], nil
}
func (path *innerPrefetchObjectRequest) buildPath() ([]string, error) {
	var allSegments []string
	if path.Entry != "" {
		allSegments = append(allSegments, base64.URLEncoding.EncodeToString([]byte(path.Entry)))
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Entry"}
	}
	return allSegments, nil
}
func (j *innerPrefetchObjectRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*prefetchobject.Request)(j))
}
func (j *innerPrefetchObjectRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*prefetchobject.Request)(j))
}
func (request *innerPrefetchObjectRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

type PrefetchObjectRequest = prefetchobject.Request
type PrefetchObjectResponse = prefetchobject.Response

// 对于设置了镜像存储的空间，从镜像源站抓取指定名称的对象并存储到该空间中，如果该空间中已存在该名称的对象，则会将镜像源站的对象覆盖空间中相同名称的对象
func (storage *Storage) PrefetchObject(ctx context.Context, request *PrefetchObjectRequest, options *Options) (response *PrefetchObjectResponse, err error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerPrefetchObjectRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceIo}
	var pathSegments []string
	pathSegments = append(pathSegments, "prefetch")
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
	return &PrefetchObjectResponse{}, resp.Body.Close()
}
