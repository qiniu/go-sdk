// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/base64"
	"encoding/json"
	auth "github.com/qiniu/go-sdk/v7/auth"
	restorearchivedobject "github.com/qiniu/go-sdk/v7/storagev2/apis/restore_archived_object"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"strconv"
	"strings"
)

type innerRestoreArchivedObjectRequest restorearchivedobject.Request

func (pp *innerRestoreArchivedObjectRequest) getBucketName(ctx context.Context) (string, error) {
	return strings.SplitN(pp.Entry, ":", 2)[0], nil
}
func (path *innerRestoreArchivedObjectRequest) buildPath() ([]string, error) {
	var allSegments []string
	if path.Entry != "" {
		allSegments = append(allSegments, base64.URLEncoding.EncodeToString([]byte(path.Entry)))
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Entry"}
	}
	if path.FreezeAfterDays != 0 {
		allSegments = append(allSegments, "freezeAfterDays", strconv.FormatInt(path.FreezeAfterDays, 10))
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "FreezeAfterDays"}
	}
	return allSegments, nil
}
func (j *innerRestoreArchivedObjectRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*restorearchivedobject.Request)(j))
}
func (j *innerRestoreArchivedObjectRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*restorearchivedobject.Request)(j))
}
func (request *innerRestoreArchivedObjectRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

type RestoreArchivedObjectRequest = restorearchivedobject.Request
type RestoreArchivedObjectResponse = restorearchivedobject.Response

// 解冻归档存储类型的文件，可设置解冻有效期1～7天，完成解冻任务通常需要1～5分钟
func (storage *Storage) RestoreArchivedObject(ctx context.Context, request *RestoreArchivedObjectRequest, options *Options) (response *RestoreArchivedObjectResponse, err error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerRestoreArchivedObjectRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceRs}
	var pathSegments []string
	pathSegments = append(pathSegments, "restoreAr")
	if segments, err := innerRequest.buildPath(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, AuthType: auth.TokenQiniu, Credentials: innerRequest.Credentials}
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
		if bucketName == "" {
			if bucketName, err = innerRequest.getBucketName(ctx); err != nil {
				return nil, err
			}
		}
		if accessKey, err = innerRequest.getAccessKey(ctx); err != nil {
			return nil, err
		} else if accessKey == "" {
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
	resp, err := storage.client.Do(ctx, &req)
	if err != nil {
		return nil, err
	}
	return &RestoreArchivedObjectResponse{}, resp.Body.Close()
}
