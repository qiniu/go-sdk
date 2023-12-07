// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/json"
	auth "github.com/qiniu/go-sdk/v7/auth"
	setbucketmaxage "github.com/qiniu/go-sdk/v7/storagev2/apis/set_bucket_max_age"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"net/url"
	"strconv"
	"strings"
)

type innerSetBucketMaxAgeRequest setbucketmaxage.Request

func (query *innerSetBucketMaxAgeRequest) getBucketName(ctx context.Context) (string, error) {
	return query.Bucket, nil
}
func (query *innerSetBucketMaxAgeRequest) buildQuery() (url.Values, error) {
	allQuery := make(url.Values)
	if query.Bucket != "" {
		allQuery.Set("bucket", query.Bucket)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Bucket"}
	}
	{
		allQuery.Set("maxAge", strconv.FormatInt(query.MaxAge, 10))
	}
	return allQuery, nil
}
func (j *innerSetBucketMaxAgeRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*setbucketmaxage.Request)(j))
}
func (j *innerSetBucketMaxAgeRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*setbucketmaxage.Request)(j))
}
func (request *innerSetBucketMaxAgeRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

type SetBucketMaxAgeRequest = setbucketmaxage.Request
type SetBucketMaxAgeResponse = setbucketmaxage.Response

// 设置存储空间的 cache-control: max-age 响应头
func (storage *Storage) SetBucketMaxAge(ctx context.Context, request *SetBucketMaxAgeRequest, options *Options) (response *SetBucketMaxAgeResponse, err error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerSetBucketMaxAgeRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceBucket}
	var pathSegments []string
	pathSegments = append(pathSegments, "maxAge")
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	if query, err := innerRequest.buildQuery(); err != nil {
		return nil, err
	} else {
		rawQuery += query.Encode()
	}
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, AuthType: auth.TokenQiniu, Credentials: innerRequest.Credentials}
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
		if bucketName == "" {
			if bucketName, err = innerRequest.getBucketName(ctx); err != nil {
				return nil, err
			}
		}
		if accessKey, err = innerRequest.getAccessKey(ctx); err != nil {
			return nil, err
		} else if accessKey == "" {
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
	return &SetBucketMaxAgeResponse{}, resp.Body.Close()
}
