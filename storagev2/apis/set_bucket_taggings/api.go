// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 设置存储空间的标签列表，包括新增和修改
package set_bucket_taggings

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
	fieldBucket string // 空间名称
}

func (query *RequestQuery) GetBucket() string {
	return query.fieldBucket
}
func (query *RequestQuery) SetBucket(value string) *RequestQuery {
	query.fieldBucket = value
	return query
}
func (query *RequestQuery) getBucketName() (string, error) {
	return query.fieldBucket, nil
}
func (query *RequestQuery) build() (url.Values, error) {
	allQuery := make(url.Values)
	if query.fieldBucket != "" {
		allQuery.Set("bucket", query.fieldBucket)
	}
	return allQuery, nil
}

type innerTagInfo struct {
	Key   string `json:"Key,omitempty"`   // 标签名称，最大 64 Byte，不能为空且大小写敏感，不能以 kodo 为前缀(预留), 不支持中文字符，可使用的字符有：字母，数字，空格，+ - = . _ : / @
	Value string `json:"Value,omitempty"` // 标签值，最大 128 Byte，不能为空且大小写敏感，不支持中文字符，可使用的字符有：字母，数字，空格，+ - = . _ : / @
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
	Tags Tags `json:"Tags,omitempty"` // 标签列表
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

// 调用 API 所用的请求体
type RequestBody = TagsInfo

// 调用 API 所用的请求
type Request struct {
	BucketHosts region.EndpointsProvider
	Query       RequestQuery
	Credentials credentials.CredentialsProvider
	Body        RequestBody
}

func (request Request) getBucketName(ctx context.Context) (string, error) {
	if bucketName, err := request.Query.getBucketName(); err != nil || bucketName != "" {
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
	pathSegments = append(pathSegments, "bucketTagging")
	path := "/" + strings.Join(pathSegments, "/")
	query, err := request.Query.build()
	if err != nil {
		return nil, err
	}
	if err := request.Body.validate(); err != nil {
		return nil, err
	}
	body, err := httpclient.GetJsonRequestBody(&request.Body)
	if err != nil {
		return nil, err
	}
	req := httpclient.Request{Method: "PUT", ServiceNames: serviceNames, Path: path, RawQuery: query.Encode(), AuthType: auth.TokenQiniu, Credentials: request.Credentials, RequestBody: body}
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
