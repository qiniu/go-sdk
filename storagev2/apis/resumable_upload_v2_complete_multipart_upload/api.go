// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 在将所有数据分片都上传完成后，必须调用 completeMultipartUpload API 来完成整个文件的 Multipart Upload。用户需要提供有效数据的分片列表（包括 PartNumber 和调用 uploadPart API 服务端返回的 Etag）。服务端收到用户提交的分片列表后，会逐一验证每个数据分片的有效性。当所有的数据分片验证通过后，会把这些数据分片组合成一个完整的对象
package resumable_upload_v2_complete_multipart_upload

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

type innerPartInfo struct {
	PartNumber int64  `json:"partNumber,omitempty"` // 每一个上传的分片都有一个标识它的号码
	Etag       string `json:"etag,omitempty"`       // 上传块的 etag
}

// 单个分片信息
type PartInfo struct {
	inner innerPartInfo
}

func (j *PartInfo) GetPartNumber() int64 {
	return j.inner.PartNumber
}
func (j *PartInfo) SetPartNumber(value int64) *PartInfo {
	j.inner.PartNumber = value
	return j
}
func (j *PartInfo) GetEtag() string {
	return j.inner.Etag
}
func (j *PartInfo) SetEtag(value string) *PartInfo {
	j.inner.Etag = value
	return j
}
func (j *PartInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(&j.inner)
}
func (j *PartInfo) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.inner)
}

//lint:ignore U1000 may not call it
func (j *PartInfo) validate() error {
	if j.inner.PartNumber == 0 {
		return errors.MissingRequiredFieldError{Name: "PartNumber"}
	}
	if j.inner.Etag == "" {
		return errors.MissingRequiredFieldError{Name: "Etag"}
	}
	return nil
}

// 分片信息列表
type Parts = []PartInfo
type innerObjectInfo struct {
	Parts      Parts             `json:"parts,omitempty"`      // 已经上传的分片列表
	FileName   string            `json:"fname,omitempty"`      // 上传的原始文件名，若未指定，则魔法变量中无法使用 fname，ext，suffix
	MimeType   string            `json:"mime_type,omitempty"`  // 若指定了则设置上传文件的 MIME 类型，若未指定，则根据文件内容自动检测 MIME 类型
	Metadata   map[string]string `json:"metadata,omitempty"`   // 用户自定义文件 metadata 信息的键值对，可以设置多个，MetaKey 和 MetaValue 都是 string，，其中 可以由字母、数字、下划线、减号组成，且长度小于等于 50，单个文件 MetaKey 和 MetaValue 总和大小不能超过 1024 字节，MetaKey 必须以 `x-qn-meta-` 作为前缀
	CustomVars map[string]string `json:"customVars,omitempty"` // 用户自定义变量
}

// 新上传的对象的相关信息
type ObjectInfo struct {
	inner innerObjectInfo
}

func (j *ObjectInfo) GetParts() Parts {
	return j.inner.Parts
}
func (j *ObjectInfo) SetParts(value Parts) *ObjectInfo {
	j.inner.Parts = value
	return j
}
func (j *ObjectInfo) GetFileName() string {
	return j.inner.FileName
}
func (j *ObjectInfo) SetFileName(value string) *ObjectInfo {
	j.inner.FileName = value
	return j
}
func (j *ObjectInfo) GetMimeType() string {
	return j.inner.MimeType
}
func (j *ObjectInfo) SetMimeType(value string) *ObjectInfo {
	j.inner.MimeType = value
	return j
}
func (j *ObjectInfo) GetMetadata() map[string]string {
	return j.inner.Metadata
}
func (j *ObjectInfo) SetMetadata(value map[string]string) *ObjectInfo {
	j.inner.Metadata = value
	return j
}
func (j *ObjectInfo) GetCustomVars() map[string]string {
	return j.inner.CustomVars
}
func (j *ObjectInfo) SetCustomVars(value map[string]string) *ObjectInfo {
	j.inner.CustomVars = value
	return j
}
func (j *ObjectInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(&j.inner)
}
func (j *ObjectInfo) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.inner)
}

//lint:ignore U1000 may not call it
func (j *ObjectInfo) validate() error {
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

// 调用 API 所用的请求体
type RequestBody = ObjectInfo
type ResponseBody = interface{}

// 调用 API 所用的请求
type Request struct {
	overwrittenBucketHosts region.EndpointsProvider
	overwrittenBucketName  string
	Path                   RequestPath
	upToken                uptoken.Provider
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
func (request Request) SetUpToken(upToken uptoken.Provider) Request {
	request.upToken = upToken
	return request
}
func (request Request) getBucketName(ctx context.Context) (string, error) {
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
func (request Request) getAccessKey(ctx context.Context) (string, error) {
	if request.upToken != nil {
		return request.upToken.RetrieveAccessKey(ctx)
	}
	return "", nil
}
func (request Request) Send(ctx context.Context, options *httpclient.HttpClientOptions) (*Response, error) {
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
	if err := request.Body.validate(); err != nil {
		return nil, err
	}
	body, err := httpclient.GetJsonRequestBody(&request.Body)
	if err != nil {
		return nil, err
	}
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, UpToken: request.upToken, RequestBody: body}
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
