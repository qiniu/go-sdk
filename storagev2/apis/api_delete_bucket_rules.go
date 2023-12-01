// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/json"
	auth "github.com/qiniu/go-sdk/v7/auth"
	deletebucketrules "github.com/qiniu/go-sdk/v7/storagev2/apis/delete_bucket_rules"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"net/url"
	"strings"
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
func (j *innerDeleteBucketRulesRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*deletebucketrules.Request)(j))
}
func (j *innerDeleteBucketRulesRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*deletebucketrules.Request)(j))
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
func (client *Client) DeleteBucketRules(ctx context.Context, request *DeleteBucketRulesRequest, options *Options) (response *DeleteBucketRulesResponse, err error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerDeleteBucketRulesRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceBucket}
	var pathSegments []string
	pathSegments = append(pathSegments, "rules", "delete")
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	body, err := innerRequest.build()
	if err != nil {
		return nil, err
	}
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, AuthType: auth.TokenQiniu, Credentials: innerRequest.Credentials, RequestBody: httpclient.GetFormRequestBody(body)}
	var queryer region.BucketRegionsQueryer
	if client.client.GetRegions() == nil && client.client.GetEndpoints() == nil {
		queryer = client.client.GetBucketQueryer()
		if queryer == nil {
			bucketHosts := httpclient.DefaultBucketHosts()
			if options.OverwrittenBucketHosts != nil {
				req.Endpoints = options.OverwrittenBucketHosts
			} else {
				req.Endpoints = bucketHosts
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
	resp, err := client.client.Do(ctx, &req)
	if err != nil {
		return nil, err
	}
	return &DeleteBucketRulesResponse{}, resp.Body.Close()
}
