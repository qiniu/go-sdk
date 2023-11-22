// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 设置空间的跨域规则
package set_bucket_cors_rules

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

// 指定空间名称
func (pp *RequestPath) GetBucket() string {
	return pp.fieldBucket
}

// 指定空间名称
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

// 指定空间名称
func (request *Request) GetBucket() string {
	return request.path.GetBucket()
}

// 指定空间名称
func (request *Request) SetBucket(value string) *Request {
	request.path.SetBucket(value)
	return request
}

// 允许的域名列表
type AllowedOriginHosts = []string

// 允许的域名。必填；支持通配符 * ；*表示全部匹配；只有第一个 * 生效；需要设置 "Scheme"；大小写敏感
type AllowedOrigin = AllowedOriginHosts

// 允许的方法列表
type AllowedMethods = []string

// 允许的方法。必填；不支持通配符；大小写不敏感；
type AllowedMethod = AllowedMethods

// 允许的 Header 列表
type AllowedHeaders = []string
type AllowedHeader = AllowedHeaders

// 暴露的 Header 列表
type ExposedHeaders = []string

// 选填；不支持通配符；X-Log, X-Reqid 是默认会暴露的两个 header；其他的 header 如果没有设置，则不会暴露；大小写不敏感；
type ExposedHeader = ExposedHeaders
type innerCorsRule struct {
	AllowedOrigin AllowedOriginHosts `json:"allowed_origin"` // 允许的域名。必填；支持通配符 * ；*表示全部匹配；只有第一个 * 生效；需要设置 "Scheme"；大小写敏感
	AllowedMethod AllowedMethods     `json:"allowed_method"` // 允许的方法。必填；不支持通配符；大小写不敏感；
	AllowedHeader AllowedHeaders     `json:"allowed_header,omitempty"`
	ExposedHeader ExposedHeaders     `json:"exposed_header,omitempty"` // 选填；不支持通配符；X-Log, X-Reqid 是默认会暴露的两个 header；其他的 header 如果没有设置，则不会暴露；大小写不敏感；
	MaxAge        int64              `json:"max_age,omitempty"`        // 结果可以缓存的时间。选填；空则不缓存
}

// 跨域规则
type CorsRule struct {
	inner innerCorsRule
}

// 允许的域名。必填；支持通配符 * ；*表示全部匹配；只有第一个 * 生效；需要设置 "Scheme"；大小写敏感
func (j *CorsRule) GetAllowedOrigin() AllowedOriginHosts {
	return j.inner.AllowedOrigin
}

// 允许的域名。必填；支持通配符 * ；*表示全部匹配；只有第一个 * 生效；需要设置 "Scheme"；大小写敏感
func (j *CorsRule) SetAllowedOrigin(value AllowedOriginHosts) *CorsRule {
	j.inner.AllowedOrigin = value
	return j
}

// 允许的方法。必填；不支持通配符；大小写不敏感；
func (j *CorsRule) GetAllowedMethod() AllowedMethods {
	return j.inner.AllowedMethod
}

// 允许的方法。必填；不支持通配符；大小写不敏感；
func (j *CorsRule) SetAllowedMethod(value AllowedMethods) *CorsRule {
	j.inner.AllowedMethod = value
	return j
}
func (j *CorsRule) GetAllowedHeader() AllowedHeaders {
	return j.inner.AllowedHeader
}
func (j *CorsRule) SetAllowedHeader(value AllowedHeaders) *CorsRule {
	j.inner.AllowedHeader = value
	return j
}

// 选填；不支持通配符；X-Log, X-Reqid 是默认会暴露的两个 header；其他的 header 如果没有设置，则不会暴露；大小写不敏感；
func (j *CorsRule) GetExposedHeader() ExposedHeaders {
	return j.inner.ExposedHeader
}

// 选填；不支持通配符；X-Log, X-Reqid 是默认会暴露的两个 header；其他的 header 如果没有设置，则不会暴露；大小写不敏感；
func (j *CorsRule) SetExposedHeader(value ExposedHeaders) *CorsRule {
	j.inner.ExposedHeader = value
	return j
}

// 结果可以缓存的时间。选填；空则不缓存
func (j *CorsRule) GetMaxAge() int64 {
	return j.inner.MaxAge
}

// 结果可以缓存的时间。选填；空则不缓存
func (j *CorsRule) SetMaxAge(value int64) *CorsRule {
	j.inner.MaxAge = value
	return j
}
func (j *CorsRule) MarshalJSON() ([]byte, error) {
	return json.Marshal(&j.inner)
}
func (j *CorsRule) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.inner)
}

//lint:ignore U1000 may not call it
func (j *CorsRule) validate() error {
	if len(j.inner.AllowedOrigin) == 0 {
		return errors.MissingRequiredFieldError{Name: "AllowedOrigin"}
	}
	if len(j.inner.AllowedMethod) == 0 {
		return errors.MissingRequiredFieldError{Name: "AllowedMethod"}
	}
	return nil
}

// 跨域规则列表
type CorsRules = []CorsRule

// 调用 API 所用的请求体
type RequestBody = CorsRules

// 调用 API 所用的请求
type Request struct {
	overwrittenBucketHosts region.EndpointsProvider
	overwrittenBucketName  string
	path                   RequestPath
	credentials            credentials.CredentialsProvider
	body                   RequestBody
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

// 获取请求体
func (request *Request) GetBody() RequestBody {
	return request.body
}

// 设置请求路径
func (request *Request) SetPath(path RequestPath) *Request {
	request.path = path
	return request
}

// 设置请求体
func (request *Request) SetBody(body RequestBody) *Request {
	request.body = body
	return request
}

// 发送请求
func (request *Request) Send(ctx context.Context, options *httpclient.HttpClientOptions) (*Response, error) {
	client := httpclient.NewHttpClient(options)
	serviceNames := []region.ServiceName{region.ServiceBucket}
	var pathSegments []string
	pathSegments = append(pathSegments, "corsRules", "set")
	if segments, err := request.path.build(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	body, err := httpclient.GetJsonRequestBody(&request.body)
	if err != nil {
		return nil, err
	}
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, AuthType: auth.TokenQiniu, Credentials: request.credentials, RequestBody: body}
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
	return &Response{}, resp.Body.Close()
}

// 获取 API 所用的响应
type Response struct{}
