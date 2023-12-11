// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/json"
	auth "github.com/qiniu/go-sdk/v7/auth"
	batchops "github.com/qiniu/go-sdk/v7/storagev2/apis/batch_ops"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"net/url"
	"strings"
)

type innerBatchOpsRequest batchops.Request

func (form *innerBatchOpsRequest) build() (url.Values, error) {
	formValues := make(url.Values)
	if len(form.Operations) > 0 {
		for _, value := range form.Operations {
			formValues.Add("op", value)
		}
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Operations"}
	}
	return formValues, nil
}
func (j *innerBatchOpsRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*batchops.Request)(j))
}
func (j *innerBatchOpsRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*batchops.Request)(j))
}
func (request *innerBatchOpsRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

type BatchOpsRequest = batchops.Request
type BatchOpsResponse = batchops.Response

// 批量操作意指在单一请求中执行多次（最大限制1000次） 查询元信息、修改元信息、移动、复制、删除、修改状态、修改存储类型、修改生命周期和解冻操作，极大提高对象管理效率。其中，解冻操作仅针对归档存储文件有效
func (storage *Storage) BatchOps(ctx context.Context, request *BatchOpsRequest, options *Options) (response *BatchOpsResponse, err error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerBatchOpsRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceRs}
	var pathSegments []string
	pathSegments = append(pathSegments, "batch")
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	body, err := innerRequest.build()
	if err != nil {
		return nil, err
	}
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, AuthType: auth.TokenQiniu, Credentials: innerRequest.Credentials, BufferResponse: true, RequestBody: httpclient.GetFormRequestBody(body)}
	var queryer region.BucketRegionsQueryer
	if storage.client.GetRegions() == nil && storage.client.GetEndpoints() == nil {
		queryer = storage.client.GetBucketQueryer()
		if queryer == nil {
			bucketHosts := httpclient.DefaultBucketHosts()
			var err error
			if options.OverwrittenBucketHosts != nil {
				if bucketHosts, err = options.OverwrittenBucketHosts.GetEndpoints(ctx); err != nil {
					return nil, err
				}
			}
			queryerOptions := region.BucketRegionsQueryerOptions{UseInsecureProtocol: storage.client.UseInsecureProtocol(), HostFreezeDuration: storage.client.GetHostFreezeDuration(), Client: storage.client.GetClient()}
			if hostRetryConfig := storage.client.GetHostRetryConfig(); hostRetryConfig != nil {
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
			req.Region = queryer.Query(accessKey, bucketName)
		}
	}
	var respBody BatchOpsResponse
	if err := storage.client.DoAndAcceptJSON(ctx, &req, &respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}
