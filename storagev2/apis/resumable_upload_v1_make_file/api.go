// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 将上传好的所有数据块按指定顺序合并成一个资源文件
package resumable_upload_v1_make_file

import (
	"context"
	"encoding/base64"
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
	fieldSize        int64
	fieldObjectName  string
	fieldFileName    string
	fieldMimeType    string
	extendedSegments []string
}

// 对象大小
func (pp *RequestPath) GetSize() int64 {
	return pp.fieldSize
}

// 对象大小
func (pp *RequestPath) SetSize(value int64) *RequestPath {
	pp.fieldSize = value
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

// 文件名称，若未指定，则魔法变量中无法使用fname，ext，fprefix
func (pp *RequestPath) GetFileName() string {
	return pp.fieldFileName
}

// 文件名称，若未指定，则魔法变量中无法使用fname，ext，fprefix
func (pp *RequestPath) SetFileName(value string) *RequestPath {
	pp.fieldFileName = value
	return pp
}

// 文件 MIME 类型，若未指定，则根据文件内容自动检测 MIME 类型
func (pp *RequestPath) GetMimeType() string {
	return pp.fieldMimeType
}

// 文件 MIME 类型，若未指定，则根据文件内容自动检测 MIME 类型
func (pp *RequestPath) SetMimeType(value string) *RequestPath {
	pp.fieldMimeType = value
	return pp
}

// 追加自定义路径参数
func (path *RequestPath) Append(key string, value string) *RequestPath {
	path.extendedSegments = append(path.extendedSegments, key)
	path.extendedSegments = append(path.extendedSegments, base64.URLEncoding.EncodeToString([]byte(value)))
	return path
}
func (path *RequestPath) build() ([]string, error) {
	var allSegments []string
	if path.fieldSize != 0 {
		allSegments = append(allSegments, strconv.FormatInt(path.fieldSize, 10))
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Size"}
	}
	if path.fieldObjectName != "" {
		allSegments = append(allSegments, "key", base64.URLEncoding.EncodeToString([]byte(path.fieldObjectName)))
	}
	if path.fieldFileName != "" {
		allSegments = append(allSegments, "fname", base64.URLEncoding.EncodeToString([]byte(path.fieldFileName)))
	}
	if path.fieldMimeType != "" {
		allSegments = append(allSegments, "mimeType", base64.URLEncoding.EncodeToString([]byte(path.fieldMimeType)))
	}
	allSegments = append(allSegments, path.extendedSegments...)
	return allSegments, nil
}

// 对象大小
func (request *Request) GetSize() int64 {
	return request.path.GetSize()
}

// 对象大小
func (request *Request) SetSize(value int64) *Request {
	request.path.SetSize(value)
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

// 文件名称，若未指定，则魔法变量中无法使用fname，ext，fprefix
func (request *Request) GetFileName() string {
	return request.path.GetFileName()
}

// 文件名称，若未指定，则魔法变量中无法使用fname，ext，fprefix
func (request *Request) SetFileName(value string) *Request {
	request.path.SetFileName(value)
	return request
}

// 文件 MIME 类型，若未指定，则根据文件内容自动检测 MIME 类型
func (request *Request) GetMimeType() string {
	return request.path.GetMimeType()
}

// 文件 MIME 类型，若未指定，则根据文件内容自动检测 MIME 类型
func (request *Request) SetMimeType(value string) *Request {
	request.path.SetMimeType(value)
	return request
}

// 获取 API 所用的响应体参数
type ResponseBody = interface{}

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
	pathSegments = append(pathSegments, "mkfile")
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
func (response *Response) GetBody() ResponseBody {
	return response.body
}

// 设置请求体
func (response *Response) SetBody(body ResponseBody) *Response {
	response.body = body
	return response
}
