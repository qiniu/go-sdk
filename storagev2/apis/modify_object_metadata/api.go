// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 修改文件元信息
package modify_object_metadata

import (
	"context"
	"encoding/base64"
	auth "github.com/qiniu/go-sdk/v7/auth"
	credentials "github.com/qiniu/go-sdk/v7/storagev2/credentials"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"strings"
)

type RequestPath struct {
	fieldEntry       string
	fieldMimeType    string
	fieldCondition   string
	extendedSegments []string
}

func (pp *RequestPath) GetEntry() string {
	return pp.fieldEntry
}
func (pp *RequestPath) SetEntry(value string) *RequestPath {
	pp.fieldEntry = value
	return pp
}
func (pp *RequestPath) GetMimeType() string {
	return pp.fieldMimeType
}
func (pp *RequestPath) SetMimeType(value string) *RequestPath {
	pp.fieldMimeType = value
	return pp
}
func (pp *RequestPath) GetCondition() string {
	return pp.fieldCondition
}
func (pp *RequestPath) SetCondition(value string) *RequestPath {
	pp.fieldCondition = value
	return pp
}
func (pp *RequestPath) getBucketName() (string, error) {
	return strings.SplitN(pp.fieldEntry, ":", 2)[0], nil
}
func (path *RequestPath) Append(key string, value string) *RequestPath {
	path.extendedSegments = append(path.extendedSegments, key)
	path.extendedSegments = append(path.extendedSegments, base64.URLEncoding.EncodeToString([]byte(value)))
	return path
}
func (path *RequestPath) build() []string {
	var allSegments []string
	if path.fieldEntry != "" {
		allSegments = append(allSegments, base64.URLEncoding.EncodeToString([]byte(path.fieldEntry)))
	}
	if path.fieldMimeType != "" {
		allSegments = append(allSegments, "mime", base64.URLEncoding.EncodeToString([]byte(path.fieldMimeType)))
	}
	if path.fieldCondition != "" {
		allSegments = append(allSegments, "cond", base64.URLEncoding.EncodeToString([]byte(path.fieldCondition)))
	}
	allSegments = append(allSegments, path.extendedSegments...)
	return allSegments
}

// 调用 API 所用的请求
type Request struct {
	BucketHosts region.EndpointsProvider
	Path        RequestPath
	Credentials credentials.CredentialsProvider
}

func (request Request) getBucketName(ctx context.Context) (string, error) {
	if bucketName, err := request.Path.getBucketName(); err != nil || bucketName != "" {
		return bucketName, err
	}
	return "", nil
}
func (request Request) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

// 获取 API 所用的响应
type Response struct{}

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
	serviceNames := []region.ServiceName{region.ServiceRs}
	var pathSegments []string
	pathSegments = append(pathSegments, "chgm")
	pathSegments = append(pathSegments, request.Path.build()...)
	path := "/" + strings.Join(pathSegments, "/")
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, AuthType: auth.TokenQiniu, Credentials: request.Credentials}
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
	resp, err := client.client.Do(ctx, &req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return &Response{}, nil
}
