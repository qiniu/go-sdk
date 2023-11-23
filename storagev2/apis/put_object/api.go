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

// 调用 API 所用的请求体
type RequestBody struct {
	fieldObjectName    string
	fieldUploadToken   uptoken.Provider
	fieldCrc32         int64
	fieldFile          io.ReadSeekCloser
	fieldFile_FileName string
	extendedMap        map[string]string
}

// 对象名称，如果不传入，则通过上传策略中的 `saveKey` 字段决定，如果 `saveKey` 也没有置顶，则使用对象的哈希值
func (form *RequestBody) GetObjectName() string {
	return form.fieldObjectName
}

// 对象名称，如果不传入，则通过上传策略中的 `saveKey` 字段决定，如果 `saveKey` 也没有置顶，则使用对象的哈希值
func (form *RequestBody) SetObjectName(value string) *RequestBody {
	form.fieldObjectName = value
	return form
}

// 上传凭证
func (form *RequestBody) GetUploadToken() uptoken.Provider {
	return form.fieldUploadToken
}

// 上传凭证
func (form *RequestBody) SetUploadToken(value uptoken.Provider) *RequestBody {
	form.fieldUploadToken = value
	return form
}

// 上传内容的 CRC32 校验码，如果指定此值，则七牛服务器会使用此值进行内容检验
func (form *RequestBody) GetCrc32() int64 {
	return form.fieldCrc32
}

// 上传内容的 CRC32 校验码，如果指定此值，则七牛服务器会使用此值进行内容检验
func (form *RequestBody) SetCrc32(value int64) *RequestBody {
	form.fieldCrc32 = value
	return form
}

// 上传文件的内容
func (form *RequestBody) GetFile() (io.ReadSeekCloser, string) {
	return form.fieldFile, form.fieldFile_FileName
}

// 上传文件的内容
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
	return request.body.GetObjectName()
}
func (request *Request) SetObjectName(value string) *Request {
	request.body.SetObjectName(value)
	return request
}
func (request *Request) GetUploadToken() uptoken.Provider {
	return request.body.GetUploadToken()
}
func (request *Request) SetUploadToken(value uptoken.Provider) *Request {
	request.body.SetUploadToken(value)
	return request
}
func (request *Request) GetCrc32() int64 {
	return request.body.GetCrc32()
}
func (request *Request) SetCrc32(value int64) *Request {
	request.body.SetCrc32(value)
	return request
}
func (request *Request) GetFile() (io.ReadSeekCloser, string) {
	return request.body.GetFile()
}
func (request *Request) SetFile(value io.ReadSeekCloser, fileName string) *Request {
	request.body.SetFile(value, fileName)
	return request
}

// 获取 API 所用的响应体参数
type ResponseBody = interface{}

// 调用 API 所用的请求
type Request struct {
	overwrittenBucketHosts region.EndpointsProvider
	overwrittenBucketName  string
	body                   RequestBody
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
func (request *Request) getBucketName(ctx context.Context) (string, error) {
	if request.overwrittenBucketName != "" {
		return request.overwrittenBucketName, nil
	}
	if bucketName, err := request.body.getBucketName(ctx); err != nil || bucketName != "" {
		return bucketName, err
	}
	return "", nil
}
func (request *Request) getAccessKey(ctx context.Context) (string, error) {
	if request.body.fieldUploadToken != nil {
		if accessKey, err := request.body.fieldUploadToken.RetrieveAccessKey(ctx); err != nil {
			return "", err
		} else {
			return accessKey, nil
		}
	}
	return "", nil
}

// 获取请求体
func (request *Request) GetBody() *RequestBody {
	return &request.body
}

// 设置请求体
func (request *Request) SetBody(body RequestBody) *Request {
	request.body = body
	return request
}

// 发送请求
func (request *Request) Send(ctx context.Context, options *httpclient.HttpClientOptions) (*Response, error) {
	client := httpclient.NewHttpClient(options)
	serviceNames := []region.ServiceName{region.ServiceUp}
	var pathSegments []string
	pathSegments = append(pathSegments, "")
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	body, err := request.body.build(ctx)
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
func (response *Response) GetBody() ResponseBody {
	return response.body
}

// 设置请求体
func (response *Response) SetBody(body ResponseBody) *Response {
	response.body = body
	return response
}
