// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 从指定 URL 抓取资源，并将该资源存储到指定空间中。每次只抓取一个文件，抓取时可以指定保存空间名和最终资源名
package async_fetch_object

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

type innerNewFetchTaskParams struct {
	Url              string  `json:"url"`                        // 需要抓取的 URL，支持设置多个用于高可用，以’;'分隔，当指定多个 URL 时可以在前一个 URL 抓取失败时重试下一个
	Bucket           string  `json:"bucket"`                     // 所在区域的存储空间
	Host             string  `json:"host,omitempty"`             // 从指定 URL 下载数据时使用的 Host
	Key              string  `json:"key,omitempty"`              // 对象名称，如果不传，则默认为文件的哈希值
	Md5              string  `json:"md5,omitempty"`              // 文件 MD5，传入以后会在存入存储时对文件做校验，校验失败则不存入指定空间
	Etag             string  `json:"etag,omitempty"`             // 对象内容的 ETag，传入以后会在存入存储时对文件做校验，校验失败则不存入指定空间
	CallbackUrl      string  `json:"callbackurl,omitempty"`      // 回调 URL
	CallbackBody     string  `json:"callbackbody,omitempty"`     // 回调负荷，如果 callback_url 不为空则必须指定
	CallbackBodyType string  `json:"callbackbodytype,omitempty"` // 回调负荷内容类型，默认为 "application/x-www-form-urlencoded"
	CallbackHost     string  `json:"callbackhost,omitempty"`     // 回调时使用的 Host
	FileType         int64   `json:"file_type"`                  // 存储文件类型 `0`: 标准存储(默认)，`1`: 低频存储，`2`: 归档存储
	IgnoreSameKey    float64 `json:"ignore_same_key,omitempty"`  // 如果空间中已经存在同名文件则放弃本次抓取（仅对比对象名称，不校验文件内容）
}

// 要抓取的资源信息
type NewFetchTaskParams struct {
	inner innerNewFetchTaskParams
}

func (j *NewFetchTaskParams) GetUrl() string {
	return j.inner.Url
}
func (j *NewFetchTaskParams) SetUrl(value string) *NewFetchTaskParams {
	j.inner.Url = value
	return j
}
func (j *NewFetchTaskParams) GetBucket() string {
	return j.inner.Bucket
}
func (j *NewFetchTaskParams) SetBucket(value string) *NewFetchTaskParams {
	j.inner.Bucket = value
	return j
}
func (j *NewFetchTaskParams) GetHost() string {
	return j.inner.Host
}
func (j *NewFetchTaskParams) SetHost(value string) *NewFetchTaskParams {
	j.inner.Host = value
	return j
}
func (j *NewFetchTaskParams) GetKey() string {
	return j.inner.Key
}
func (j *NewFetchTaskParams) SetKey(value string) *NewFetchTaskParams {
	j.inner.Key = value
	return j
}
func (j *NewFetchTaskParams) GetMd5() string {
	return j.inner.Md5
}
func (j *NewFetchTaskParams) SetMd5(value string) *NewFetchTaskParams {
	j.inner.Md5 = value
	return j
}
func (j *NewFetchTaskParams) GetEtag() string {
	return j.inner.Etag
}
func (j *NewFetchTaskParams) SetEtag(value string) *NewFetchTaskParams {
	j.inner.Etag = value
	return j
}
func (j *NewFetchTaskParams) GetCallbackUrl() string {
	return j.inner.CallbackUrl
}
func (j *NewFetchTaskParams) SetCallbackUrl(value string) *NewFetchTaskParams {
	j.inner.CallbackUrl = value
	return j
}
func (j *NewFetchTaskParams) GetCallbackBody() string {
	return j.inner.CallbackBody
}
func (j *NewFetchTaskParams) SetCallbackBody(value string) *NewFetchTaskParams {
	j.inner.CallbackBody = value
	return j
}
func (j *NewFetchTaskParams) GetCallbackBodyType() string {
	return j.inner.CallbackBodyType
}
func (j *NewFetchTaskParams) SetCallbackBodyType(value string) *NewFetchTaskParams {
	j.inner.CallbackBodyType = value
	return j
}
func (j *NewFetchTaskParams) GetCallbackHost() string {
	return j.inner.CallbackHost
}
func (j *NewFetchTaskParams) SetCallbackHost(value string) *NewFetchTaskParams {
	j.inner.CallbackHost = value
	return j
}
func (j *NewFetchTaskParams) GetFileType() int64 {
	return j.inner.FileType
}
func (j *NewFetchTaskParams) SetFileType(value int64) *NewFetchTaskParams {
	j.inner.FileType = value
	return j
}
func (j *NewFetchTaskParams) GetIgnoreSameKey() float64 {
	return j.inner.IgnoreSameKey
}
func (j *NewFetchTaskParams) SetIgnoreSameKey(value float64) *NewFetchTaskParams {
	j.inner.IgnoreSameKey = value
	return j
}
func (j *NewFetchTaskParams) MarshalJSON() ([]byte, error) {
	return json.Marshal(&j.inner)
}
func (j *NewFetchTaskParams) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.inner)
}

//lint:ignore U1000 may not call it
func (j *NewFetchTaskParams) validate() error {
	if j.inner.Url == "" {
		return errors.MissingRequiredFieldError{Name: "Url"}
	}
	if j.inner.Bucket == "" {
		return errors.MissingRequiredFieldError{Name: "Bucket"}
	}
	return nil
}

// 调用 API 所用的请求体
type RequestBody = NewFetchTaskParams
type innerNewFetchTaskInfo struct {
	Id               string `json:"id"`   // 异步任务 ID
	QueuedTasksCount int64  `json:"wait"` // 当前任务前面的排队任务数量，`0` 表示当前任务正在进行，`-1` 表示任务已经至少被处理过一次（可能会进入重试逻辑）
}

// 返回的异步任务信息
type NewFetchTaskInfo struct {
	inner innerNewFetchTaskInfo
}

func (j *NewFetchTaskInfo) GetId() string {
	return j.inner.Id
}
func (j *NewFetchTaskInfo) SetId(value string) *NewFetchTaskInfo {
	j.inner.Id = value
	return j
}
func (j *NewFetchTaskInfo) GetQueuedTasksCount() int64 {
	return j.inner.QueuedTasksCount
}
func (j *NewFetchTaskInfo) SetQueuedTasksCount(value int64) *NewFetchTaskInfo {
	j.inner.QueuedTasksCount = value
	return j
}
func (j *NewFetchTaskInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(&j.inner)
}
func (j *NewFetchTaskInfo) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.inner)
}

//lint:ignore U1000 may not call it
func (j *NewFetchTaskInfo) validate() error {
	if j.inner.Id == "" {
		return errors.MissingRequiredFieldError{Name: "Id"}
	}
	if j.inner.QueuedTasksCount == 0 {
		return errors.MissingRequiredFieldError{Name: "QueuedTasksCount"}
	}
	return nil
}

// 获取 API 所用的响应体参数
type ResponseBody = NewFetchTaskInfo

// 调用 API 所用的请求
type Request struct {
	overwrittenBucketHosts region.EndpointsProvider
	overwrittenBucketName  string
	credentials            credentials.CredentialsProvider
	Body                   RequestBody
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
	serviceNames := []region.ServiceName{region.ServiceApi}
	var pathSegments []string
	pathSegments = append(pathSegments, "sisyphus", "fetch")
	path := "/" + strings.Join(pathSegments, "/")
	if err := request.Body.validate(); err != nil {
		return nil, err
	}
	body, err := httpclient.GetJsonRequestBody(&request.Body)
	if err != nil {
		return nil, err
	}
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, AuthType: auth.TokenQiniu, Credentials: request.credentials, RequestBody: body}
	var queryer region.BucketRegionsQueryer
	if client.GetRegions() == nil && client.GetEndpoints() == nil {
		queryer = client.GetBucketQueryer()
		if queryer == nil {
			bucketHosts := httpclient.DefaultBucketHosts()
			var err error
			if request.overwrittenBucketHosts != nil {
				if bucketHosts, err = request.overwrittenBucketHosts.GetEndpoints(ctx); err != nil {
					return nil, err
				}
			}
			if queryer, err = region.NewBucketRegionsQueryer(bucketHosts, nil); err != nil {
				return nil, err
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
