// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 批量操作意指在单一请求中执行多次（最大限制1000次） 查询元信息、修改元信息、移动、复制、删除、修改状态、修改存储类型、修改生命周期和解冻操作，极大提高对象管理效率。其中，解冻操作仅针对归档存储文件有效
package batch_ops

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

type RequestBody struct {
	fieldOperations []string // 单一对象管理指令
}

func (form *RequestBody) GetOperations() []string {
	return form.fieldOperations
}
func (form *RequestBody) SetOperations(value []string) *RequestBody {
	form.fieldOperations = value
	return form
}
func (form *RequestBody) build() (url.Values, error) {
	formValues := make(url.Values)
	if len(form.fieldOperations) > 0 {
		for _, value := range form.fieldOperations {
			formValues.Add("op", value)
		}
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Operations"}
	}
	return formValues, nil
}

type innerOperationResponseData struct {
	Error                       string `json:"error,omitempty"`                   // 管理指令的错误信息，仅在发生错误时才返回
	Size                        int64  `json:"fsize,omitempty"`                   // 对象大小，单位为字节，仅对 stat 指令才有效
	Hash                        string `json:"hash,omitempty"`                    // 对象哈希值，仅对 stat 指令才有效
	MimeType                    string `json:"mimeType,omitempty"`                // 对象 MIME 类型，仅对 stat 指令才有效
	Type                        int64  `json:"type,omitempty"`                    // 对象存储类型，`0` 表示普通存储，`1` 表示低频存储，`2` 表示归档存储，仅对 stat 指令才有效
	PutTime                     int64  `json:"putTime,omitempty"`                 // 文件上传时间，UNIX 时间戳格式，单位为 100 纳秒，仅对 stat 指令才有效
	EndUser                     string `json:"endUser,omitempty"`                 // 资源内容的唯一属主标识
	RestoringStatus             int64  `json:"restoreStatus,omitempty"`           // 归档存储文件的解冻状态，`2` 表示解冻完成，`1` 表示解冻中；归档文件冻结时，不返回该字段，仅对 stat 指令才有效
	Status                      int64  `json:"status,omitempty"`                  // 文件状态。`1` 表示禁用；只有禁用状态的文件才会返回该字段，仅对 stat 指令才有效
	Md5                         string `json:"md5,omitempty"`                     // 对象 MD5 值，只有通过直传文件和追加文件 API 上传的文件，服务端确保有该字段返回，仅对 stat 指令才有效
	ExpirationTime              int64  `json:"expiration,omitempty"`              // 文件过期删除日期，UNIX 时间戳格式，文件在设置过期时间后才会返回该字段，仅对 stat 指令才有效
	TransitionToIaTime          int64  `json:"transitionToIA,omitempty"`          // 文件生命周期中转为低频存储的日期，UNIX 时间戳格式，文件在设置转低频后才会返回该字段，仅对 stat 指令才有效
	TransitionToArchiveTime     int64  `json:"transitionToARCHIVE,omitempty"`     // 文件生命周期中转为归档存储的日期，UNIX 时间戳格式，文件在设置转归档后才会返回该字段，仅对 stat 指令才有效
	TransitionToDeepArchiveTime int64  `json:"transitionToDeepArchive,omitempty"` // 文件生命周期中转为深度归档存储的日期，UNIX 时间戳格式，文件在设置转归档后才会返回该字段，仅对 stat 指令才有效
}

// 管理指令的响应数据
type OperationResponseData struct {
	inner innerOperationResponseData
}

func (j *OperationResponseData) GetError() string {
	return j.inner.Error
}
func (j *OperationResponseData) SetError(value string) *OperationResponseData {
	j.inner.Error = value
	return j
}
func (j *OperationResponseData) GetSize() int64 {
	return j.inner.Size
}
func (j *OperationResponseData) SetSize(value int64) *OperationResponseData {
	j.inner.Size = value
	return j
}
func (j *OperationResponseData) GetHash() string {
	return j.inner.Hash
}
func (j *OperationResponseData) SetHash(value string) *OperationResponseData {
	j.inner.Hash = value
	return j
}
func (j *OperationResponseData) GetMimeType() string {
	return j.inner.MimeType
}
func (j *OperationResponseData) SetMimeType(value string) *OperationResponseData {
	j.inner.MimeType = value
	return j
}
func (j *OperationResponseData) GetType() int64 {
	return j.inner.Type
}
func (j *OperationResponseData) SetType(value int64) *OperationResponseData {
	j.inner.Type = value
	return j
}
func (j *OperationResponseData) GetPutTime() int64 {
	return j.inner.PutTime
}
func (j *OperationResponseData) SetPutTime(value int64) *OperationResponseData {
	j.inner.PutTime = value
	return j
}
func (j *OperationResponseData) GetEndUser() string {
	return j.inner.EndUser
}
func (j *OperationResponseData) SetEndUser(value string) *OperationResponseData {
	j.inner.EndUser = value
	return j
}
func (j *OperationResponseData) GetRestoringStatus() int64 {
	return j.inner.RestoringStatus
}
func (j *OperationResponseData) SetRestoringStatus(value int64) *OperationResponseData {
	j.inner.RestoringStatus = value
	return j
}
func (j *OperationResponseData) GetStatus() int64 {
	return j.inner.Status
}
func (j *OperationResponseData) SetStatus(value int64) *OperationResponseData {
	j.inner.Status = value
	return j
}
func (j *OperationResponseData) GetMd5() string {
	return j.inner.Md5
}
func (j *OperationResponseData) SetMd5(value string) *OperationResponseData {
	j.inner.Md5 = value
	return j
}
func (j *OperationResponseData) GetExpirationTime() int64 {
	return j.inner.ExpirationTime
}
func (j *OperationResponseData) SetExpirationTime(value int64) *OperationResponseData {
	j.inner.ExpirationTime = value
	return j
}
func (j *OperationResponseData) GetTransitionToIaTime() int64 {
	return j.inner.TransitionToIaTime
}
func (j *OperationResponseData) SetTransitionToIaTime(value int64) *OperationResponseData {
	j.inner.TransitionToIaTime = value
	return j
}
func (j *OperationResponseData) GetTransitionToArchiveTime() int64 {
	return j.inner.TransitionToArchiveTime
}
func (j *OperationResponseData) SetTransitionToArchiveTime(value int64) *OperationResponseData {
	j.inner.TransitionToArchiveTime = value
	return j
}
func (j *OperationResponseData) GetTransitionToDeepArchiveTime() int64 {
	return j.inner.TransitionToDeepArchiveTime
}
func (j *OperationResponseData) SetTransitionToDeepArchiveTime(value int64) *OperationResponseData {
	j.inner.TransitionToDeepArchiveTime = value
	return j
}
func (j *OperationResponseData) MarshalJSON() ([]byte, error) {
	return json.Marshal(&j.inner)
}
func (j *OperationResponseData) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.inner)
}

//lint:ignore U1000 may not call it
func (j *OperationResponseData) validate() error {
	return nil
}

// 响应数据
type Data = OperationResponseData
type innerOperationResponse struct {
	Code int64                 `json:"code"`           // 响应状态码
	Data OperationResponseData `json:"data,omitempty"` // 响应数据
}

// 每个管理指令的响应信息
type OperationResponse struct {
	inner innerOperationResponse
}

func (j *OperationResponse) GetCode() int64 {
	return j.inner.Code
}
func (j *OperationResponse) SetCode(value int64) *OperationResponse {
	j.inner.Code = value
	return j
}
func (j *OperationResponse) GetData() OperationResponseData {
	return j.inner.Data
}
func (j *OperationResponse) SetData(value OperationResponseData) *OperationResponse {
	j.inner.Data = value
	return j
}
func (j *OperationResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(&j.inner)
}
func (j *OperationResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.inner)
}

//lint:ignore U1000 may not call it
func (j *OperationResponse) validate() error {
	if j.inner.Code == 0 {
		return errors.MissingRequiredFieldError{Name: "Code"}
	}
	return nil
}

// 所有管理指令的响应信息
type OperationResponses = []OperationResponse

// 获取 API 所用的响应体参数
type ResponseBody = OperationResponses

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
	serviceNames := []region.ServiceName{region.ServiceRs}
	var pathSegments []string
	pathSegments = append(pathSegments, "batch")
	path := "/" + strings.Join(pathSegments, "/")
	body, err := request.Body.build()
	if err != nil {
		return nil, err
	}
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, AuthType: auth.TokenQiniu, Credentials: request.credentials, RequestBody: httpclient.GetFormRequestBody(body)}
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
