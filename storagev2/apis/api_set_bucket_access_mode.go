// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/json"
	auth "github.com/qiniu/go-sdk/v7/auth"
	setbucketaccessmode "github.com/qiniu/go-sdk/v7/storagev2/apis/set_bucket_access_mode"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"strconv"
	"strings"
)

type innerSetBucketAccessModeRequest setbucketaccessmode.Request

func (pp *innerSetBucketAccessModeRequest) getBucketName(ctx context.Context) (string, error) {
	return pp.Bucket, nil
}
func (path *innerSetBucketAccessModeRequest) buildPath() ([]string, error) {
	var allSegments []string
	if path.Bucket != "" {
		allSegments = append(allSegments, path.Bucket)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Bucket"}
	}
	allSegments = append(allSegments, "mode", strconv.FormatInt(path.Mode, 10))
	return allSegments, nil
}
func (j *innerSetBucketAccessModeRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*setbucketaccessmode.Request)(j))
}
func (j *innerSetBucketAccessModeRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*setbucketaccessmode.Request)(j))
}
func (request *innerSetBucketAccessModeRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

type SetBucketAccessModeRequest = setbucketaccessmode.Request
type SetBucketAccessModeResponse = setbucketaccessmode.Response

// 设置存储空间的原图保护
func (storage *Storage) SetBucketAccessMode(ctx context.Context, request *SetBucketAccessModeRequest, options *Options) (response *SetBucketAccessModeResponse, err error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerSetBucketAccessModeRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceBucket}
	var pathSegments []string
	pathSegments = append(pathSegments, "accessMode")
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
	return &SetBucketAccessModeResponse{}, resp.Body.Close()
}
