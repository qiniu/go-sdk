// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 设置存储空间的镜像源
package set_buckets_mirror

import (
	"context"
	"encoding/base64"
	auth "github.com/qiniu/go-sdk/v7/auth"
	credentials "github.com/qiniu/go-sdk/v7/storagev2/credentials"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"strings"
)

type RequestPath struct {
	fieldBucket     string
	fieldSrcSiteUrl string
	fieldHost       string
}

func (pp *RequestPath) GetBucket() string {
	return pp.fieldBucket
}
func (pp *RequestPath) SetBucket(value string) *RequestPath {
	pp.fieldBucket = value
	return pp
}
func (pp *RequestPath) GetSrcSiteUrl() string {
	return pp.fieldSrcSiteUrl
}
func (pp *RequestPath) SetSrcSiteUrl(value string) *RequestPath {
	pp.fieldSrcSiteUrl = value
	return pp
}
func (pp *RequestPath) GetHost() string {
	return pp.fieldHost
}
func (pp *RequestPath) SetHost(value string) *RequestPath {
	pp.fieldHost = value
	return pp
}
func (pp *RequestPath) getBucketName() (string, error) {
	return pp.fieldBucket, nil
}
func (path *RequestPath) build() ([]string, error) {
	var allSegments []string
	if path.fieldBucket != "" {
		allSegments = append(allSegments, path.fieldBucket)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Bucket"}
	}
	if path.fieldSrcSiteUrl != "" {
		allSegments = append(allSegments, "from", base64.URLEncoding.EncodeToString([]byte(path.fieldSrcSiteUrl)))
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "SrcSiteUrl"}
	}
	if path.fieldHost != "" {
		allSegments = append(allSegments, "host", base64.URLEncoding.EncodeToString([]byte(path.fieldHost)))
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Host"}
	}
	return allSegments, nil
}

// 调用 API 所用的请求
type Request struct {
	overwrittenBucketHosts region.EndpointsProvider
	overwrittenBucketName  string
	Path                   RequestPath
	credentials            credentials.CredentialsProvider
}

func (request Request) OverwriteBucketHosts(bucketHosts region.EndpointsProvider) Request {
	request.overwrittenBucketHosts = bucketHosts
	return request
}
func (request Request) OverwriteBucketName(bucketName string) Request {
	request.overwrittenBucketName = bucketName
	return request
}
func (request Request) SetCredentials(credentials credentials.CredentialsProvider) Request {
	request.credentials = credentials
	return request
}
func (request Request) getBucketName(ctx context.Context) (string, error) {
	if request.overwrittenBucketName != "" {
		return request.overwrittenBucketName, nil
	}
	if bucketName, err := request.Path.getBucketName(); err != nil || bucketName != "" {
		return bucketName, err
	}
	return "", nil
}
func (request Request) getAccessKey(ctx context.Context) (string, error) {
	if request.credentials != nil {
		if credentials, err := request.credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}
func (request Request) Send(ctx context.Context, options *httpclient.HttpClientOptions) (*Response, error) {
	client := httpclient.NewHttpClient(options)
	serviceNames := []region.ServiceName{region.ServiceBucket}
	var pathSegments []string
	pathSegments = append(pathSegments, "image")
	if segments, err := request.Path.build(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, AuthType: auth.TokenQiniu, Credentials: request.credentials}
	var queryer region.BucketRegionsQueryer
	if client.GetRegions() == nil && client.GetEndpoints() == nil {
		queryer = client.GetBucketQueryer()
		if queryer == nil {
			bucketHosts := httpclient.DefaultBucketHosts()
			if request.overwrittenBucketHosts != nil {
				req.Endpoints = request.overwrittenBucketHosts
			} else {
				req.Endpoints = bucketHosts
			}
		}
	}
	if queryer != nil {
		bucketName, err := request.getBucketName(ctx)
		if err != nil {
			return nil, err
		}
		accessKey, err := request.getAccessKey(ctx)
		if err != nil {
			return nil, err
		}
		if accessKey != "" && bucketName != "" {
			req.Region = queryer.Query(accessKey, bucketName)
		}
	}
	resp, err := client.Do(ctx, &req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return &Response{}, nil
}

// 获取 API 所用的响应
type Response struct{}
