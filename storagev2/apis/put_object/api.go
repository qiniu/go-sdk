// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 在一次 HTTP 会话中上传单一的一个文件
package put_object

import (
	"context"
	io "github.com/qiniu/go-sdk/v7/internal/io"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	uptoken "github.com/qiniu/go-sdk/v7/storagev2/uptoken"
	"strconv"
	"strings"
)

type RequestBody struct {
	fieldObjectName    string
	fieldUploadToken   uptoken.Provider
	fieldCrc32         int64
	fieldFile          io.ReadSeekCloser
	fieldFile_FileName string
	extendedMap        map[string]string
}

func (form *RequestBody) GetObjectName() string {
	return form.fieldObjectName
}
func (form *RequestBody) SetObjectName(value string) *RequestBody {
	form.fieldObjectName = value
	return form
}
func (form *RequestBody) GetUploadToken() uptoken.Provider {
	return form.fieldUploadToken
}
func (form *RequestBody) SetUploadToken(value uptoken.Provider) *RequestBody {
	form.fieldUploadToken = value
	return form
}
func (form *RequestBody) GetCrc32() int64 {
	return form.fieldCrc32
}
func (form *RequestBody) SetCrc32(value int64) *RequestBody {
	form.fieldCrc32 = value
	return form
}
func (form *RequestBody) GetFile() (io.ReadSeekCloser, string) {
	return form.fieldFile, form.fieldFile_FileName
}
func (form *RequestBody) SetFile(value io.ReadSeekCloser, fileName string) *RequestBody {
	form.fieldFile = value
	form.fieldFile_FileName = fileName
	return form
}
func (form *RequestBody) getBucketName(ctx context.Context) (string, error) {
	putPolicy, err := form.fieldUploadToken.RetrievePutPolicy(ctx)
	if err != nil {
		return "", err
	} else {
		return putPolicy.GetBucketName()
	}
}
func (form *RequestBody) Set(key string, value string) *RequestBody {
	if form.extendedMap == nil {
		form.extendedMap = make(map[string]string)
	}
	form.extendedMap[key] = value
	return form
}
func (form *RequestBody) build(ctx context.Context) (*httpclient.MultipartForm, error) {
	multipartForm := new(httpclient.MultipartForm)
	if form.fieldObjectName != "" {
		multipartForm.SetValue("key", form.fieldObjectName)
	}
	if form.fieldUploadToken != nil {
		upToken, err := form.fieldUploadToken.RetrieveUpToken(ctx)
		if err != nil {
			return nil, err
		}
		multipartForm.SetValue("token", upToken)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "UploadToken"}
	}
	if form.fieldCrc32 != 0 {
		multipartForm.SetValue("crc32", strconv.FormatInt(form.fieldCrc32, 10))
	}
	if form.fieldFile != nil {
		multipartForm.SetFile("file", form.fieldFile_FileName, form.fieldFile)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "File"}
	}
	for key, value := range form.extendedMap {
		multipartForm.SetValue(key, value)
	}
	return multipartForm, nil
}
func (request *Request) GetObjectName() string {
	return request.Body.GetObjectName()
}
func (request *Request) SetObjectName(value string) *Request {
	request.Body.SetObjectName(value)
	return request
}
func (request *Request) GetUploadToken() uptoken.Provider {
	return request.Body.GetUploadToken()
}
func (request *Request) SetUploadToken(value uptoken.Provider) *Request {
	request.Body.SetUploadToken(value)
	return request
}
func (request *Request) GetCrc32() int64 {
	return request.Body.GetCrc32()
}
func (request *Request) SetCrc32(value int64) *Request {
	request.Body.SetCrc32(value)
	return request
}
func (request *Request) GetFile() (io.ReadSeekCloser, string) {
	return request.Body.GetFile()
}
func (request *Request) SetFile(value io.ReadSeekCloser, fileName string) *Request {
	request.Body.SetFile(value, fileName)
	return request
}

type ResponseBody = interface{}

// 调用 API 所用的请求
type Request struct {
	overwrittenBucketHosts region.EndpointsProvider
	overwrittenBucketName  string
	Body                   RequestBody
}

func (request *Request) OverwriteBucketHosts(bucketHosts region.EndpointsProvider) *Request {
	request.overwrittenBucketHosts = bucketHosts
	return request
}
func (request *Request) OverwriteBucketName(bucketName string) *Request {
	request.overwrittenBucketName = bucketName
	return request
}
func (request *Request) getBucketName(ctx context.Context) (string, error) {
	if request.overwrittenBucketName != "" {
		return request.overwrittenBucketName, nil
	}
	if bucketName, err := request.Body.getBucketName(ctx); err != nil || bucketName != "" {
		return bucketName, err
	}
	return "", nil
}
func (request *Request) getAccessKey(ctx context.Context) (string, error) {
	if request.Body.fieldUploadToken != nil {
		if accessKey, err := request.Body.fieldUploadToken.RetrieveAccessKey(ctx); err != nil {
			return "", err
		} else {
			return accessKey, nil
		}
	}
	return "", nil
}
func (request *Request) Send(ctx context.Context, options *httpclient.HttpClientOptions) (*Response, error) {
	client := httpclient.NewHttpClient(options)
	serviceNames := []region.ServiceName{region.ServiceUp}
	var pathSegments []string
	pathSegments = append(pathSegments, "")
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	body, err := request.Body.build(ctx)
	if err != nil {
		return nil, err
	}
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, RequestBody: httpclient.GetMultipartFormRequestBody(body)}
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
