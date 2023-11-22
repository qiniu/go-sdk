// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 删除空间规则
package delete_bucket_rules

import (
	"context"
	auth "github.com/qiniu/go-sdk/v7/auth"
	credentials "github.com/qiniu/go-sdk/v7/storagev2/credentials"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"net/url"
	"strings"
)

type RequestBody struct {
	fieldBucket string // 空间名称
	fieldName   string // 要删除的规则名称
}

func (form *RequestBody) GetBucket() string {
	return form.fieldBucket
}
func (form *RequestBody) SetBucket(value string) *RequestBody {
	form.fieldBucket = value
	return form
}
func (form *RequestBody) GetName() string {
	return form.fieldName
}
func (form *RequestBody) SetName(value string) *RequestBody {
	form.fieldName = value
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
	if form.fieldName != "" {
		formValues.Set("name", form.fieldName)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Name"}
	}
	return formValues, nil
}
func (request *Request) GetBucket() string {
	return request.Body.GetBucket()
}
func (request *Request) SetBucket(value string) *Request {
	request.Body.SetBucket(value)
	return request
}
func (request *Request) GetName() string {
	return request.Body.GetName()
}
func (request *Request) SetName(value string) *Request {
	request.Body.SetName(value)
	return request
}

// 调用 API 所用的请求
type Request struct {
	overwrittenBucketHosts region.EndpointsProvider
	overwrittenBucketName  string
	credentials            credentials.CredentialsProvider
	Body                   RequestBody
}

func (request *Request) OverwriteBucketHosts(bucketHosts region.EndpointsProvider) *Request {
	request.overwrittenBucketHosts = bucketHosts
	return request
}
func (request *Request) OverwriteBucketName(bucketName string) *Request {
	request.overwrittenBucketName = bucketName
	return request
}
func (request *Request) SetCredentials(credentials credentials.CredentialsProvider) *Request {
	request.credentials = credentials
	return request
}
func (request *Request) getBucketName(ctx context.Context) (string, error) {
	if request.overwrittenBucketName != "" {
		return request.overwrittenBucketName, nil
	}
	if bucketName, err := request.Body.getBucketName(); err != nil || bucketName != "" {
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
func (request *Request) Send(ctx context.Context, options *httpclient.HttpClientOptions) (*Response, error) {
	client := httpclient.NewHttpClient(options)
	serviceNames := []region.ServiceName{region.ServiceBucket}
	var pathSegments []string
	pathSegments = append(pathSegments, "rules", "delete")
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	body, err := request.Body.build()
	if err != nil {
		return nil, err
	}
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, AuthType: auth.TokenQiniu, Credentials: request.credentials, RequestBody: httpclient.GetFormRequestBody(body)}
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
