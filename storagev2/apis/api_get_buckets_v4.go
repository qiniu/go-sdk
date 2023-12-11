// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/json"
	auth "github.com/qiniu/go-sdk/v7/auth"
	getbucketsv4 "github.com/qiniu/go-sdk/v7/storagev2/apis/get_buckets_v4"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"net/url"
	"strconv"
	"strings"
)

type innerGetBucketsV4Request getbucketsv4.Request

func (query *innerGetBucketsV4Request) buildQuery() (url.Values, error) {
	allQuery := make(url.Values)
	if query.Region != "" {
		allQuery.Set("region", query.Region)
	}
	if query.Limit != 0 {
		allQuery.Set("limit", strconv.FormatInt(query.Limit, 10))
	}
	if query.Marker != "" {
		allQuery.Set("marker", query.Marker)
	}
	return allQuery, nil
}
func (j *innerGetBucketsV4Request) MarshalJSON() ([]byte, error) {
	return json.Marshal((*getbucketsv4.Request)(j))
}
func (j *innerGetBucketsV4Request) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*getbucketsv4.Request)(j))
}
func (request *innerGetBucketsV4Request) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

type GetBucketsV4Request = getbucketsv4.Request
type GetBucketsV4Response = getbucketsv4.Response

// 获取拥有的所有存储空间列表
func (storage *Storage) GetBucketsV4(ctx context.Context, request *GetBucketsV4Request, options *Options) (response *GetBucketsV4Response, err error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerGetBucketsV4Request)(request)
	serviceNames := []region.ServiceName{region.ServiceBucket}
	var pathSegments []string
	pathSegments = append(pathSegments, "buckets")
	path := "/" + strings.Join(pathSegments, "/")
	rawQuery := "apiVersion=v4&"
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
	var respBody GetBucketsV4Response
	if err := storage.client.DoAndAcceptJSON(ctx, &req, &respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}
