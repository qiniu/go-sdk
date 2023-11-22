// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 将源空间的指定对象复制到目标空间
package copy_object

import (
	"context"
	"encoding/base64"
	auth "github.com/qiniu/go-sdk/v7/auth"
	credentials "github.com/qiniu/go-sdk/v7/storagev2/credentials"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"strconv"
	"strings"
)

// 调用 API 所用的路径参数
type RequestPath struct {
	fieldSrcEntry  string
	fieldDestEntry string
	fieldIsForce   bool
}

// 指定源对象空间与源对象名称
func (pp *RequestPath) GetSrcEntry() string {
	return pp.fieldSrcEntry
}

// 指定源对象空间与源对象名称
func (pp *RequestPath) SetSrcEntry(value string) *RequestPath {
	pp.fieldSrcEntry = value
	return pp
}

// 指定目标对象空间与目标对象名称
func (pp *RequestPath) GetDestEntry() string {
	return pp.fieldDestEntry
}

// 指定目标对象空间与目标对象名称
func (pp *RequestPath) SetDestEntry(value string) *RequestPath {
	pp.fieldDestEntry = value
	return pp
}

// 如果目标对象名已被占用，则返回错误码 614，且不做任何覆盖操作；如果指定为 true，会强制覆盖目标对象
func (pp *RequestPath) IsForce() bool {
	return pp.fieldIsForce
}

// 如果目标对象名已被占用，则返回错误码 614，且不做任何覆盖操作；如果指定为 true，会强制覆盖目标对象
func (pp *RequestPath) SetForce(value bool) *RequestPath {
	pp.fieldIsForce = value
	return pp
}
func (pp *RequestPath) getBucketName() (string, error) {
	return strings.SplitN(pp.fieldSrcEntry, ":", 2)[0], nil
}
func (path *RequestPath) build() ([]string, error) {
	var allSegments []string
	if path.fieldSrcEntry != "" {
		allSegments = append(allSegments, base64.URLEncoding.EncodeToString([]byte(path.fieldSrcEntry)))
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "SrcEntry"}
	}
	if path.fieldDestEntry != "" {
		allSegments = append(allSegments, base64.URLEncoding.EncodeToString([]byte(path.fieldDestEntry)))
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "DestEntry"}
	}
	if path.fieldIsForce {
		allSegments = append(allSegments, "force", strconv.FormatBool(path.fieldIsForce))
	}
	return allSegments, nil
}

// 指定源对象空间与源对象名称
func (request *Request) GetSrcEntry() string {
	return request.path.GetSrcEntry()
}

// 指定源对象空间与源对象名称
func (request *Request) SetSrcEntry(value string) *Request {
	request.path.SetSrcEntry(value)
	return request
}

// 指定目标对象空间与目标对象名称
func (request *Request) GetDestEntry() string {
	return request.path.GetDestEntry()
}

// 指定目标对象空间与目标对象名称
func (request *Request) SetDestEntry(value string) *Request {
	request.path.SetDestEntry(value)
	return request
}

// 如果目标对象名已被占用，则返回错误码 614，且不做任何覆盖操作；如果指定为 true，会强制覆盖目标对象
func (request *Request) IsForce() bool {
	return request.path.IsForce()
}

// 如果目标对象名已被占用，则返回错误码 614，且不做任何覆盖操作；如果指定为 true，会强制覆盖目标对象
func (request *Request) SetForce(value bool) *Request {
	request.path.SetForce(value)
	return request
}

// 调用 API 所用的请求
type Request struct {
	overwrittenBucketHosts region.EndpointsProvider
	overwrittenBucketName  string
	path                   RequestPath
	credentials            credentials.CredentialsProvider
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

// 设置鉴权
func (request *Request) SetCredentials(credentials credentials.CredentialsProvider) *Request {
	request.credentials = credentials
	return request
}
func (request *Request) getBucketName(ctx context.Context) (string, error) {
	if request.overwrittenBucketName != "" {
		return request.overwrittenBucketName, nil
	}
	if bucketName, err := request.path.getBucketName(); err != nil || bucketName != "" {
		return bucketName, err
	}
	return "", nil
}
func (request *Request) getAccessKey(ctx context.Context) (string, error) {
	if request.credentials != nil {
		if credentials, err := request.credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
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
	serviceNames := []region.ServiceName{region.ServiceRs}
	var pathSegments []string
	pathSegments = append(pathSegments, "copy")
	if segments, err := request.path.build(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, AuthType: auth.TokenQiniu, Credentials: request.credentials}
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
	resp, err := client.Do(ctx, &req)
	if err != nil {
		return nil, err
	}
	return &Response{}, resp.Body.Close()
}

// 获取 API 所用的响应
type Response struct{}
