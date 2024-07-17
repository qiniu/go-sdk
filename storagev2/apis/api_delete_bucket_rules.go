// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	auth "github.com/qiniu/go-sdk/v7/auth"
	uplog "github.com/qiniu/go-sdk/v7/internal/uplog"
	deletebucketrules "github.com/qiniu/go-sdk/v7/storagev2/apis/delete_bucket_rules"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	uptoken "github.com/qiniu/go-sdk/v7/storagev2/uptoken"
	"net/url"
	"strings"
	"time"
)

type innerDeleteBucketRulesRequest deletebucketrules.Request

func (form *innerDeleteBucketRulesRequest) getBucketName(ctx context.Context) (string, error) {
	return form.Bucket, nil
}
func (form *innerDeleteBucketRulesRequest) build() (url.Values, error) {
	formValues := make(url.Values)
	if form.Bucket != "" {
		formValues.Set("bucket", form.Bucket)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Bucket"}
	}
	if form.Name != "" {
		formValues.Set("name", form.Name)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Name"}
	}
	return formValues, nil
}
func (request *innerDeleteBucketRulesRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

type DeleteBucketRulesRequest = deletebucketrules.Request
type DeleteBucketRulesResponse = deletebucketrules.Response

// 删除空间规则
func (storage *Storage) DeleteBucketRules(ctx context.Context, request *DeleteBucketRulesRequest, options *Options) (*DeleteBucketRulesResponse, error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerDeleteBucketRulesRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceBucket}
	if innerRequest.Credentials == nil && storage.client.GetCredentials() == nil {
		return nil, errors.MissingRequiredFieldError{Name: "Credentials"}
	}
	var pathSegments []string
	pathSegments = append(pathSegments, "rules", "delete")
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	body, err := innerRequest.build()
	if err != nil {
		return nil, err
	}
	bucketName := options.OverwrittenBucketName
	if bucketName == "" {
		var err error
		if bucketName, err = innerRequest.getBucketName(ctx); err != nil {
			return nil, err
		}
	}
	var objectName string
	uplogInterceptor, err := uplog.NewRequestUplog("deleteBucketRules", bucketName, objectName, func() (string, error) {
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
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, Endpoints: options.OverwrittenEndpoints, Region: options.OverwrittenRegion, Interceptors: []httpclient.Interceptor{uplogInterceptor}, AuthType: auth.TokenQiniu, Credentials: innerRequest.Credentials, RequestBody: httpclient.GetFormRequestBody(body), OnRequestProgress: options.OnRequestProgress}
	if options.OverwrittenEndpoints == nil && options.OverwrittenRegion == nil && storage.client.GetRegions() == nil {
		query := storage.client.GetBucketQuery()
		if query == nil {
			bucketHosts := httpclient.DefaultBucketHosts()
			if options.OverwrittenBucketHosts != nil {
				req.Endpoints = options.OverwrittenBucketHosts
			} else {
				req.Endpoints = bucketHosts
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
	return &DeleteBucketRulesResponse{}, resp.Body.Close()
}
