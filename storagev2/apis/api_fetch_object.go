// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/base64"
	"encoding/json"
	auth "github.com/qiniu/go-sdk/v7/auth"
	fetchobject "github.com/qiniu/go-sdk/v7/storagev2/apis/fetch_object"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"strings"
)

type innerFetchObjectRequest fetchobject.Request

func (pp *innerFetchObjectRequest) getBucketName(ctx context.Context) (string, error) {
	return strings.SplitN(pp.ToEntry, ":", 2)[0], nil
}
func (path *innerFetchObjectRequest) buildPath() ([]string, error) {
	var allSegments []string
	if path.FromUrl != "" {
		allSegments = append(allSegments, base64.URLEncoding.EncodeToString([]byte(path.FromUrl)))
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "FromUrl"}
	}
	if path.ToEntry != "" {
		allSegments = append(allSegments, "to", base64.URLEncoding.EncodeToString([]byte(path.ToEntry)))
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "ToEntry"}
	}
	if path.Host != "" {
		allSegments = append(allSegments, "host", base64.URLEncoding.EncodeToString([]byte(path.Host)))
	}
	return allSegments, nil
}
func (j *innerFetchObjectRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*fetchobject.Request)(j))
}
func (j *innerFetchObjectRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*fetchobject.Request)(j))
}
func (request *innerFetchObjectRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

type FetchObjectRequest = fetchobject.Request
type FetchObjectResponse = fetchobject.Response

// 从指定 URL 抓取指定名称的对象并存储到该空间中
func (client *Client) FetchObject(ctx context.Context, request *FetchObjectRequest, options *Options) (response *FetchObjectResponse, err error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerFetchObjectRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceIo}
	var pathSegments []string
	pathSegments = append(pathSegments, "fetch")
	if segments, err := innerRequest.buildPath(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, AuthType: auth.TokenQiniu, Credentials: innerRequest.Credentials, BufferResponse: true}
	var queryer region.BucketRegionsQueryer
	if client.client.GetRegions() == nil && client.client.GetEndpoints() == nil {
		queryer = client.client.GetBucketQueryer()
		if queryer == nil {
			bucketHosts := httpclient.DefaultBucketHosts()
			var err error
			if options.OverwrittenBucketHosts != nil {
				if bucketHosts, err = options.OverwrittenBucketHosts.GetEndpoints(ctx); err != nil {
					return nil, err
				}
			}
			queryerOptions := region.BucketRegionsQueryerOptions{UseInsecureProtocol: client.client.UseInsecureProtocol(), HostFreezeDuration: client.client.GetHostFreezeDuration(), Client: client.client.GetClient()}
			if hostRetryConfig := client.client.GetHostRetryConfig(); hostRetryConfig != nil {
				queryerOptions.RetryMax = hostRetryConfig.RetryMax
			}
			if queryer, err = region.NewBucketRegionsQueryer(bucketHosts, &queryerOptions); err != nil {
				return nil, err
			}
		}
	}
	if queryer != nil {
		bucketName := options.OverwrittenBucketName
		var accessKey string
		var err error
		if bucketName == "" {
			if bucketName, err = innerRequest.getBucketName(ctx); err != nil {
				return nil, err
			}
		}
		if accessKey, err = innerRequest.getAccessKey(ctx); err != nil {
			return nil, err
		} else if accessKey == "" {
			if credentialsProvider := client.client.GetCredentials(); credentialsProvider != nil {
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
	var respBody FetchObjectResponse
	if _, err := client.client.AcceptJson(ctx, &req, &respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}
