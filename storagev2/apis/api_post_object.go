// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	postobject "github.com/qiniu/go-sdk/v7/storagev2/apis/post_object"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	uplog "github.com/qiniu/go-sdk/v7/storagev2/internal/uplog"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"strconv"
	"strings"
)

type innerPostObjectRequest postobject.Request

func (form *innerPostObjectRequest) getBucketName(ctx context.Context) (string, error) {
	putPolicy, err := form.UploadToken.GetPutPolicy(ctx)
	if err != nil {
		return "", err
	} else {
		return putPolicy.GetBucketName()
	}
}
func (form *innerPostObjectRequest) getObjectName() string {
	var objectName string
	if form.ObjectName != nil {
		objectName = *form.ObjectName
	}
	return objectName
}
func (form *innerPostObjectRequest) build(ctx context.Context) (*httpclient.MultipartForm, error) {
	multipartForm := new(httpclient.MultipartForm)
	if form.ObjectName != nil {
		multipartForm.SetValue("key", *form.ObjectName)
	}
	if form.UploadToken != nil {
		upToken, err := form.UploadToken.GetUpToken(ctx)
		if err != nil {
			return nil, err
		}
		multipartForm.SetValue("token", upToken)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "UploadToken"}
	}
	if form.Crc32 != 0 {
		multipartForm.SetValue("crc32", strconv.FormatInt(form.Crc32, 10))
	}
	if form.File.Data != nil {
		if form.File.Name == "" {
			return nil, errors.MissingRequiredFieldError{Name: "File.Name"}
		}
		multipartForm.SetFile("file", form.File.Name, form.File.ContentType, form.File.Data)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "File"}
	}
	for key, value := range form.CustomData {
		multipartForm.SetValue(key, value)
	}
	return multipartForm, nil
}
func (request *innerPostObjectRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.UploadToken != nil {
		if accessKey, err := request.UploadToken.GetAccessKey(ctx); err != nil {
			return "", err
		} else {
			return accessKey, nil
		}
	}
	return "", nil
}

type PostObjectRequest = postobject.Request
type PostObjectResponse = postobject.Response

// 在一次 HTTP 会话中上传单一的一个文件
func (storage *Storage) PostObject(ctx context.Context, request *PostObjectRequest, options *Options) (*PostObjectResponse, error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerPostObjectRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceUp}
	var pathSegments []string
	pathSegments = append(pathSegments, "")
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	body, err := innerRequest.build(ctx)
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
	objectName := innerRequest.getObjectName()
	uplogInterceptor, err := uplog.NewRequestUplog("postObject", bucketName, objectName, nil)
	if err != nil {
		return nil, err
	}
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, Endpoints: options.OverwrittenEndpoints, Region: options.OverwrittenRegion, Interceptors: []httpclient.Interceptor{uplogInterceptor}, BufferResponse: true, RequestBody: httpclient.GetMultipartFormRequestBody(body), OnRequestProgress: options.OnRequestProgress}
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
	ctx = httpclient.WithoutSignature(ctx)
	respBody := PostObjectResponse{Body: innerRequest.ResponseBody}
	if err := storage.client.DoAndAcceptJSON(ctx, &req, &respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}
