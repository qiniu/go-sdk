// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 仅获取对象的元信息，不返回对象的内容
package stat_object

import (
	"context"
	"encoding/base64"
	"encoding/json"
	auth "github.com/qiniu/go-sdk/v7/auth"
	credentials "github.com/qiniu/go-sdk/v7/storagev2/credentials"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"net/url"
	"strconv"
	"strings"
)

type RequestPath struct {
	fieldEntry string
}

func (pp *RequestPath) GetEntry() string {
	return pp.fieldEntry
}
func (pp *RequestPath) SetEntry(value string) *RequestPath {
	pp.fieldEntry = value
	return pp
}
func (pp *RequestPath) getBucketName() (string, error) {
	return strings.SplitN(pp.fieldEntry, ":", 2)[0], nil
}
func (path *RequestPath) build() ([]string, error) {
	var allSegments []string
	if path.fieldEntry != "" {
		allSegments = append(allSegments, base64.URLEncoding.EncodeToString([]byte(path.fieldEntry)))
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Entry"}
	}
	return allSegments, nil
}

// 调用 API 所用的 URL 查询参数
type RequestQuery struct {
	fieldNeedParts bool // 如果文件是通过分片上传的，是否返回对应的分片信息
}

func (query *RequestQuery) GetNeedParts() bool {
	return query.fieldNeedParts
}
func (query *RequestQuery) SetNeedParts(value bool) *RequestQuery {
	query.fieldNeedParts = value
	return query
}
func (query *RequestQuery) build() (url.Values, error) {
	allQuery := make(url.Values)
	if query.fieldNeedParts {
		allQuery.Set("needparts", strconv.FormatBool(query.fieldNeedParts))
	}
	return allQuery, nil
}

// 每个分片的大小
type PartSizes = []int64

// 每个分片的大小，如没有指定 need_parts 参数则不返回
type Parts = PartSizes
type innerObjectMetadata struct {
	Size                        int64             `json:"fsize"`                             // 对象大小，单位为字节
	Hash                        string            `json:"hash"`                              // 对象哈希值
	MimeType                    string            `json:"mimeType"`                          // 对象 MIME 类型
	Type                        int64             `json:"type"`                              // 对象存储类型，`0` 表示普通存储，`1` 表示低频存储，`2` 表示归档存储
	PutTime                     int64             `json:"putTime"`                           // 文件上传时间，UNIX 时间戳格式，单位为 100 纳秒
	EndUser                     string            `json:"endUser,omitempty"`                 // 资源内容的唯一属主标识
	RestoringStatus             int64             `json:"restoreStatus,omitempty"`           // 归档存储文件的解冻状态，`2` 表示解冻完成，`1` 表示解冻中；归档文件冻结时，不返回该字段
	Status                      int64             `json:"status,omitempty"`                  // 文件状态。`1` 表示禁用；只有禁用状态的文件才会返回该字段
	Md5                         string            `json:"md5,omitempty"`                     // 对象 MD5 值，只有通过直传文件和追加文件 API 上传的文件，服务端确保有该字段返回
	ExpirationTime              int64             `json:"expiration,omitempty"`              // 文件过期删除日期，UNIX 时间戳格式，文件在设置过期时间后才会返回该字段
	TransitionToIaTime          int64             `json:"transitionToIA,omitempty"`          // 文件生命周期中转为低频存储的日期，UNIX 时间戳格式，文件在设置转低频后才会返回该字段
	TransitionToArchiveTime     int64             `json:"transitionToARCHIVE,omitempty"`     // 文件生命周期中转为归档存储的日期，UNIX 时间戳格式，文件在设置转归档后才会返回该字段
	TransitionToDeepArchiveTime int64             `json:"transitionToDeepArchive,omitempty"` // 文件生命周期中转为深度归档存储的日期，UNIX 时间戳格式，文件在设置转归档后才会返回该字段
	Metadata                    map[string]string `json:"x-qn-meta,omitempty"`               // 对象存储元信息
	Parts                       PartSizes         `json:"parts,omitempty"`                   // 每个分片的大小，如没有指定 need_parts 参数则不返回
}

// 文件元信息
type ObjectMetadata struct {
	inner innerObjectMetadata
}

func (j *ObjectMetadata) GetSize() int64 {
	return j.inner.Size
}
func (j *ObjectMetadata) SetSize(value int64) *ObjectMetadata {
	j.inner.Size = value
	return j
}
func (j *ObjectMetadata) GetHash() string {
	return j.inner.Hash
}
func (j *ObjectMetadata) SetHash(value string) *ObjectMetadata {
	j.inner.Hash = value
	return j
}
func (j *ObjectMetadata) GetMimeType() string {
	return j.inner.MimeType
}
func (j *ObjectMetadata) SetMimeType(value string) *ObjectMetadata {
	j.inner.MimeType = value
	return j
}
func (j *ObjectMetadata) GetType() int64 {
	return j.inner.Type
}
func (j *ObjectMetadata) SetType(value int64) *ObjectMetadata {
	j.inner.Type = value
	return j
}
func (j *ObjectMetadata) GetPutTime() int64 {
	return j.inner.PutTime
}
func (j *ObjectMetadata) SetPutTime(value int64) *ObjectMetadata {
	j.inner.PutTime = value
	return j
}
func (j *ObjectMetadata) GetEndUser() string {
	return j.inner.EndUser
}
func (j *ObjectMetadata) SetEndUser(value string) *ObjectMetadata {
	j.inner.EndUser = value
	return j
}
func (j *ObjectMetadata) GetRestoringStatus() int64 {
	return j.inner.RestoringStatus
}
func (j *ObjectMetadata) SetRestoringStatus(value int64) *ObjectMetadata {
	j.inner.RestoringStatus = value
	return j
}
func (j *ObjectMetadata) GetStatus() int64 {
	return j.inner.Status
}
func (j *ObjectMetadata) SetStatus(value int64) *ObjectMetadata {
	j.inner.Status = value
	return j
}
func (j *ObjectMetadata) GetMd5() string {
	return j.inner.Md5
}
func (j *ObjectMetadata) SetMd5(value string) *ObjectMetadata {
	j.inner.Md5 = value
	return j
}
func (j *ObjectMetadata) GetExpirationTime() int64 {
	return j.inner.ExpirationTime
}
func (j *ObjectMetadata) SetExpirationTime(value int64) *ObjectMetadata {
	j.inner.ExpirationTime = value
	return j
}
func (j *ObjectMetadata) GetTransitionToIaTime() int64 {
	return j.inner.TransitionToIaTime
}
func (j *ObjectMetadata) SetTransitionToIaTime(value int64) *ObjectMetadata {
	j.inner.TransitionToIaTime = value
	return j
}
func (j *ObjectMetadata) GetTransitionToArchiveTime() int64 {
	return j.inner.TransitionToArchiveTime
}
func (j *ObjectMetadata) SetTransitionToArchiveTime(value int64) *ObjectMetadata {
	j.inner.TransitionToArchiveTime = value
	return j
}
func (j *ObjectMetadata) GetTransitionToDeepArchiveTime() int64 {
	return j.inner.TransitionToDeepArchiveTime
}
func (j *ObjectMetadata) SetTransitionToDeepArchiveTime(value int64) *ObjectMetadata {
	j.inner.TransitionToDeepArchiveTime = value
	return j
}
func (j *ObjectMetadata) GetMetadata() map[string]string {
	return j.inner.Metadata
}
func (j *ObjectMetadata) SetMetadata(value map[string]string) *ObjectMetadata {
	j.inner.Metadata = value
	return j
}
func (j *ObjectMetadata) GetParts() PartSizes {
	return j.inner.Parts
}
func (j *ObjectMetadata) SetParts(value PartSizes) *ObjectMetadata {
	j.inner.Parts = value
	return j
}
func (j *ObjectMetadata) MarshalJSON() ([]byte, error) {
	return json.Marshal(&j.inner)
}
func (j *ObjectMetadata) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.inner)
}

//lint:ignore U1000 may not call it
func (j *ObjectMetadata) validate() error {
	if j.inner.Size == 0 {
		return errors.MissingRequiredFieldError{Name: "Size"}
	}
	if j.inner.Hash == "" {
		return errors.MissingRequiredFieldError{Name: "Hash"}
	}
	if j.inner.MimeType == "" {
		return errors.MissingRequiredFieldError{Name: "MimeType"}
	}
	if j.inner.Type == 0 {
		return errors.MissingRequiredFieldError{Name: "Type"}
	}
	if j.inner.PutTime == 0 {
		return errors.MissingRequiredFieldError{Name: "PutTime"}
	}
	return nil
}

// 获取 API 所用的响应体参数
type ResponseBody = ObjectMetadata

// 调用 API 所用的请求
type Request struct {
	overwrittenBucketHosts region.EndpointsProvider
	overwrittenBucketName  string
	Path                   RequestPath
	Query                  RequestQuery
	credentials            credentials.CredentialsProvider
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
	if bucketName, err := request.Path.getBucketName(); err != nil || bucketName != "" {
		return bucketName, err
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
	pathSegments = append(pathSegments, "stat")
	if segments, err := request.Path.build(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	query, err := request.Query.build()
	if err != nil {
		return nil, err
	}
	req := httpclient.Request{Method: "GET", ServiceNames: serviceNames, Path: path, RawQuery: query.Encode(), AuthType: auth.TokenQiniu, Credentials: request.credentials}
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
