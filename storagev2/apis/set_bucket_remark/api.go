// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 设置空间备注
package set_bucket_remark

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

// 空间名称
func (pp *RequestPath) GetBucket() string {
	return pp.fieldBucket
}

// 空间名称
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

// 空间名称
func (request *Request) GetBucket() string {
	return request.Path.GetBucket()
}

// 空间名称
func (request *Request) SetBucket(value string) *Request {
	request.Path.SetBucket(value)
	return request
}

type innerRequestBody struct {
	Remark string `json:"remark"` // 空间备注信息, 字符长度不能超过 100, 允许为空
}

// 调用 API 所用的请求体
type RequestBody struct {
	inner innerRequestBody
}

// 空间备注信息, 字符长度不能超过 100, 允许为空
func (j *RequestBody) GetRemark() string {
	return j.inner.Remark
}

// 空间备注信息, 字符长度不能超过 100, 允许为空
func (j *RequestBody) SetRemark(value string) *RequestBody {
	j.inner.Remark = value
	return j
}
func (j *RequestBody) MarshalJSON() ([]byte, error) {
	return json.Marshal(&j.inner)
}
func (j *RequestBody) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.inner)
}

//lint:ignore U1000 may not call it
func (j *RequestBody) validate() error {
	if j.inner.Remark == "" {
		return errors.MissingRequiredFieldError{Name: "Remark"}
	}
	return nil
}

// 空间备注信息, 字符长度不能超过 100, 允许为空
func (request *Request) GetRemark() string {
	return request.Body.GetRemark()
}

// 空间备注信息, 字符长度不能超过 100, 允许为空
func (request *Request) SetRemark(value string) *Request {
	request.Body.SetRemark(value)
	return request
}

// 调用 API 所用的请求
type Request struct {
	overwrittenBucketHosts region.EndpointsProvider
	overwrittenBucketName  string
	Path                   RequestPath
	credentials            credentials.CredentialsProvider
	Body                   RequestBody
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
	if bucketName, err := request.Path.getBucketName(); err != nil || bucketName != "" {
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

// 发送请求
func (request *Request) Send(ctx context.Context, options *httpclient.HttpClientOptions) (*Response, error) {
	client := httpclient.NewHttpClient(options)
	serviceNames := []region.ServiceName{region.ServiceBucket}
	var pathSegments []string
	pathSegments = append(pathSegments, "buckets")
	if segments, err := request.Path.build(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	rawQuery += "remark" + "&"
	if err := request.Body.validate(); err != nil {
		return nil, err
	}
	body, err := httpclient.GetJsonRequestBody(&request.Body)
	if err != nil {
		return nil, err
	}
	req := httpclient.Request{Method: "PUT", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, AuthType: auth.TokenQiniu, Credentials: request.credentials, RequestBody: body}
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
