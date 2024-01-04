// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/json"
	auth "github.com/qiniu/go-sdk/v7/auth"
	setbucketcorsrules "github.com/qiniu/go-sdk/v7/storagev2/apis/set_bucket_cors_rules"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"strings"
)

type innerSetBucketCORSRulesRequest setbucketcorsrules.Request

func (pp *innerSetBucketCORSRulesRequest) getBucketName(ctx context.Context) (string, error) {
	return pp.Bucket, nil
}
func (path *innerSetBucketCORSRulesRequest) buildPath() ([]string, error) {
	var allSegments []string
	if path.Bucket != "" {
		allSegments = append(allSegments, path.Bucket)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Bucket"}
	}
	return allSegments, nil
}
func (j *innerSetBucketCORSRulesRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*setbucketcorsrules.Request)(j))
}
func (j *innerSetBucketCORSRulesRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*setbucketcorsrules.Request)(j))
}
func (request *innerSetBucketCORSRulesRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

type SetBucketCORSRulesRequest = setbucketcorsrules.Request
type SetBucketCORSRulesResponse = setbucketcorsrules.Response

// 设置空间的跨域规则
func (storage *Storage) SetBucketCORSRules(ctx context.Context, request *SetBucketCORSRulesRequest, options *Options) (*SetBucketCORSRulesResponse, error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerSetBucketCORSRulesRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceBucket}
	var pathSegments []string
	pathSegments = append(pathSegments, "corsRules", "set")
	if segments, err := innerRequest.buildPath(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	body, err := httpclient.GetJsonRequestBody(&innerRequest)
	if err != nil {
		return nil, err
	}
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, Endpoints: options.OverwrittenEndpoints, Region: options.OverwrittenRegion, AuthType: auth.TokenQiniu, Credentials: innerRequest.Credentials, RequestBody: body}
	if options.OverwrittenEndpoints == nil && options.OverwrittenRegion == nil && storage.client.GetRegions() == nil {
		query := storage.client.GetBucketQuery()
		if query == nil {
			bucketHosts := httpclient.DefaultBucketHosts()
			if options.OverwrittenBucketHosts != nil {
				req.Endpoints = options.OverwrittenBucketHosts
			} else {
				req.Endpoints = bucketHosts
			}
		}
		if query != nil {
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
				req.Region = query.Query(accessKey, bucketName)
			}
		}
	}
	resp, err := storage.client.Do(ctx, &req)
	if err != nil {
		return nil, err
	}
	return &SetBucketCORSRulesResponse{}, resp.Body.Close()
}
