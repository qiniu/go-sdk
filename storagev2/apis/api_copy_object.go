// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/base64"
	auth "github.com/qiniu/go-sdk/v7/auth"
	copyobject "github.com/qiniu/go-sdk/v7/storagev2/apis/copy_object"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	uplog "github.com/qiniu/go-sdk/v7/storagev2/internal/uplog"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	uptoken "github.com/qiniu/go-sdk/v7/storagev2/uptoken"
	"strconv"
	"strings"
	"time"
)

type innerCopyObjectRequest copyobject.Request

func (pp *innerCopyObjectRequest) getBucketName(ctx context.Context) (string, error) {
	return strings.SplitN(pp.SrcEntry, ":", 2)[0], nil
}
func (pp *innerCopyObjectRequest) getObjectName() string {
	parts := strings.SplitN(pp.SrcEntry, ":", 2)
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}
func (path *innerCopyObjectRequest) buildPath() ([]string, error) {
	var allSegments []string
	if path.SrcEntry != "" {
		allSegments = append(allSegments, base64.URLEncoding.EncodeToString([]byte(path.SrcEntry)))
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "SrcEntry"}
	}
	if path.DestEntry != "" {
		allSegments = append(allSegments, base64.URLEncoding.EncodeToString([]byte(path.DestEntry)))
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "DestEntry"}
	}
	if path.IsForce {
		allSegments = append(allSegments, "force", strconv.FormatBool(path.IsForce))
	}
	return allSegments, nil
}
func (request *innerCopyObjectRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

type CopyObjectRequest = copyobject.Request
type CopyObjectResponse = copyobject.Response

// 将源空间的指定对象复制到目标空间
func (storage *Storage) CopyObject(ctx context.Context, request *CopyObjectRequest, options *Options) (*CopyObjectResponse, error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerCopyObjectRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceRs}
	if innerRequest.Credentials == nil && storage.client.GetCredentials() == nil {
		return nil, errors.MissingRequiredFieldError{Name: "Credentials"}
	}
	var pathSegments []string
	pathSegments = append(pathSegments, "copy")
	if segments, err := innerRequest.buildPath(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	bucketName := options.OverwrittenBucketName
	if bucketName == "" {
		var err error
		if bucketName, err = innerRequest.getBucketName(ctx); err != nil {
			return nil, err
		}
	}
	objectName := innerRequest.getObjectName()
	uplogInterceptor, err := uplog.NewRequestUplog("copyObject", bucketName, objectName, func() (string, error) {
		credentials := innerRequest.Credentials
		if credentials == nil {
			credentials = storage.client.GetCredentials()
		}
		putPolicy, err := uptoken.NewPutPolicy(bucketName, time.Now().Add(time.Hour))
		if err != nil {
			return "", err
		}
		return uptoken.NewSigner(putPolicy, credentials).GetUpToken(ctx)
	})
	if err != nil {
		return nil, err
	}
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, Endpoints: options.OverwrittenEndpoints, Region: options.OverwrittenRegion, Interceptors: []httpclient.Interceptor{uplogInterceptor}, AuthType: auth.TokenQiniu, Credentials: innerRequest.Credentials, OnRequestProgress: options.OnRequestProgress}
	if options.OverwrittenEndpoints == nil && options.OverwrittenRegion == nil && storage.client.GetRegions() == nil {
		query := storage.client.GetBucketQuery()
		if query == nil {
			bucketHosts := httpclient.DefaultBucketHosts()
			if options.OverwrittenBucketHosts != nil {
				if bucketHosts, err = options.OverwrittenBucketHosts.GetEndpoints(ctx); err != nil {
					return nil, err
				}
			}
			queryOptions := region.BucketRegionsQueryOptions{UseInsecureProtocol: storage.client.UseInsecureProtocol(), HostFreezeDuration: storage.client.GetHostFreezeDuration(), Client: storage.client.GetClient()}
			if hostRetryConfig := storage.client.GetHostRetryConfig(); hostRetryConfig != nil {
				queryOptions.RetryMax = hostRetryConfig.RetryMax
			}
			if query, err = region.NewBucketRegionsQuery(bucketHosts, &queryOptions); err != nil {
				return nil, err
			}
		}
		if query != nil {
			var accessKey string
			var err error
			if accessKey, err = innerRequest.getAccessKey(ctx); err != nil {
				return nil, err
			}
			if accessKey == "" {
				if credentialsProvider := storage.client.GetCredentials(); credentialsProvider != nil {
					if creds, err := credentialsProvider.Get(ctx); err != nil {
						return nil, err
					} else if creds != nil {
						accessKey = creds.AccessKey
					}
				}
			}
			if accessKey != "" && bucketName != "" {
				req.Region = query.Query(accessKey, bucketName)
			}
		}
	}
	resp, err := storage.client.Do(ctx, &req)
	if err != nil {
		return nil, err
	}
	return &CopyObjectResponse{}, resp.Body.Close()
}
