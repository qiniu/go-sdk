// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 列举出指定 UploadId 所属任务所有已经上传成功的分片
package resumable_upload_v2_list_parts

import (
	"context"
	"encoding/base64"
	"encoding/json"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	uptoken "github.com/qiniu/go-sdk/v7/storagev2/uptoken"
	"net/url"
	"strconv"
	"strings"
)

type RequestPath struct {
	fieldBucketName string
	fieldObjectName string
	fieldUploadId   string
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
	if path.fieldUploadId != "" {
		allSegments = append(allSegments, "uploads", path.fieldUploadId)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "UploadId"}
	}
	return allSegments, nil
}

// 调用 API 所用的 URL 查询参数
type RequestQuery struct {
	fieldMaxParts         int64 // 响应中的最大分片数目。默认值：1000，最大值：1000
	fieldPartNumberMarker int64 // 指定列举的起始位置，只有 partNumber 值大于该参数的分片会被列出
}

func (query *RequestQuery) GetMaxParts() int64 {
	return query.fieldMaxParts
}
func (query *RequestQuery) SetMaxParts(value int64) *RequestQuery {
	query.fieldMaxParts = value
	return query
}
func (query *RequestQuery) GetPartNumberMarker() int64 {
	return query.fieldPartNumberMarker
}
func (query *RequestQuery) SetPartNumberMarker(value int64) *RequestQuery {
	query.fieldPartNumberMarker = value
	return query
}
func (query *RequestQuery) build() (url.Values, error) {
	allQuery := make(url.Values)
	if query.fieldMaxParts != 0 {
		allQuery.Set("max-parts", strconv.FormatInt(query.fieldMaxParts, 10))
	}
	if query.fieldPartNumberMarker != 0 {
		allQuery.Set("part-number_marker", strconv.FormatInt(query.fieldPartNumberMarker, 10))
	}
	return allQuery, nil
}

type innerListedPartInfo struct {
	Size       int64  `json:"size"`       // 分片大小
	Etag       string `json:"etag"`       // 分片内容的 etag
	PartNumber int64  `json:"partNumber"` // 每一个上传的分片都有一个标识它的号码
	PutTime    int64  `json:"putTime"`    // 分片上传时间 UNIX 时间戳
}

// 单个已经上传的分片信息
type ListedPartInfo struct {
	inner innerListedPartInfo
}

func (j *ListedPartInfo) GetSize() int64 {
	return j.inner.Size
}
func (j *ListedPartInfo) SetSize(value int64) *ListedPartInfo {
	j.inner.Size = value
	return j
}
func (j *ListedPartInfo) GetEtag() string {
	return j.inner.Etag
}
func (j *ListedPartInfo) SetEtag(value string) *ListedPartInfo {
	j.inner.Etag = value
	return j
}
func (j *ListedPartInfo) GetPartNumber() int64 {
	return j.inner.PartNumber
}
func (j *ListedPartInfo) SetPartNumber(value int64) *ListedPartInfo {
	j.inner.PartNumber = value
	return j
}
func (j *ListedPartInfo) GetPutTime() int64 {
	return j.inner.PutTime
}
func (j *ListedPartInfo) SetPutTime(value int64) *ListedPartInfo {
	j.inner.PutTime = value
	return j
}
func (j *ListedPartInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(&j.inner)
}
func (j *ListedPartInfo) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.inner)
}

//lint:ignore U1000 may not call it
func (j *ListedPartInfo) validate() error {
	if j.inner.Size == 0 {
		return errors.MissingRequiredFieldError{Name: "Size"}
	}
	if j.inner.Etag == "" {
		return errors.MissingRequiredFieldError{Name: "Etag"}
	}
	if j.inner.PartNumber == 0 {
		return errors.MissingRequiredFieldError{Name: "PartNumber"}
	}
	if j.inner.PutTime == 0 {
		return errors.MissingRequiredFieldError{Name: "PutTime"}
	}
	return nil
}

// 所有已经上传的分片信息
type ListedParts = []ListedPartInfo

// 返回所有已经上传成功的分片信息
type Parts = ListedParts
type innerListedPartsResponse struct {
	UploadId         string      `json:"uploadId"`         // 在服务端申请的 Multipart Upload 任务 id
	ExpiredAt        int64       `json:"expireAt"`         // UploadId 的过期时间 UNIX 时间戳，过期之后 UploadId 不可用
	PartNumberMarker int64       `json:"partNumberMarker"` // 下次继续列举的起始位置，0 表示列举结束，没有更多分片
	Parts            ListedParts `json:"parts"`            // 返回所有已经上传成功的分片信息
}

// 返回所有已经上传成功的分片信息
type ListedPartsResponse struct {
	inner innerListedPartsResponse
}

func (j *ListedPartsResponse) GetUploadId() string {
	return j.inner.UploadId
}
func (j *ListedPartsResponse) SetUploadId(value string) *ListedPartsResponse {
	j.inner.UploadId = value
	return j
}
func (j *ListedPartsResponse) GetExpiredAt() int64 {
	return j.inner.ExpiredAt
}
func (j *ListedPartsResponse) SetExpiredAt(value int64) *ListedPartsResponse {
	j.inner.ExpiredAt = value
	return j
}
func (j *ListedPartsResponse) GetPartNumberMarker() int64 {
	return j.inner.PartNumberMarker
}
func (j *ListedPartsResponse) SetPartNumberMarker(value int64) *ListedPartsResponse {
	j.inner.PartNumberMarker = value
	return j
}
func (j *ListedPartsResponse) GetParts() ListedParts {
	return j.inner.Parts
}
func (j *ListedPartsResponse) SetParts(value ListedParts) *ListedPartsResponse {
	j.inner.Parts = value
	return j
}
func (j *ListedPartsResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(&j.inner)
}
func (j *ListedPartsResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.inner)
}

//lint:ignore U1000 may not call it
func (j *ListedPartsResponse) validate() error {
	if j.inner.UploadId == "" {
		return errors.MissingRequiredFieldError{Name: "UploadId"}
	}
	if j.inner.ExpiredAt == 0 {
		return errors.MissingRequiredFieldError{Name: "ExpiredAt"}
	}
	if j.inner.PartNumberMarker == 0 {
		return errors.MissingRequiredFieldError{Name: "PartNumberMarker"}
	}
	if len(j.inner.Parts) > 0 {
		return errors.MissingRequiredFieldError{Name: "Parts"}
	}
	for _, value := range j.inner.Parts {
		if err := value.validate(); err != nil {
			return err
		}
	}
	return nil
}

// 获取 API 所用的响应体参数
type ResponseBody = ListedPartsResponse

// 调用 API 所用的请求
type Request struct {
	overwrittenBucketHosts region.EndpointsProvider
	overwrittenBucketName  string
	Path                   RequestPath
	Query                  RequestQuery
	upToken                uptoken.Provider
}

func (request *Request) OverwriteBucketHosts(bucketHosts region.EndpointsProvider) *Request {
	request.overwrittenBucketHosts = bucketHosts
	return request
}
func (request *Request) OverwriteBucketName(bucketName string) *Request {
	request.overwrittenBucketName = bucketName
	return request
}
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
func (request *Request) Send(ctx context.Context, options *httpclient.HttpClientOptions) (*Response, error) {
	client := httpclient.NewHttpClient(options)
	serviceNames := []region.ServiceName{region.ServiceUp}
	var pathSegments []string
	pathSegments = append(pathSegments, "buckets")
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
	req := httpclient.Request{Method: "GET", ServiceNames: serviceNames, Path: path, RawQuery: query.Encode(), UpToken: request.upToken}
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
