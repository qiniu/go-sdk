// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 修改已上传对象的生命周期
package modify_object_life_cycle

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

type RequestPath struct {
	fieldEntry                  string
	fieldToIaAfterDays          int64
	fieldToArchiveAfterDays     int64
	fieldToDeepArchiveAfterDays int64
	fieldDeleteAfterDays        int64
}

func (pp *RequestPath) GetEntry() string {
	return pp.fieldEntry
}
func (pp *RequestPath) SetEntry(value string) *RequestPath {
	pp.fieldEntry = value
	return pp
}
func (pp *RequestPath) GetToIaAfterDays() int64 {
	return pp.fieldToIaAfterDays
}
func (pp *RequestPath) SetToIaAfterDays(value int64) *RequestPath {
	pp.fieldToIaAfterDays = value
	return pp
}
func (pp *RequestPath) GetToArchiveAfterDays() int64 {
	return pp.fieldToArchiveAfterDays
}
func (pp *RequestPath) SetToArchiveAfterDays(value int64) *RequestPath {
	pp.fieldToArchiveAfterDays = value
	return pp
}
func (pp *RequestPath) GetToDeepArchiveAfterDays() int64 {
	return pp.fieldToDeepArchiveAfterDays
}
func (pp *RequestPath) SetToDeepArchiveAfterDays(value int64) *RequestPath {
	pp.fieldToDeepArchiveAfterDays = value
	return pp
}
func (pp *RequestPath) GetDeleteAfterDays() int64 {
	return pp.fieldDeleteAfterDays
}
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
	if path.fieldToIaAfterDays != 0 {
		allSegments = append(allSegments, "toIAAfterDays", strconv.FormatInt(path.fieldToIaAfterDays, 10))
	}
	if path.fieldToArchiveAfterDays != 0 {
		allSegments = append(allSegments, "toArchiveAfterDays", strconv.FormatInt(path.fieldToArchiveAfterDays, 10))
	}
	if path.fieldToDeepArchiveAfterDays != 0 {
		allSegments = append(allSegments, "toDeepArchiveAfterDays", strconv.FormatInt(path.fieldToDeepArchiveAfterDays, 10))
	}
	if path.fieldDeleteAfterDays != 0 {
		allSegments = append(allSegments, "deleteAfterDays", strconv.FormatInt(path.fieldDeleteAfterDays, 10))
	}
	return allSegments, nil
}

// 调用 API 所用的请求
type Request struct {
	overwrittenBucketHosts region.EndpointsProvider
	overwrittenBucketName  string
	Path                   RequestPath
	credentials            credentials.CredentialsProvider
}

func (request Request) OverwriteBucketHosts(bucketHosts region.EndpointsProvider) Request {
	request.overwrittenBucketHosts = bucketHosts
	return request
}
func (request Request) OverwriteBucketName(bucketName string) Request {
	request.overwrittenBucketName = bucketName
	return request
}
func (request Request) SetCredentials(credentials credentials.CredentialsProvider) Request {
	request.credentials = credentials
	return request
}
func (request Request) getBucketName(ctx context.Context) (string, error) {
	if request.overwrittenBucketName != "" {
		return request.overwrittenBucketName, nil
	}
	if bucketName, err := request.Path.getBucketName(); err != nil || bucketName != "" {
		return bucketName, err
	}
	return "", nil
}
func (request Request) getAccessKey(ctx context.Context) (string, error) {
	if request.credentials != nil {
		if credentials, err := request.credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}
func (request Request) Send(ctx context.Context, options *httpclient.HttpClientOptions) (*Response, error) {
	client := httpclient.NewHttpClient(options)
	serviceNames := []region.ServiceName{region.ServiceRs}
	var pathSegments []string
	pathSegments = append(pathSegments, "lifecycle")
	if segments, err := request.Path.build(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, AuthType: auth.TokenQiniu, Credentials: request.credentials}
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
	defer resp.Body.Close()
	return &Response{}, nil
}

// 获取 API 所用的响应
type Response struct{}
