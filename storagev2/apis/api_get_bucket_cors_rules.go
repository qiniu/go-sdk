// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/json"
	auth "github.com/qiniu/go-sdk/v7/auth"
	getbucketcorsrules "github.com/qiniu/go-sdk/v7/storagev2/apis/get_bucket_cors_rules"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"strings"
)

type innerGetBucketCorsRulesRequest getbucketcorsrules.Request

func (pp *innerGetBucketCorsRulesRequest) getBucketName(ctx context.Context) (string, error) {
	return pp.Bucket, nil
}
func (path *innerGetBucketCorsRulesRequest) buildPath() ([]string, error) {
	var allSegments []string
	if path.Bucket != "" {
		allSegments = append(allSegments, path.Bucket)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Bucket"}
	}
	return allSegments, nil
}
func (j *innerGetBucketCorsRulesRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*getbucketcorsrules.Request)(j))
}
func (j *innerGetBucketCorsRulesRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*getbucketcorsrules.Request)(j))
}
func (request *innerGetBucketCorsRulesRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

type GetBucketCorsRulesRequest = getbucketcorsrules.Request
type GetBucketCorsRulesResponse = getbucketcorsrules.Response

// 设置空间的跨域规则
func (storage *Storage) GetBucketCorsRules(ctx context.Context, request *GetBucketCorsRulesRequest, options *Options) (response *GetBucketCorsRulesResponse, err error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerGetBucketCorsRulesRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceBucket}
	var pathSegments []string
	pathSegments = append(pathSegments, "corsRules", "get")
	if segments, err := innerRequest.buildPath(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
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
	var respBody GetBucketCorsRulesResponse
	if _, err := storage.client.DoAndAcceptJSON(ctx, &req, &respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}
