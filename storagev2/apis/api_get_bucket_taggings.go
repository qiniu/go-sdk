// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/json"
	auth "github.com/qiniu/go-sdk/v7/auth"
	getbuckettaggings "github.com/qiniu/go-sdk/v7/storagev2/apis/get_bucket_taggings"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"net/url"
	"strings"
)

type innerGetBucketTaggingsRequest getbuckettaggings.Request

func (query *innerGetBucketTaggingsRequest) getBucketName(ctx context.Context) (string, error) {
	return query.BucketName, nil
}
func (query *innerGetBucketTaggingsRequest) buildQuery() (url.Values, error) {
	allQuery := make(url.Values)
	if query.BucketName != "" {
		allQuery.Set("bucket", query.BucketName)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "BucketName"}
	}
	return allQuery, nil
}
func (j *innerGetBucketTaggingsRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*getbuckettaggings.Request)(j))
}
func (j *innerGetBucketTaggingsRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*getbuckettaggings.Request)(j))
}
func (request *innerGetBucketTaggingsRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

type GetBucketTaggingsRequest = getbuckettaggings.Request
type GetBucketTaggingsResponse = getbuckettaggings.Response

// 查询指定的存储空间已设置的标签信息
func (storage *Storage) GetBucketTaggings(ctx context.Context, request *GetBucketTaggingsRequest, options *Options) (response *GetBucketTaggingsResponse, err error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerGetBucketTaggingsRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceBucket}
	var pathSegments []string
	pathSegments = append(pathSegments, "bucketTagging")
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	if query, err := innerRequest.buildQuery(); err != nil {
		return nil, err
	} else {
		rawQuery += query.Encode()
	}
	req := httpclient.Request{Method: "GET", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, Endpoints: options.OverwrittenEndpoints, Region: options.OverwrittenRegion, AuthType: auth.TokenQiniu, Credentials: innerRequest.Credentials, BufferResponse: true}
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
	var respBody GetBucketTaggingsResponse
	if err := storage.client.DoAndAcceptJSON(ctx, &req, &respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}
