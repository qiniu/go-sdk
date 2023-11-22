// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 创建一个新的存储空间
package create_bucket

import (
	"context"
	auth "github.com/qiniu/go-sdk/v7/auth"
	credentials "github.com/qiniu/go-sdk/v7/storagev2/credentials"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"strings"
)

// 调用 API 所用的路径参数
type RequestPath struct {
	fieldBucket string
	fieldRegion string
}

// 空间名称，要求在对象存储系统范围内唯一，由 3～63 个字符组成，支持小写字母、短划线-和数字，且必须以小写字母或数字开头和结尾
func (pp *RequestPath) GetBucket() string {
	return pp.fieldBucket
}

// 空间名称，要求在对象存储系统范围内唯一，由 3～63 个字符组成，支持小写字母、短划线-和数字，且必须以小写字母或数字开头和结尾
func (pp *RequestPath) SetBucket(value string) *RequestPath {
	pp.fieldBucket = value
	return pp
}

// 存储区域 ID，默认 z0
func (pp *RequestPath) GetRegion() string {
	return pp.fieldRegion
}

// 存储区域 ID，默认 z0
func (pp *RequestPath) SetRegion(value string) *RequestPath {
	pp.fieldRegion = value
	return pp
}
func (path *RequestPath) build() ([]string, error) {
	var allSegments []string
	if path.fieldBucket != "" {
		allSegments = append(allSegments, path.fieldBucket)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Bucket"}
	}
	if path.fieldRegion != "" {
		allSegments = append(allSegments, "region", path.fieldRegion)
	}
	return allSegments, nil
}

// 空间名称，要求在对象存储系统范围内唯一，由 3～63 个字符组成，支持小写字母、短划线-和数字，且必须以小写字母或数字开头和结尾
func (request *Request) GetBucket() string {
	return request.Path.GetBucket()
}

// 空间名称，要求在对象存储系统范围内唯一，由 3～63 个字符组成，支持小写字母、短划线-和数字，且必须以小写字母或数字开头和结尾
func (request *Request) SetBucket(value string) *Request {
	request.Path.SetBucket(value)
	return request
}

// 存储区域 ID，默认 z0
func (request *Request) GetRegion() string {
	return request.Path.GetRegion()
}

// 存储区域 ID，默认 z0
func (request *Request) SetRegion(value string) *Request {
	request.Path.SetRegion(value)
	return request
}

// 调用 API 所用的请求
type Request struct {
	overwrittenBucketHosts region.EndpointsProvider
	overwrittenBucketName  string
	Path                   RequestPath
	credentials            credentials.CredentialsProvider
}

// 覆盖默认的存储区域域名列表
func (request *Request) OverwriteBucketHosts(bucketHosts region.EndpointsProvider) *Request {
	request.overwrittenBucketHosts = bucketHosts
	return request
}

// 覆盖存储空间名称
func (request *Request) OverwriteBucketName(bucketName string) *Request {
	request.overwrittenBucketName = bucketName
	return request
}

// 设置鉴权
func (request *Request) SetCredentials(credentials credentials.CredentialsProvider) *Request {
	request.credentials = credentials
	return request
}
func (request *Request) getBucketName(ctx context.Context) (string, error) {
	if request.overwrittenBucketName != "" {
		return request.overwrittenBucketName, nil
	}
	return "", nil
}
func (request *Request) getAccessKey(ctx context.Context) (string, error) {
	if request.credentials != nil {
		if credentials, err := request.credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

// 发送请求
func (request *Request) Send(ctx context.Context, options *httpclient.HttpClientOptions) (*Response, error) {
	client := httpclient.NewHttpClient(options)
	serviceNames := []region.ServiceName{region.ServiceBucket}
	var pathSegments []string
	pathSegments = append(pathSegments, "mkbucketv3")
	if segments, err := request.Path.build(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, AuthType: auth.TokenQiniu, Credentials: request.credentials}
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
		if accessKey == "" {
			if credentialsProvider := client.GetCredentials(); credentialsProvider != nil {
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
	resp, err := client.Do(ctx, &req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return &Response{}, nil
}

// 获取 API 所用的响应
type Response struct{}
