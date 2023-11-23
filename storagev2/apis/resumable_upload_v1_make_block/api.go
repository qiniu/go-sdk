// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 为后续分片上传创建一个新的块，同时上传第一片数据
package resumable_upload_v1_make_block

import (
	"context"
	"encoding/json"
	io "github.com/qiniu/go-sdk/v7/internal/io"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	uptoken "github.com/qiniu/go-sdk/v7/storagev2/uptoken"
	"strconv"
	"strings"
)

// 调用 API 所用的路径参数
type RequestPath struct {
	fieldBlockSize int64
}

// 块大小，单位为字节，每块均为 4 MB，最后一块大小不超过 4 MB
func (pp *RequestPath) GetBlockSize() int64 {
	return pp.fieldBlockSize
}

// 块大小，单位为字节，每块均为 4 MB，最后一块大小不超过 4 MB
func (pp *RequestPath) SetBlockSize(value int64) *RequestPath {
	pp.fieldBlockSize = value
	return pp
}
func (path *RequestPath) build() ([]string, error) {
	var allSegments []string
	if path.fieldBlockSize != 0 {
		allSegments = append(allSegments, strconv.FormatInt(path.fieldBlockSize, 10))
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "BlockSize"}
	}
	return allSegments, nil
}

// 块大小，单位为字节，每块均为 4 MB，最后一块大小不超过 4 MB
func (request *Request) GetBlockSize() int64 {
	return request.path.GetBlockSize()
}

// 块大小，单位为字节，每块均为 4 MB，最后一块大小不超过 4 MB
func (request *Request) SetBlockSize(value int64) *Request {
	request.path.SetBlockSize(value)
	return request
}

type innerNewBlockInfo struct {
	Ctx       string `json:"ctx"`        // 本次上传成功后的块级上传控制信息，用于后续上传片（bput）及创建文件（mkfile）
	Checksum  string `json:"checksum"`   // 上传块 SHA1 值，使用 URL 安全的 Base64 编码
	Crc32     int64  `json:"crc32"`      // 上传块 CRC32 值，客户可通过此字段对上传块的完整性进行校验
	Offset    int64  `json:"offset"`     // 下一个上传块在切割块中的偏移
	Host      string `json:"host"`       // 后续上传接收地址
	ExpiredAt int64  `json:"expired_at"` // `ctx` 过期时间
}

// 返回下一片数据的上传信息
type NewBlockInfo struct {
	inner innerNewBlockInfo
}

// 本次上传成功后的块级上传控制信息，用于后续上传片（bput）及创建文件（mkfile）
func (j *NewBlockInfo) GetCtx() string {
	return j.inner.Ctx
}

// 本次上传成功后的块级上传控制信息，用于后续上传片（bput）及创建文件（mkfile）
func (j *NewBlockInfo) SetCtx(value string) *NewBlockInfo {
	j.inner.Ctx = value
	return j
}

// 上传块 SHA1 值，使用 URL 安全的 Base64 编码
func (j *NewBlockInfo) GetChecksum() string {
	return j.inner.Checksum
}

// 上传块 SHA1 值，使用 URL 安全的 Base64 编码
func (j *NewBlockInfo) SetChecksum(value string) *NewBlockInfo {
	j.inner.Checksum = value
	return j
}

// 上传块 CRC32 值，客户可通过此字段对上传块的完整性进行校验
func (j *NewBlockInfo) GetCrc32() int64 {
	return j.inner.Crc32
}

// 上传块 CRC32 值，客户可通过此字段对上传块的完整性进行校验
func (j *NewBlockInfo) SetCrc32(value int64) *NewBlockInfo {
	j.inner.Crc32 = value
	return j
}

// 下一个上传块在切割块中的偏移
func (j *NewBlockInfo) GetOffset() int64 {
	return j.inner.Offset
}

// 下一个上传块在切割块中的偏移
func (j *NewBlockInfo) SetOffset(value int64) *NewBlockInfo {
	j.inner.Offset = value
	return j
}

// 后续上传接收地址
func (j *NewBlockInfo) GetHost() string {
	return j.inner.Host
}

// 后续上传接收地址
func (j *NewBlockInfo) SetHost(value string) *NewBlockInfo {
	j.inner.Host = value
	return j
}

// `ctx` 过期时间
func (j *NewBlockInfo) GetExpiredAt() int64 {
	return j.inner.ExpiredAt
}

// `ctx` 过期时间
func (j *NewBlockInfo) SetExpiredAt(value int64) *NewBlockInfo {
	j.inner.ExpiredAt = value
	return j
}
func (j *NewBlockInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(&j.inner)
}
func (j *NewBlockInfo) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.inner)
}

//lint:ignore U1000 may not call it
func (j *NewBlockInfo) validate() error {
	if j.inner.Ctx == "" {
		return errors.MissingRequiredFieldError{Name: "Ctx"}
	}
	if j.inner.Checksum == "" {
		return errors.MissingRequiredFieldError{Name: "Checksum"}
	}
	if j.inner.Crc32 == 0 {
		return errors.MissingRequiredFieldError{Name: "Crc32"}
	}
	if j.inner.Offset == 0 {
		return errors.MissingRequiredFieldError{Name: "Offset"}
	}
	if j.inner.Host == "" {
		return errors.MissingRequiredFieldError{Name: "Host"}
	}
	if j.inner.ExpiredAt == 0 {
		return errors.MissingRequiredFieldError{Name: "ExpiredAt"}
	}
	return nil
}

// 获取 API 所用的响应体参数
type ResponseBody = NewBlockInfo

// 本次上传成功后的块级上传控制信息，用于后续上传片（bput）及创建文件（mkfile）
func (request *Response) GetCtx() string {
	return request.body.GetCtx()
}

// 本次上传成功后的块级上传控制信息，用于后续上传片（bput）及创建文件（mkfile）
func (request *Response) SetCtx(value string) *Response {
	request.body.SetCtx(value)
	return request
}

// 上传块 SHA1 值，使用 URL 安全的 Base64 编码
func (request *Response) GetChecksum() string {
	return request.body.GetChecksum()
}

// 上传块 SHA1 值，使用 URL 安全的 Base64 编码
func (request *Response) SetChecksum(value string) *Response {
	request.body.SetChecksum(value)
	return request
}

// 上传块 CRC32 值，客户可通过此字段对上传块的完整性进行校验
func (request *Response) GetCrc32() int64 {
	return request.body.GetCrc32()
}

// 上传块 CRC32 值，客户可通过此字段对上传块的完整性进行校验
func (request *Response) SetCrc32(value int64) *Response {
	request.body.SetCrc32(value)
	return request
}

// 下一个上传块在切割块中的偏移
func (request *Response) GetOffset() int64 {
	return request.body.GetOffset()
}

// 下一个上传块在切割块中的偏移
func (request *Response) SetOffset(value int64) *Response {
	request.body.SetOffset(value)
	return request
}

// 后续上传接收地址
func (request *Response) GetHost() string {
	return request.body.GetHost()
}

// 后续上传接收地址
func (request *Response) SetHost(value string) *Response {
	request.body.SetHost(value)
	return request
}

// `ctx` 过期时间
func (request *Response) GetExpiredAt() int64 {
	return request.body.GetExpiredAt()
}

// `ctx` 过期时间
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
	body                   io.ReadSeekCloser
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

// 获取请求体
func (request *Request) GetBody() io.ReadSeekCloser {
	return request.body
}

// 设置请求路径
func (request *Request) SetPath(path RequestPath) *Request {
	request.path = path
	return request
}

// 设置请求体
func (request *Request) SetBody(body io.ReadSeekCloser) *Request {
	request.body = body
	return request
}

// 发送请求
func (request *Request) Send(ctx context.Context, options *httpclient.HttpClientOptions) (*Response, error) {
	client := httpclient.NewHttpClient(options)
	serviceNames := []region.ServiceName{region.ServiceUp}
	var pathSegments []string
	pathSegments = append(pathSegments, "mkblk")
	if segments, err := request.path.build(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, UpToken: request.upToken, RequestBody: httpclient.GetRequestBodyFromReadSeekCloser(request.body)}
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
			queryerOptions := region.BucketRegionsQueryerOptions{UseInsecureProtocol: options.UseInsecureProtocol, HostFreezeDuration: options.HostFreezeDuration, Client: options.Client}
			if hostRetryConfig := options.HostRetryConfig; hostRetryConfig != nil {
				queryerOptions.RetryMax = hostRetryConfig.RetryMax
			}
			if queryer, err = region.NewBucketRegionsQueryer(bucketHosts, &queryerOptions); err != nil {
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
