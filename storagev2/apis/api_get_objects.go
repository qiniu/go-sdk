// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/json"
	auth "github.com/qiniu/go-sdk/v7/auth"
	getobjects "github.com/qiniu/go-sdk/v7/storagev2/apis/get_objects"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"net/url"
	"strconv"
	"strings"
)

type innerGetObjectsRequest getobjects.Request

func (query *innerGetObjectsRequest) getBucketName(ctx context.Context) (string, error) {
	return query.Bucket, nil
}
func (query *innerGetObjectsRequest) buildQuery() (url.Values, error) {
	allQuery := make(url.Values)
	if query.Bucket != "" {
		allQuery.Set("bucket", query.Bucket)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Bucket"}
	}
	if query.Marker != "" {
		allQuery.Set("marker", query.Marker)
	}
	if query.Limit != 0 {
		allQuery.Set("limit", strconv.FormatInt(query.Limit, 10))
	}
	if query.Prefix != "" {
		allQuery.Set("prefix", query.Prefix)
	}
	if query.Delimiter != "" {
		allQuery.Set("delimiter", query.Delimiter)
	}
	if query.NeedParts {
		allQuery.Set("needparts", strconv.FormatBool(query.NeedParts))
	}
	return allQuery, nil
}
func (j *innerGetObjectsRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*getobjects.Request)(j))
}
func (j *innerGetObjectsRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*getobjects.Request)(j))
}
func (request *innerGetObjectsRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

type GetObjectsRequest = getobjects.Request
type GetObjectsResponse = getobjects.Response

// 列举指定存储空间里的所有对象条目
func (storage *Storage) GetObjects(ctx context.Context, request *GetObjectsRequest, options *Options) (response *GetObjectsResponse, err error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerGetObjectsRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceRsf}
	var pathSegments []string
	pathSegments = append(pathSegments, "list")
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	if query, err := innerRequest.buildQuery(); err != nil {
		return nil, err
	} else {
		rawQuery += query.Encode()
	}
	req := httpclient.Request{Method: "GET", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, AuthType: auth.TokenQiniu, Credentials: innerRequest.Credentials, BufferResponse: true}
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
	var respBody GetObjectsResponse
	if err := storage.client.DoAndAcceptJSON(ctx, &req, &respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}
