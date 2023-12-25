// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/base64"
	"encoding/json"
	auth "github.com/qiniu/go-sdk/v7/auth"
	setbucketsmirror "github.com/qiniu/go-sdk/v7/storagev2/apis/set_buckets_mirror"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"strings"
)

type innerSetBucketsMirrorRequest setbucketsmirror.Request

func (pp *innerSetBucketsMirrorRequest) getBucketName(ctx context.Context) (string, error) {
	return pp.Bucket, nil
}
func (path *innerSetBucketsMirrorRequest) buildPath() ([]string, error) {
	var allSegments []string
	if path.Bucket != "" {
		allSegments = append(allSegments, path.Bucket)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Bucket"}
	}
	if path.SrcSiteUrl != "" {
		allSegments = append(allSegments, "from", base64.URLEncoding.EncodeToString([]byte(path.SrcSiteUrl)))
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "SrcSiteUrl"}
	}
	if path.Host != "" {
		allSegments = append(allSegments, "host", base64.URLEncoding.EncodeToString([]byte(path.Host)))
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Host"}
	}
	return allSegments, nil
}
func (j *innerSetBucketsMirrorRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*setbucketsmirror.Request)(j))
}
func (j *innerSetBucketsMirrorRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*setbucketsmirror.Request)(j))
}
func (request *innerSetBucketsMirrorRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

type SetBucketsMirrorRequest = setbucketsmirror.Request
type SetBucketsMirrorResponse = setbucketsmirror.Response

// 设置存储空间的镜像源
func (storage *Storage) SetBucketsMirror(ctx context.Context, request *SetBucketsMirrorRequest, options *Options) (response *SetBucketsMirrorResponse, err error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerSetBucketsMirrorRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceBucket}
	var pathSegments []string
	pathSegments = append(pathSegments, "image")
	if segments, err := innerRequest.buildPath(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, Endpoints: options.OverwrittenEndpoints, Region: options.OverwrittenRegion, AuthType: auth.TokenQiniu, Credentials: innerRequest.Credentials}
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
	resp, err := storage.client.Do(ctx, &req)
	if err != nil {
		return nil, err
	}
	return &SetBucketsMirrorResponse{}, resp.Body.Close()
}
