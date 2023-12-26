// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/json"
	auth "github.com/qiniu/go-sdk/v7/auth"
	getbucketquota "github.com/qiniu/go-sdk/v7/storagev2/apis/get_bucket_quota"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"strings"
)

type innerGetBucketQuotaRequest getbucketquota.Request

func (pp *innerGetBucketQuotaRequest) getBucketName(ctx context.Context) (string, error) {
	return pp.Bucket, nil
}
func (path *innerGetBucketQuotaRequest) buildPath() ([]string, error) {
	var allSegments []string
	if path.Bucket != "" {
		allSegments = append(allSegments, path.Bucket)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Bucket"}
	}
	return allSegments, nil
}
func (j *innerGetBucketQuotaRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*getbucketquota.Request)(j))
}
func (j *innerGetBucketQuotaRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*getbucketquota.Request)(j))
}
func (request *innerGetBucketQuotaRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

type GetBucketQuotaRequest = getbucketquota.Request
type GetBucketQuotaResponse = getbucketquota.Response

// 获取用户存储空间配额限制
func (storage *Storage) GetBucketQuota(ctx context.Context, request *GetBucketQuotaRequest, options *Options) (*GetBucketQuotaResponse, error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerGetBucketQuotaRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceBucket}
	var pathSegments []string
	pathSegments = append(pathSegments, "getbucketquota")
	if segments, err := innerRequest.buildPath(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, Endpoints: options.OverwrittenEndpoints, Region: options.OverwrittenRegion, AuthType: auth.TokenQiniu, Credentials: innerRequest.Credentials, BufferResponse: true}
	if options.OverwrittenEndpoints == nil && options.OverwrittenRegion == nil && storage.client.GetRegions() == nil {
		queryer := storage.client.GetBucketQueryer()
		if queryer == nil {
			bucketHosts := httpclient.DefaultBucketHosts()
			if options.OverwrittenBucketHosts != nil {
				req.Endpoints = options.OverwrittenBucketHosts
			} else {
				req.Endpoints = bucketHosts
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
	}
	var respBody GetBucketQuotaResponse
	if err := storage.client.DoAndAcceptJSON(ctx, &req, &respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}
