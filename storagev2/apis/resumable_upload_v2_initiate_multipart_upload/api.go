// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 使用 Multipart Upload 方式上传数据前，必须先调用 API 来获取一个全局唯一的 UploadId，后续的块数据通过 uploadPart API 上传，整个文件完成 completeMultipartUpload API，已经上传块的删除 abortMultipartUpload API 都依赖该 UploadId
package resumable_upload_v2_initiate_multipart_upload

import (
	"context"
	"encoding/base64"
	"encoding/json"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	uptoken "github.com/qiniu/go-sdk/v7/storagev2/uptoken"
	"strings"
)

// 调用 API 所用的路径参数
type RequestPath struct {
	fieldBucketName string
	fieldObjectName string
}

// 存储空间名称
func (pp *RequestPath) GetBucketName() string {
	return pp.fieldBucketName
}

// 存储空间名称
func (pp *RequestPath) SetBucketName(value string) *RequestPath {
	pp.fieldBucketName = value
	return pp
}

// 对象名称
func (pp *RequestPath) GetObjectName() string {
	return pp.fieldObjectName
}

// 对象名称
func (pp *RequestPath) SetObjectName(value string) *RequestPath {
	pp.fieldObjectName = value
	return pp
}
func (path *RequestPath) build() ([]string, error) {
	var allSegments []string
	if path.fieldBucketName != "" {
		allSegments = append(allSegments, path.fieldBucketName)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "BucketName"}
	}
	if path.fieldObjectName != "" {
		allSegments = append(allSegments, "objects", base64.URLEncoding.EncodeToString([]byte(path.fieldObjectName)))
	} else {
		allSegments = append(allSegments, "objects", "~")
	}
	return allSegments, nil
}

// 存储空间名称
func (request *Request) GetBucketName() string {
	return request.path.GetBucketName()
}

// 存储空间名称
func (request *Request) SetBucketName(value string) *Request {
	request.path.SetBucketName(value)
	return request
}

// 对象名称
func (request *Request) GetObjectName() string {
	return request.path.GetObjectName()
}

// 对象名称
func (request *Request) SetObjectName(value string) *Request {
	request.path.SetObjectName(value)
	return request
}

type innerNewMultipartUpload struct {
	UploadId  string `json:"uploadId"` // 初始化文件生成的 id
	ExpiredAt int64  `json:"expireAt"` // UploadId 的过期时间 UNIX 时间戳，过期之后 UploadId 不可用
}

// 返回本次 MultipartUpload 相关信息
type NewMultipartUpload struct {
	inner innerNewMultipartUpload
}

// 初始化文件生成的 id
func (j *NewMultipartUpload) GetUploadId() string {
	return j.inner.UploadId
}

// 初始化文件生成的 id
func (j *NewMultipartUpload) SetUploadId(value string) *NewMultipartUpload {
	j.inner.UploadId = value
	return j
}

// UploadId 的过期时间 UNIX 时间戳，过期之后 UploadId 不可用
func (j *NewMultipartUpload) GetExpiredAt() int64 {
	return j.inner.ExpiredAt
}

// UploadId 的过期时间 UNIX 时间戳，过期之后 UploadId 不可用
func (j *NewMultipartUpload) SetExpiredAt(value int64) *NewMultipartUpload {
	j.inner.ExpiredAt = value
	return j
}
func (j *NewMultipartUpload) MarshalJSON() ([]byte, error) {
	return json.Marshal(&j.inner)
}
func (j *NewMultipartUpload) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.inner)
}

//lint:ignore U1000 may not call it
func (j *NewMultipartUpload) validate() error {
	if j.inner.UploadId == "" {
		return errors.MissingRequiredFieldError{Name: "UploadId"}
	}
	if j.inner.ExpiredAt == 0 {
		return errors.MissingRequiredFieldError{Name: "ExpiredAt"}
	}
	return nil
}

// 获取 API 所用的响应体参数
type ResponseBody = NewMultipartUpload

// 初始化文件生成的 id
func (request *Response) GetUploadId() string {
	return request.body.GetUploadId()
}

// 初始化文件生成的 id
func (request *Response) SetUploadId(value string) *Response {
	request.body.SetUploadId(value)
	return request
}

// UploadId 的过期时间 UNIX 时间戳，过期之后 UploadId 不可用
func (request *Response) GetExpiredAt() int64 {
	return request.body.GetExpiredAt()
}

// UploadId 的过期时间 UNIX 时间戳，过期之后 UploadId 不可用
func (request *Response) SetExpiredAt(value int64) *Response {
	request.body.SetExpiredAt(value)
	return request
}

// 调用 API 所用的请求
type Request struct {
	overwrittenBucketHosts region.EndpointsProvider
	overwrittenBucketName  string
	path                   RequestPath
	upToken                uptoken.Provider
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

// 设置上传凭证
func (request *Request) SetUpToken(upToken uptoken.Provider) *Request {
	request.upToken = upToken
	return request
}
func (request *Request) getBucketName(ctx context.Context) (string, error) {
	if request.overwrittenBucketName != "" {
		return request.overwrittenBucketName, nil
	}
	if request.upToken != nil {
		if putPolicy, err := request.upToken.RetrievePutPolicy(ctx); err != nil {
			return "", err
		} else {
			return putPolicy.GetBucketName()
		}
	}
	return "", nil
}
func (request *Request) getAccessKey(ctx context.Context) (string, error) {
	if request.upToken != nil {
		return request.upToken.RetrieveAccessKey(ctx)
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
	serviceNames := []region.ServiceName{region.ServiceUp}
	var pathSegments []string
	pathSegments = append(pathSegments, "buckets")
	if segments, err := request.path.build(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	pathSegments = append(pathSegments, "uploads")
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, UpToken: request.upToken}
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
