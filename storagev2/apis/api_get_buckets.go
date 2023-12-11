// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/json"
	auth "github.com/qiniu/go-sdk/v7/auth"
	getbuckets "github.com/qiniu/go-sdk/v7/storagev2/apis/get_buckets"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"net/url"
	"strings"
)

type innerGetBucketsRequest getbuckets.Request

func (query *innerGetBucketsRequest) buildQuery() (url.Values, error) {
	allQuery := make(url.Values)
	if query.Shared != "" {
		allQuery.Set("shared", query.Shared)
	}
	return allQuery, nil
}
func (j *innerGetBucketsRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*getbuckets.Request)(j))
}
func (j *innerGetBucketsRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*getbuckets.Request)(j))
}
func (request *innerGetBucketsRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

type GetBucketsRequest = getbuckets.Request
type GetBucketsResponse = getbuckets.Response

// 获取拥有的所有存储空间列表
func (storage *Storage) GetBuckets(ctx context.Context, request *GetBucketsRequest, options *Options) (response *GetBucketsResponse, err error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerGetBucketsRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceBucket}
	var pathSegments []string
	pathSegments = append(pathSegments, "buckets")
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	if query, err := innerRequest.buildQuery(); err != nil {
		return nil, err
	} else {
		rawQuery += query.Encode()
	}
	req := httpclient.Request{Method: "GET", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, AuthType: auth.TokenQiniu, Credentials: innerRequest.Credentials, BufferResponse: true}
	var queryer region.BucketRegionsQueryer
	if storage.client.GetRegions() == nil && storage.client.GetEndpoints() == nil {
		queryer = storage.client.GetBucketQueryer()
		if queryer == nil {
			bucketHosts := httpclient.DefaultBucketHosts()
			if options.OverwrittenBucketHosts != nil {
				req.Endpoints = options.OverwrittenBucketHosts
			} else {
				req.Endpoints = bucketHosts
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
	var respBody GetBucketsResponse
	if err := storage.client.DoAndAcceptJSON(ctx, &req, &respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}
