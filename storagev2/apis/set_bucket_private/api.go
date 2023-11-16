// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 设置存储空间的访问权限
package set_bucket_private

import (
	"context"
	auth "github.com/qiniu/go-sdk/v7/auth"
	credentials "github.com/qiniu/go-sdk/v7/storagev2/credentials"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"net/url"
	"strconv"
	"strings"
)

type RequestBody struct {
	fieldBucket    string // 空间名称
	fieldIsPrivate int64  // `0`: 公开，`1`: 私有
}

func (form *RequestBody) GetBucket() string {
	return form.fieldBucket
}
func (form *RequestBody) SetBucket(value string) *RequestBody {
	form.fieldBucket = value
	return form
}
func (form *RequestBody) GetIsPrivate() int64 {
	return form.fieldIsPrivate
}
func (form *RequestBody) SetIsPrivate(value int64) *RequestBody {
	form.fieldIsPrivate = value
	return form
}
func (form *RequestBody) getBucketName() (string, error) {
	return form.fieldBucket, nil
}
func (form *RequestBody) build() (url.Values, error) {
	formValues := make(url.Values)
	if form.fieldBucket != "" {
		formValues.Set("bucket", form.fieldBucket)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Bucket"}
	}
	if form.fieldIsPrivate != 0 {
		formValues.Set("private", strconv.FormatInt(form.fieldIsPrivate, 10))
	}
	return formValues, nil
}

// 调用 API 所用的请求
type Request struct {
	BucketHosts region.EndpointsProvider
	Credentials credentials.CredentialsProvider
	Body        RequestBody
}

func (request Request) getBucketName(ctx context.Context) (string, error) {
	if bucketName, err := request.Body.getBucketName(); err != nil || bucketName != "" {
		return bucketName, err
	}
	return "", nil
}
func (request Request) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

// 获取 API 所用的响应
type Response struct{}

// API 调用客户端
type Client struct {
	client *httpclient.HttpClient
}

// 创建 API 调用客户端
func NewClient(options *httpclient.HttpClientOptions) *Client {
	client := httpclient.NewHttpClient(options)
	return &Client{client: client}
}
func (client *Client) Send(ctx context.Context, request *Request) (*Response, error) {
	serviceNames := []region.ServiceName{region.ServiceBucket}
	var pathSegments []string
	pathSegments = append(pathSegments, "private")
	path := "/" + strings.Join(pathSegments, "/")
	body, err := request.Body.build()
	if err != nil {
		return nil, err
	}
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, AuthType: auth.TokenQiniu, Credentials: request.Credentials, RequestBody: httpclient.GetFormRequestBody(body)}
	var queryer region.BucketRegionsQueryer
	if client.client.GetRegions() == nil && client.client.GetEndpoints() == nil {
		queryer = client.client.GetBucketQueryer()
		if queryer == nil {
			bucketHosts := httpclient.DefaultBucketHosts()
			if request.BucketHosts != nil {
				req.Endpoints = request.BucketHosts
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
	resp, err := client.client.Do(ctx, &req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return &Response{}, nil
}
