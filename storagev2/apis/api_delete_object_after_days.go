// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	"encoding/base64"
	"encoding/json"
	auth "github.com/qiniu/go-sdk/v7/auth"
	deleteobjectafterdays "github.com/qiniu/go-sdk/v7/storagev2/apis/delete_object_after_days"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	"strconv"
	"strings"
)

type innerDeleteObjectAfterDaysRequest deleteobjectafterdays.Request

func (pp *innerDeleteObjectAfterDaysRequest) getBucketName(ctx context.Context) (string, error) {
	return strings.SplitN(pp.Entry, ":", 2)[0], nil
}
func (path *innerDeleteObjectAfterDaysRequest) buildPath() ([]string, error) {
	var allSegments []string
	if path.Entry != "" {
		allSegments = append(allSegments, base64.URLEncoding.EncodeToString([]byte(path.Entry)))
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Entry"}
	}
	if path.DeleteAfterDays != 0 {
		allSegments = append(allSegments, strconv.FormatInt(path.DeleteAfterDays, 10))
	}
	return allSegments, nil
}
func (j *innerDeleteObjectAfterDaysRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal((*deleteobjectafterdays.Request)(j))
}
func (j *innerDeleteObjectAfterDaysRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*deleteobjectafterdays.Request)(j))
}
func (request *innerDeleteObjectAfterDaysRequest) getAccessKey(ctx context.Context) (string, error) {
	if request.Credentials != nil {
		if credentials, err := request.Credentials.Get(ctx); err != nil {
			return "", err
		} else {
			return credentials.AccessKey, nil
		}
	}
	return "", nil
}

type DeleteObjectAfterDaysRequest = deleteobjectafterdays.Request
type DeleteObjectAfterDaysResponse = deleteobjectafterdays.Response

// 更新文件生命周期
func (storage *Storage) DeleteObjectAfterDays(ctx context.Context, request *DeleteObjectAfterDaysRequest, options *Options) (*DeleteObjectAfterDaysResponse, error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerDeleteObjectAfterDaysRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceRs}
	var pathSegments []string
	pathSegments = append(pathSegments, "deleteAfterDays")
	if segments, err := innerRequest.buildPath(); err != nil {
		return nil, err
	} else {
		pathSegments = append(pathSegments, segments...)
	}
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, Endpoints: options.OverwrittenEndpoints, Region: options.OverwrittenRegion, AuthType: auth.TokenQiniu, Credentials: innerRequest.Credentials}
	if options.OverwrittenEndpoints == nil && options.OverwrittenRegion == nil && storage.client.GetRegions() == nil {
		queryer := storage.client.GetBucketQueryer()
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
	}
	resp, err := storage.client.Do(ctx, &req)
	if err != nil {
		return nil, err
	}
	return &DeleteObjectAfterDaysResponse{}, resp.Body.Close()
}
