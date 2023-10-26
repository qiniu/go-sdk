// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 根据 UploadId 终止 Multipart Upload
package resumable_upload_v2_abort_multipart_upload

import (
	"context"
	"encoding/base64"
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
	return allSegments
}

// 调用 API 所用的请求
type Request struct {
	BucketHosts region.EndpointsProvider
	Path        RequestPath
	UpToken     uptoken.Provider
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
	serviceNames := []region.ServiceName{region.ServiceUp}
	var pathSegments []string
	pathSegments = append(pathSegments, "buckets")
	pathSegments = append(pathSegments, request.Path.build()...)
	path := "/" + strings.Join(pathSegments, "/")
	req := httpclient.Request{Method: "DELETE", ServiceNames: serviceNames, Path: path, UpToken: request.UpToken}
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
