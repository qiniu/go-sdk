// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 列举指定存储空间里的所有对象条目
package get_objects_v2

import (
	"context"
	auth "github.com/qiniu/go-sdk/v7/auth"
	credentials "github.com/qiniu/go-sdk/v7/storagev2/credentials"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"io"
	"net/url"
	"strconv"
	"strings"
)

// 调用 API 所用的 URL 查询参数
type RequestQuery struct {
	fieldBucket    string // 指定存储空间
	fieldMarker    string // 上一次列举返回的位置标记，作为本次列举的起点信息
	fieldLimit     int64  // 本次列举的条目数，范围为 1-1000
	fieldPrefix    string // 指定前缀，只有资源名匹配该前缀的资源会被列出
	fieldDelimiter string // 指定目录分隔符，列出所有公共前缀（模拟列出目录效果）
	fieldNeedParts bool   // 如果文件是通过分片上传的，是否返回对应的分片信息
}

func (query *RequestQuery) GetBucket() string {
	return query.fieldBucket
}
func (query *RequestQuery) SetBucket(value string) *RequestQuery {
	query.fieldBucket = value
	return query
}
func (query *RequestQuery) GetMarker() string {
	return query.fieldMarker
}
func (query *RequestQuery) SetMarker(value string) *RequestQuery {
	query.fieldMarker = value
	return query
}
func (query *RequestQuery) GetLimit() int64 {
	return query.fieldLimit
}
func (query *RequestQuery) SetLimit(value int64) *RequestQuery {
	query.fieldLimit = value
	return query
}
func (query *RequestQuery) GetPrefix() string {
	return query.fieldPrefix
}
func (query *RequestQuery) SetPrefix(value string) *RequestQuery {
	query.fieldPrefix = value
	return query
}
func (query *RequestQuery) GetDelimiter() string {
	return query.fieldDelimiter
}
func (query *RequestQuery) SetDelimiter(value string) *RequestQuery {
	query.fieldDelimiter = value
	return query
}
func (query *RequestQuery) GetNeedParts() bool {
	return query.fieldNeedParts
}
func (query *RequestQuery) SetNeedParts(value bool) *RequestQuery {
	query.fieldNeedParts = value
	return query
}
func (query *RequestQuery) getBucketName() (string, error) {
	return query.fieldBucket, nil
}
func (query *RequestQuery) build() (url.Values, error) {
	allQuery := make(url.Values)
	if query.fieldBucket != "" {
		allQuery.Set("bucket", query.fieldBucket)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Bucket"}
	}
	if query.fieldMarker != "" {
		allQuery.Set("marker", query.fieldMarker)
	}
	if query.fieldLimit != 0 {
		allQuery.Set("limit", strconv.FormatInt(query.fieldLimit, 10))
	}
	if query.fieldPrefix != "" {
		allQuery.Set("prefix", query.fieldPrefix)
	}
	if query.fieldDelimiter != "" {
		allQuery.Set("delimiter", query.fieldDelimiter)
	}
	if query.fieldNeedParts {
		allQuery.Set("needparts", strconv.FormatBool(query.fieldNeedParts))
	}
	return allQuery, nil
}

// 调用 API 所用的请求
type Request struct {
	overwrittenBucketHosts region.EndpointsProvider
	overwrittenBucketName  string
	Query                  RequestQuery
	credentials            credentials.CredentialsProvider
}

func (request *Request) OverwriteBucketHosts(bucketHosts region.EndpointsProvider) *Request {
	request.overwrittenBucketHosts = bucketHosts
	return request
}
func (request *Request) OverwriteBucketName(bucketName string) *Request {
	request.overwrittenBucketName = bucketName
	return request
}
func (request *Request) SetCredentials(credentials credentials.CredentialsProvider) *Request {
	request.credentials = credentials
	return request
}
func (request *Request) getBucketName(ctx context.Context) (string, error) {
	if request.overwrittenBucketName != "" {
		return request.overwrittenBucketName, nil
	}
	if bucketName, err := request.Query.getBucketName(); err != nil || bucketName != "" {
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
func (request *Request) Send(ctx context.Context, options *httpclient.HttpClientOptions) (*Response, error) {
	client := httpclient.NewHttpClient(options)
	serviceNames := []region.ServiceName{region.ServiceRsf}
	var pathSegments []string
	pathSegments = append(pathSegments, "v2", "list")
	path := "/" + strings.Join(pathSegments, "/")
	query, err := request.Query.build()
	if err != nil {
		return nil, err
	}
	req := httpclient.Request{Method: "GET", ServiceNames: serviceNames, Path: path, RawQuery: query.Encode(), AuthType: auth.TokenQiniu, Credentials: request.credentials}
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
	return &Response{Body: resp.Body}, nil
}

// 获取 API 所用的响应
type Response struct {
	Body io.ReadCloser
}
