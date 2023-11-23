// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 更新文件生命周期
package delete_object_after_days

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
	fieldEntry           string
	fieldDeleteAfterDays int64
}

// 指定目标对象空间与目标对象名称
func (pp *RequestPath) GetEntry() string {
	return pp.fieldEntry
}

// 指定目标对象空间与目标对象名称
func (pp *RequestPath) SetEntry(value string) *RequestPath {
	pp.fieldEntry = value
	return pp
}

// 指定文件上传后在设置的 DeleteAfterDays 过期删除，删除后不可恢复，设置为 0 表示取消已设置的过期删除的生命周期规则
func (pp *RequestPath) GetDeleteAfterDays() int64 {
	return pp.fieldDeleteAfterDays
}

// 指定文件上传后在设置的 DeleteAfterDays 过期删除，删除后不可恢复，设置为 0 表示取消已设置的过期删除的生命周期规则
func (pp *RequestPath) SetDeleteAfterDays(value int64) *RequestPath {
	pp.fieldDeleteAfterDays = value
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
	if path.fieldDeleteAfterDays != 0 {
		allSegments = append(allSegments, strconv.FormatInt(path.fieldDeleteAfterDays, 10))
	}
	return allSegments, nil
}

// 指定目标对象空间与目标对象名称
func (request *Request) GetEntry() string {
	return request.path.GetEntry()
}

// 指定目标对象空间与目标对象名称
func (request *Request) SetEntry(value string) *Request {
	request.path.SetEntry(value)
	return request
}

// 指定文件上传后在设置的 DeleteAfterDays 过期删除，删除后不可恢复，设置为 0 表示取消已设置的过期删除的生命周期规则
func (request *Request) GetDeleteAfterDays() int64 {
	return request.path.GetDeleteAfterDays()
}

// 指定文件上传后在设置的 DeleteAfterDays 过期删除，删除后不可恢复，设置为 0 表示取消已设置的过期删除的生命周期规则
func (request *Request) SetDeleteAfterDays(value int64) *Request {
	request.path.SetDeleteAfterDays(value)
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
	pathSegments = append(pathSegments, "deleteAfterDays")
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
	resp, err := client.Do(ctx, &req)
	if err != nil {
		return nil, err
	}
	return &Response{}, resp.Body.Close()
}

// 获取 API 所用的响应
type Response struct{}
