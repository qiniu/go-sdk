// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/json"
	auth "github.com/qiniu/go-sdk/v7/auth"
	getasyncfetchtask "github.com/qiniu/go-sdk/v7/storagev2/apis/get_async_fetch_task"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"net/url"
	"strings"
)

type innerGetAsyncFetchTaskRequest getasyncfetchtask.Request

func (query *innerGetAsyncFetchTaskRequest) buildQuery() (url.Values, error) {
	allQuery := make(url.Values)
	if query.Id != "" {
		allQuery.Set("id", query.Id)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Id"}
	}
	return allQuery, nil
}
func (j *innerGetAsyncFetchTaskRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*getasyncfetchtask.Request)(j))
}
func (j *innerGetAsyncFetchTaskRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*getasyncfetchtask.Request)(j))
}
func (request *innerGetAsyncFetchTaskRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

type GetAsyncFetchTaskRequest = getasyncfetchtask.Request
type GetAsyncFetchTaskResponse = getasyncfetchtask.Response

// 查询异步抓取任务
func (storage *Storage) GetAsyncFetchTask(ctx context.Context, request *GetAsyncFetchTaskRequest, options *Options) (response *GetAsyncFetchTaskResponse, err error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerGetAsyncFetchTaskRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceApi}
	var pathSegments []string
	pathSegments = append(pathSegments, "sisyphus", "fetch")
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
	var respBody GetAsyncFetchTaskResponse
	if err := storage.client.DoAndAcceptJSON(ctx, &req, &respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}
