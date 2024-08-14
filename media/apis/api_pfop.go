// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

package apis

import (
	"context"
	auth "github.com/qiniu/go-sdk/v7/auth"
	uplog "github.com/qiniu/go-sdk/v7/internal/uplog"
	pfop "github.com/qiniu/go-sdk/v7/media/apis/pfop"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region "github.com/qiniu/go-sdk/v7/storagev2/region"
	uptoken "github.com/qiniu/go-sdk/v7/storagev2/uptoken"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type innerPfopRequest pfop.Request

func (form *innerPfopRequest) build() (url.Values, error) {
	formValues := make(url.Values)
	if form.BucketName != "" {
		formValues.Set("bucket", form.BucketName)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "BucketName"}
	}
	if form.ObjectName != "" {
		formValues.Set("key", form.ObjectName)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "ObjectName"}
	}
	if form.Fops != "" {
		formValues.Set("fops", form.Fops)
	} else {
		return nil, errors.MissingRequiredFieldError{Name: "Fops"}
	}
	if form.NotifyUrl != "" {
		formValues.Set("notifyURL", form.NotifyUrl)
	}
	if form.Force != 0 {
		formValues.Set("force", strconv.FormatInt(form.Force, 10))
	}
	if form.Type != 0 {
		formValues.Set("type", strconv.FormatInt(form.Type, 10))
	}
	if form.Pipeline != "" {
		formValues.Set("pipeline", form.Pipeline)
	}
	return formValues, nil
}

type PfopRequest = pfop.Request
type PfopResponse = pfop.Response

// 触发持久化数据处理命令
func (media *Media) Pfop(ctx context.Context, request *PfopRequest, options *Options) (*PfopResponse, error) {
	if options == nil {
		options = &Options{}
	}
	innerRequest := (*innerPfopRequest)(request)
	serviceNames := []region.ServiceName{region.ServiceApi}
	if innerRequest.Credentials == nil && media.client.GetCredentials() == nil {
		return nil, errors.MissingRequiredFieldError{Name: "Credentials"}
	}
	pathSegments := make([]string, 0, 1)
	pathSegments = append(pathSegments, "pfop")
	path := "/" + strings.Join(pathSegments, "/")
	var rawQuery string
	body, err := innerRequest.build()
	if err != nil {
		return nil, err
	}
	uplogInterceptor, err := uplog.NewRequestUplog("pfop", "", "", func() (string, error) {
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
	req := httpclient.Request{Method: "POST", ServiceNames: serviceNames, Path: path, RawQuery: rawQuery, Endpoints: options.OverwrittenEndpoints, Region: options.OverwrittenRegion, Interceptors: []httpclient.Interceptor{uplogInterceptor}, AuthType: auth.TokenQiniu, Credentials: innerRequest.Credentials, BufferResponse: true, RequestBody: httpclient.GetFormRequestBody(body), OnRequestProgress: options.OnRequestProgress}
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
	var respBody PfopResponse
	if err := media.client.DoAndAcceptJSON(ctx, &req, &respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}
