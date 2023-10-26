// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 初始化一个 Multipart Upload 任务之后，可以根据指定的对象名称和 UploadId 来分片上传数据
package resumable_upload_v2_upload_part

import (
	"context"
	"encoding/base64"
	"encoding/json"
	io "github.com/qiniu/go-sdk/v7/internal/io"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	uptoken "github.com/qiniu/go-sdk/v7/storagev2/uptoken"
	"net/http"
	"strconv"
	"strings"
)

type RequestPath struct {
	fieldBucketName string
	fieldObjectName string
	fieldUploadId   string
	fieldPartNumber int64
}

func (pp *RequestPath) GetBucketName() string {
	return pp.fieldBucketName
}
func (pp *RequestPath) SetBucketName(value string) *RequestPath {
	pp.fieldBucketName = value
	return pp
}
func (pp *RequestPath) GetObjectName() string {
	return pp.fieldObjectName
}
func (pp *RequestPath) SetObjectName(value string) *RequestPath {
	pp.fieldObjectName = value
	return pp
}
func (pp *RequestPath) GetUploadId() string {
	return pp.fieldUploadId
}
func (pp *RequestPath) SetUploadId(value string) *RequestPath {
	pp.fieldUploadId = value
	return pp
}
func (pp *RequestPath) GetPartNumber() int64 {
	return pp.fieldPartNumber
}
func (pp *RequestPath) SetPartNumber(value int64) *RequestPath {
	pp.fieldPartNumber = value
	return pp
}
func (path *RequestPath) build() []string {
	var allSegments []string
	if path.fieldBucketName != "" {
		allSegments = append(allSegments, path.fieldBucketName)
	}
	if path.fieldObjectName != "" {
		allSegments = append(allSegments, "objects", base64.URLEncoding.EncodeToString([]byte(path.fieldObjectName)))
	} else {
		allSegments = append(allSegments, "objects", "~")
	}
	if path.fieldUploadId != "" {
		allSegments = append(allSegments, "uploads", path.fieldUploadId)
	}
	if path.fieldPartNumber != 0 {
		allSegments = append(allSegments, strconv.FormatInt(path.fieldPartNumber, 10))
	}
	return allSegments
}

// 调用 API 所用的 HTTP 头参数
type RequestHeaders struct {
	fieldMd5 string // 上传块内容的 md5 值，如果指定服务端会进行校验，不指定不校验
}

func (header *RequestHeaders) GetMd5() string {
	return header.fieldMd5
}
func (header *RequestHeaders) SetMd5(value string) *RequestHeaders {
	header.fieldMd5 = value
	return header
}
func (headers *RequestHeaders) build() http.Header {
	allHeaders := make(http.Header)
	if headers.fieldMd5 != "" {
		allHeaders.Set("Content-MD5", headers.fieldMd5)
	}
	return allHeaders
}

type innerNewPartInfo struct {
	Etag string `json:"etag,omitempty"` // 上传块内容的 etag，用来标识块，completeMultipartUpload API 调用的时候作为参数进行文件合成
	Md5  string `json:"md5,omitempty"`  // 上传块内容的 MD5 值
}

// 返回本次上传的分片相关信息
type NewPartInfo struct {
	inner innerNewPartInfo
}

func (j *NewPartInfo) GetEtag() string {
	return j.inner.Etag
}
func (j *NewPartInfo) SetEtag(value string) *NewPartInfo {
	j.inner.Etag = value
	return j
}
func (j *NewPartInfo) GetMd5() string {
	return j.inner.Md5
}
func (j *NewPartInfo) SetMd5(value string) *NewPartInfo {
	j.inner.Md5 = value
	return j
}
func (j *NewPartInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(&j.inner)
}
func (j *NewPartInfo) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.inner)
}

// 获取 API 所用的响应体参数
type ResponseBody = NewPartInfo

// 调用 API 所用的请求
type Request struct {
	BucketHosts region.EndpointsProvider
	Path        RequestPath
	Headers     RequestHeaders
	UpToken     uptoken.Provider
	Body        io.ReadSeekCloser
}

func (request Request) getBucketName(ctx context.Context) (string, error) {
	if request.UpToken != nil {
		if putPolicy, err := request.UpToken.RetrievePutPolicy(ctx); err != nil {
			return "", err
		} else {
			return putPolicy.GetBucketName()
		}
	}
	return "", nil
}
func (request Request) getAccessKey(ctx context.Context) (string, error) {
	if request.UpToken != nil {
		return request.UpToken.RetrieveAccessKey(ctx)
	}
	return "", nil
}

// 获取 API 所用的响应
type Response struct {
	Body ResponseBody
}

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
	serviceNames := []region.ServiceName{region.ServiceUp}
	var pathSegments []string
	pathSegments = append(pathSegments, "buckets")
	pathSegments = append(pathSegments, request.Path.build()...)
	path := "/" + strings.Join(pathSegments, "/")
	req := httpclient.Request{Method: "PUT", ServiceNames: serviceNames, Path: path, Header: request.Headers.build(), UpToken: request.UpToken, RequestBody: httpclient.GetRequestBodyFromReadSeekCloser(request.Body)}
	var queryer *region.BucketRegionsQueryer
	if client.client.GetRegions() == nil && client.client.GetEndpoints() == nil {
		queryer = client.client.GetBucketQueryer()
		if queryer == nil {
			bucketHosts := httpclient.DefaultBucketHosts()
			var err error
			if request.BucketHosts != nil {
				if bucketHosts, err = request.BucketHosts.GetEndpoints(ctx); err != nil {
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
		req.Region = queryer.Query(accessKey, bucketName)
	}
	var respBody ResponseBody
	if _, err := client.client.AcceptJson(ctx, &req, &respBody); err != nil {
		return nil, err
	}
	return &Response{Body: respBody}, nil
}
