// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 获取用户存储空间配额限制
package get_bucket_quota

import (
	"context"
	"encoding/json"
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
}

// 指定存储空间
func (pp *RequestPath) GetBucket() string {
	return pp.fieldBucket
}

// 指定存储空间
func (pp *RequestPath) SetBucket(value string) *RequestPath {
	pp.fieldBucket = value
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
	return allSegments, nil
}

// 指定存储空间
func (request *Request) GetBucket() string {
	return request.path.GetBucket()
}

// 指定存储空间
func (request *Request) SetBucket(value string) *Request {
	request.path.SetBucket(value)
	return request
}

type innerResponseBody struct {
	Size  int64 `json:"size,omitempty"`  // 空间存储量配额
	Count int64 `json:"count,omitempty"` // 空间文件数配额
}

// 获取 API 所用的响应体参数
type ResponseBody struct {
	inner innerResponseBody
}

// 空间存储量配额
func (j *ResponseBody) GetSize() int64 {
	return j.inner.Size
}

// 空间存储量配额
func (j *ResponseBody) SetSize(value int64) *ResponseBody {
	j.inner.Size = value
	return j
}

// 空间文件数配额
func (j *ResponseBody) GetCount() int64 {
	return j.inner.Count
}

// 空间文件数配额
func (j *ResponseBody) SetCount(value int64) *ResponseBody {
	j.inner.Count = value
	return j
}
func (j *ResponseBody) MarshalJSON() ([]byte, error) {
	return json.Marshal(&j.inner)
}
func (j *ResponseBody) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.inner)
}

//lint:ignore U1000 may not call it
func (j *ResponseBody) validate() error {
	return nil
}

// 空间存储量配额
func (request *Response) GetSize() int64 {
	return request.body.GetSize()
}

// 空间存储量配额
func (request *Response) SetSize(value int64) *Response {
	request.body.SetSize(value)
	return request
}

// 空间文件数配额
func (request *Response) GetCount() int64 {
	return request.body.GetCount()
}

// 空间文件数配额
func (request *Response) SetCount(value int64) *Response {
	request.body.SetCount(value)
	return request
}

// 调用 API 所用的请求
type Request struct {
	overwrittenBucketHosts region.EndpointsProvider
	overwrittenBucketName  string
	path                   RequestPath
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
	if bucketName, err := request.path.getBucketName(); err != nil || bucketName != "" {
		return bucketName, err
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

// 获取请求路径
func (request *Request) GetPath() *RequestPath {
	return &request.path
}

// 设置请求路径
func (request *Request) SetPath(path RequestPath) *Request {
	request.path = path
	return request
}

// 发送请求
func (request *Request) Send(ctx context.Context, options *httpclient.HttpClientOptions) (*Response, error) {
	client := httpclient.NewHttpClient(options)
	serviceNames := []region.ServiceName{region.ServiceBucket}
	var pathSegments []string
	pathSegments = append(pathSegments, "getbucketquota")
	if segments, err := request.path.build(); err != nil {
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
	var respBody ResponseBody
	if _, err := client.AcceptJson(ctx, &req, &respBody); err != nil {
		return nil, err
	}
	return &Response{body: respBody}, nil
}

// 获取 API 所用的响应
type Response struct {
	body ResponseBody
}

// 获取请求体
func (response *Response) GetBody() *ResponseBody {
	return &response.body
}

// 设置请求体
func (response *Response) SetBody(body ResponseBody) *Response {
	response.body = body
	return response
}