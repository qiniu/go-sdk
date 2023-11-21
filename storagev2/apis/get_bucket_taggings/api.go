// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 查询指定的存储空间已设置的标签信息
package get_bucket_taggings

import (
	"context"
	"encoding/json"
	auth "github.com/qiniu/go-sdk/v7/auth"
	credentials "github.com/qiniu/go-sdk/v7/storagev2/credentials"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"net/url"
	"strings"
)

// 调用 API 所用的 URL 查询参数
type RequestQuery struct {
	fieldBucketName string // 空间名称
}

func (query *RequestQuery) GetBucketName() string {
	return query.fieldBucketName
}
func (query *RequestQuery) SetBucketName(value string) *RequestQuery {
	query.fieldBucketName = value
	return query
}
func (query *RequestQuery) getBucketName() (string, error) {
	return query.fieldBucketName, nil
}
func (query *RequestQuery) build() (url.Values, error) {
	allQuery := make(url.Values)
	if query.fieldBucketName != "" {
		allQuery.Set("bucket", query.fieldBucketName)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "BucketName"}
	}
	return allQuery, nil
}

type innerTagInfo struct {
	Key   string `json:"Key"`   // 标签名称，最大 64 Byte，不能为空且大小写敏感，不能以 kodo 为前缀(预留), 不支持中文字符，可使用的字符有：字母，数字，空格，+ - = . _ : / @
	Value string `json:"Value"` // 标签值，最大 128 Byte，不能为空且大小写敏感，不支持中文字符，可使用的字符有：字母，数字，空格，+ - = . _ : / @
}

// 标签键值对
type TagInfo struct {
	inner innerTagInfo
}

func (j *TagInfo) GetKey() string {
	return j.inner.Key
}
func (j *TagInfo) SetKey(value string) *TagInfo {
	j.inner.Key = value
	return j
}
func (j *TagInfo) GetValue() string {
	return j.inner.Value
}
func (j *TagInfo) SetValue(value string) *TagInfo {
	j.inner.Value = value
	return j
}
func (j *TagInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(&j.inner)
}
func (j *TagInfo) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.inner)
}

//lint:ignore U1000 may not call it
func (j *TagInfo) validate() error {
	if j.inner.Key == "" {
		return errors.MissingRequiredFieldError{Name: "Key"}
	}
	if j.inner.Value == "" {
		return errors.MissingRequiredFieldError{Name: "Value"}
	}
	return nil
}

// 标签列表
type Tags = []TagInfo
type innerTagsInfo struct {
	Tags Tags `json:"Tags"` // 标签列表
}

// 存储空间标签信息
type TagsInfo struct {
	inner innerTagsInfo
}

func (j *TagsInfo) GetTags() Tags {
	return j.inner.Tags
}
func (j *TagsInfo) SetTags(value Tags) *TagsInfo {
	j.inner.Tags = value
	return j
}
func (j *TagsInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(&j.inner)
}
func (j *TagsInfo) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.inner)
}

//lint:ignore U1000 may not call it
func (j *TagsInfo) validate() error {
	if len(j.inner.Tags) > 0 {
		return errors.MissingRequiredFieldError{Name: "Tags"}
	}
	for _, value := range j.inner.Tags {
		if err := value.validate(); err != nil {
			return err
		}
	}
	return nil
}

// 获取 API 所用的响应体参数
type ResponseBody = TagsInfo

// 调用 API 所用的请求
type Request struct {
	overwrittenBucketHosts region.EndpointsProvider
	overwrittenBucketName  string
	Query                  RequestQuery
	credentials            credentials.CredentialsProvider
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
	if bucketName, err := request.Query.getBucketName(); err != nil || bucketName != "" {
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
	pathSegments = append(pathSegments, "bucketTagging")
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	if query, err := request.Query.build(); err != nil {
		return nil, err
	} else {
		rawQuery += query.Encode()
	}
	req := httpclient.Request{Method: "GET", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, AuthType: auth.TokenQiniu, Credentials: request.credentials}
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
	return &Response{Body: respBody}, nil
}

// 获取 API 所用的响应
type Response struct {
	Body ResponseBody
}
