// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	auth "github.com/qiniu/go-sdk/v7/auth"
	addbucketrules "github.com/qiniu/go-sdk/v7/storagev2/apis/add_bucket_rules"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"net/url"
	"strconv"
	"strings"
)

type innerAddBucketRulesRequest addbucketrules.Request

func (form *innerAddBucketRulesRequest) getBucketName(ctx context.Context) (string, error) {
	return form.Bucket, nil
}
func (form *innerAddBucketRulesRequest) build() (url.Values, error) {
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
	formValues.Set("prefix", form.Prefix)
	formValues.Set("delete_after_days", strconv.FormatInt(form.DeleteAfterDays, 10))
	formValues.Set("to_line_after_days", strconv.FormatInt(form.ToIaAfterDays, 10))
	formValues.Set("to_archive_after_days", strconv.FormatInt(form.ToArchiveAfterDays, 10))
	formValues.Set("to_deep_archive_after_days", strconv.FormatInt(form.ToDeepArchiveAfterDays, 10))
	formValues.Set("to_archive_ir_after_days", strconv.FormatInt(form.ToArchiveIrAfterDays, 10))
	return formValues, nil
}
func (request *innerAddBucketRulesRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

type AddBucketRulesRequest = addbucketrules.Request
type AddBucketRulesResponse = addbucketrules.Response

// 增加空间规则
func (storage *Storage) AddBucketRules(ctx context.Context, request *AddBucketRulesRequest, options *Options) (*AddBucketRulesResponse, error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerAddBucketRulesRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceBucket}
	var pathSegments []string
	pathSegments = append(pathSegments, "rules", "add")
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	body, err := innerRequest.build()
	if err != nil {
		return nil, err
	}
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, Endpoints: options.OverwrittenEndpoints, Region: options.OverwrittenRegion, AuthType: auth.TokenQiniu, Credentials: innerRequest.Credentials, RequestBody: httpclient.GetFormRequestBody(body)}
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
	return &AddBucketRulesResponse{}, resp.Body.Close()
}
