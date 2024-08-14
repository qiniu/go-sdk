// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	auth "github.com/qiniu/go-sdk/v7/auth"
	uplog "github.com/qiniu/go-sdk/v7/internal/uplog"
	prefop "github.com/qiniu/go-sdk/v7/media/apis/prefop"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	uptoken "github.com/qiniu/go-sdk/v7/storagev2/uptoken"
	"net/url"
	"strings"
	"time"
)

type innerPrefopRequest prefop.Request

func (query *innerPrefopRequest) buildQuery() (url.Values, error) {
	allQuery := make(url.Values)
	if query.PersistentId != "" {
		allQuery.Set("id", query.PersistentId)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "PersistentId"}
	}
	return allQuery, nil
}

type PrefopRequest = prefop.Request
type PrefopResponse = prefop.Response

// 查询持久化数据处理命令的执行状态
func (media *Media) Prefop(ctx context.Context, request *PrefopRequest, options *Options) (*PrefopResponse, error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerPrefopRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceApi}
	if innerRequest.Credentials == nil && media.client.GetCredentials() == nil {
		return nil, errors.MissingRequiredFieldError{Name: "Credentials"}
	}
	pathSegments := make([]string, 0, 3)
	pathSegments = append(pathSegments, "status", "get", "prefop")
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	if query, err := innerRequest.buildQuery(); err != nil {
		return nil, err
	} else {
		rawQuery += query.Encode()
	}
	uplogInterceptor, err := uplog.NewRequestUplog("prefop", "", "", func() (string, error) {
		credentials := innerRequest.Credentials
		if credentials == nil {
			credentials = media.client.GetCredentials()
		}
		putPolicy, err := uptoken.NewPutPolicy("", time.Now().Add(time.Hour))
		if err != nil {
			return "", err
		}
		return uptoken.NewSigner(putPolicy, credentials).GetUpToken(ctx)
	})
	if err != nil {
		return nil, err
	}
	req := httpclient.Request{Method: "GET", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, Endpoints: options.OverwrittenEndpoints, Region: options.OverwrittenRegion, Interceptors: []httpclient.Interceptor{uplogInterceptor}, AuthType: auth.TokenQiniu, Credentials: innerRequest.Credentials, BufferResponse: true, OnRequestProgress: options.OnRequestProgress}
	if options.OverwrittenEndpoints == nil && options.OverwrittenRegion == nil && media.client.GetRegions() == nil {
		bucketHosts := httpclient.DefaultBucketHosts()

		req.Region = media.client.GetAllRegions()
		if req.Region == nil {
			if options.OverwrittenBucketHosts != nil {
				if bucketHosts, err = options.OverwrittenBucketHosts.GetEndpoints(ctx); err != nil {
					return nil, err
				}
			}
			allRegionsOptions := region.AllRegionsProviderOptions{UseInsecureProtocol: media.client.UseInsecureProtocol(), HostFreezeDuration: media.client.GetHostFreezeDuration(), Resolver: media.client.GetResolver(), Chooser: media.client.GetChooser(), BeforeSign: media.client.GetBeforeSignCallback(), AfterSign: media.client.GetAfterSignCallback(), SignError: media.client.GetSignErrorCallback(), BeforeResolve: media.client.GetBeforeResolveCallback(), AfterResolve: media.client.GetAfterResolveCallback(), ResolveError: media.client.GetResolveErrorCallback(), BeforeBackoff: media.client.GetBeforeBackoffCallback(), AfterBackoff: media.client.GetAfterBackoffCallback(), BeforeRequest: media.client.GetBeforeRequestCallback(), AfterResponse: media.client.GetAfterResponseCallback()}
			if hostRetryConfig := media.client.GetHostRetryConfig(); hostRetryConfig != nil {
				allRegionsOptions.RetryMax = hostRetryConfig.RetryMax
				allRegionsOptions.Backoff = hostRetryConfig.Backoff
			}
			credentials := innerRequest.Credentials
			if credentials == nil {
				credentials = media.client.GetCredentials()
			}
			if req.Region, err = region.NewAllRegionsProvider(credentials, bucketHosts, &allRegionsOptions); err != nil {
				return nil, err
			}
		}
	}
	var respBody PrefopResponse
	if err := media.client.DoAndAcceptJSON(ctx, &req, &respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}
