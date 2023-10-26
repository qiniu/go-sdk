package http_client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/qiniu/go-sdk/v7/auth"
	clientv1 "github.com/qiniu/go-sdk/v7/client"
	"github.com/qiniu/go-sdk/v7/internal/clientv2"
	"github.com/qiniu/go-sdk/v7/internal/hostprovider"
	compatible_io "github.com/qiniu/go-sdk/v7/internal/io"
	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/region"
	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"
)

var (
	ErrNoRegion              = errors.New("no region from provider")
	ErrNoEndpointsConfigured = errors.New("no endpoints configured")
)

type (
	InterceptorPriority = clientv2.InterceptorPriority
	Interceptor         = clientv2.Interceptor
	Client              = clientv2.Client
	GetRequestBody      = clientv2.GetRequestBody

	// HttpClient 提供了对七牛 HTTP 客户端
	HttpClient struct {
		useHttps           bool
		client             Client
		bucketQueryer      *region.BucketRegionsQueryer
		endpoints          region.EndpointsProvider
		regions            region.RegionsProvider
		credentials        credentials.CredentialsProvider
		hostRetryConfig    *clientv2.RetryConfig
		hostsRetryConfig   *clientv2.RetryConfig
		hostFreezeDuration time.Duration
		shouldFreezeHost   func(req *http.Request, resp *http.Response, err error) bool
	}

	// HttpClientOptions 为构建 HttpClient 提供了可选参数
	HttpClientOptions struct {
		Client             Client
		BucketQueryer      *region.BucketRegionsQueryer
		Endpoints          region.EndpointsProvider
		Regions            region.RegionsProvider
		Credentials        credentials.CredentialsProvider
		Interceptors       []Interceptor
		UseHttps           bool
		HostRetryConfig    *clientv2.RetryConfig
		HostsRetryConfig   *clientv2.RetryConfig
		HostFreezeDuration time.Duration
		ShouldFreezeHost   func(req *http.Request, resp *http.Response, err error) bool
	}

	Request struct {
		Method       string
		ServiceNames []region.ServiceName
		Endpoints    region.EndpointsProvider
		Region       region.RegionsProvider
		Path         string
		RawQuery     string
		Query        url.Values
		Header       http.Header
		RequestBody  GetRequestBody
		Credentials  credentials.CredentialsProvider
		AuthType     auth.TokenType
		UpToken      uptoken.Retriever
	}
)

// NewHttpClient 用来构建一个新的七牛 HTTP 客户端
func NewHttpClient(options *HttpClientOptions) *HttpClient {
	if options == nil {
		options = &HttpClientOptions{}
	}

	if options.HostFreezeDuration < time.Millisecond {
		options.HostFreezeDuration = 600 * time.Second
	}
	if options.ShouldFreezeHost == nil {
		options.ShouldFreezeHost = func(req *http.Request, resp *http.Response, err error) bool {
			return true
		}
	}

	return &HttpClient{
		client:             clientv2.NewClient(options.Client, options.Interceptors...),
		useHttps:           options.UseHttps,
		bucketQueryer:      options.BucketQueryer,
		endpoints:          options.Endpoints,
		regions:            options.Regions,
		credentials:        options.Credentials,
		hostRetryConfig:    options.HostRetryConfig,
		hostsRetryConfig:   options.HostsRetryConfig,
		hostFreezeDuration: options.HostFreezeDuration,
		shouldFreezeHost:   options.ShouldFreezeHost,
	}
}

// Do 发送 HTTP 请求
func (httpClient *HttpClient) Do(ctx context.Context, request *Request) (*http.Response, error) {
	req, err := httpClient.makeReq(ctx, request)
	if err != nil {
		return nil, err
	}
	credentialsProvider := request.Credentials
	if credentialsProvider == nil {
		credentialsProvider = httpClient.credentials
	}
	if credentialsProvider != nil {
		if credentials, err := credentialsProvider.Get(ctx); err != nil {
			return nil, err
		} else if err = credentials.AddToken(request.AuthType, req); err != nil {
			return nil, err
		}
	} else if request.UpToken != nil {
		if upToken, err := request.UpToken.RetrieveUpToken(ctx); err != nil {
			return nil, err
		} else {
			req.Header.Set("Authorization", "UpToken "+upToken)
		}
	}
	return httpClient.client.Do(req)
}

// AcceptJson 发送 HTTP 请求并接收 JSON 响应
func (httpClient *HttpClient) AcceptJson(ctx context.Context, request *Request, ret interface{}) (*http.Response, error) {
	resp, err := httpClient.Do(ctx, request)
	if err != nil {
		return resp, err
	}
	if ret == nil || resp.ContentLength == 0 {
		return resp, nil
	}
	if err = clientv1.DecodeJsonFromReader(resp.Body, ret); err != nil {
		return resp, err
	}
	return resp, nil
}

func (httpClient *HttpClient) GetBucketQueryer() *region.BucketRegionsQueryer {
	return httpClient.bucketQueryer
}

func (httpClient *HttpClient) GetEndpoints() region.EndpointsProvider {
	return httpClient.endpoints
}

func (httpClient *HttpClient) GetRegions() region.RegionsProvider {
	return httpClient.regions
}

func (httpClient *HttpClient) getEndpoints(ctx context.Context, request *Request) (region.Endpoints, error) {
	getEndpointsFromEndpointsProvider := func(ctx context.Context, endpoints region.EndpointsProvider) (region.Endpoints, error) {
		return endpoints.GetEndpoints(ctx)
	}
	getEndpointsFromRegionsProvider := func(ctx context.Context, regions region.RegionsProvider, serviceNames []region.ServiceName) (region.Endpoints, error) {
		rs, err := regions.GetRegions(ctx)
		if err != nil {
			return region.Endpoints{}, err
		} else if len(rs) == 0 {
			return region.Endpoints{}, ErrNoRegion
		}
		r := rs[0]
		return r.Endpoints(request.ServiceNames)
	}
	if request.Endpoints != nil {
		return getEndpointsFromEndpointsProvider(ctx, request.Endpoints)
	} else if request.Region != nil && len(request.ServiceNames) > 0 {
		return getEndpointsFromRegionsProvider(ctx, request.Region, request.ServiceNames)
	} else if httpClient.endpoints != nil {
		return getEndpointsFromEndpointsProvider(ctx, httpClient.endpoints)
	} else if httpClient.regions != nil && len(request.ServiceNames) > 0 {
		return getEndpointsFromRegionsProvider(ctx, httpClient.regions, request.ServiceNames)
	}
	return region.Endpoints{}, ErrNoEndpointsConfigured
}

func (httpClient *HttpClient) makeReq(ctx context.Context, request *Request) (*http.Request, error) {
	endpoints, err := httpClient.getEndpoints(ctx, request)
	if err != nil {
		return nil, err
	}
	hostProvider := endpoints.ToHostProvider()
	url, err := httpClient.generateUrl(request, hostProvider)
	if err != nil {
		return nil, err
	}

	interceptors := make([]Interceptor, 0, 2)
	hostsRetryConfig := httpClient.hostsRetryConfig
	if hostsRetryConfig == nil {
		hostsRetryConfig = &clientv2.RetryConfig{
			RetryMax: len(endpoints.Preferred) + len(endpoints.Alternative),
		}
	}
	interceptors = append(interceptors, clientv2.NewHostsRetryInterceptor(clientv2.HostsRetryConfig{
		RetryConfig:        *hostsRetryConfig,
		HostProvider:       hostProvider,
		HostFreezeDuration: httpClient.hostFreezeDuration,
		ShouldFreezeHost:   httpClient.shouldFreezeHost,
	}))
	if httpClient.hostRetryConfig != nil {
		interceptors = append(interceptors, clientv2.NewSimpleRetryInterceptor(*httpClient.hostRetryConfig))
	}
	req, err := clientv2.NewRequest(clientv2.RequestParams{
		Context: ctx,
		Method:  request.Method,
		Url:     url,
		Header:  request.Header,
		GetBody: request.RequestBody,
	})
	if err != nil {
		return nil, err
	}
	return clientv2.WithInterceptors(req, interceptors...), nil
}

func (httpClient *HttpClient) generateUrl(request *Request, hostProvider hostprovider.HostProvider) (string, error) {
	var url string
	host, err := hostProvider.Provider()
	if err != nil {
		return "", err
	}
	if strings.Contains(host, "://") {
		url = host
	} else {
		if httpClient.useHttps {
			url = "https://"
		} else {
			url = "http://"
		}
		url += host
	}
	if !strings.HasPrefix(request.Path, "/") {
		url += "/"
	}
	url += request.Path
	if request.RawQuery != "" || request.Query != nil {
		url += "?"
		var rawQuery string
		if request.RawQuery != "" {
			rawQuery = request.RawQuery
		}
		if request.Query != nil {
			if rawQuery != "" {
				rawQuery += "&"
			}
			rawQuery += request.Query.Encode()
		}
		url += rawQuery
	}
	return url, nil
}

func (options *HttpClientOptions) SetBucketHosts(bucketHosts region.Endpoints) (err error) {
	options.BucketQueryer, err = region.NewBucketRegionsQueryer(bucketHosts, nil)
	return
}

// GetFormRequestBody 将数据通过 Form 作为请求 Body 发送
func GetFormRequestBody(info map[string][]string) GetRequestBody {
	return clientv2.GetFormRequestBody(info)
}

// GetJsonRequestBody 将数据通过 JSON 作为请求 Body 发送
func GetJsonRequestBody(object interface{}) (GetRequestBody, error) {
	return clientv2.GetJsonRequestBody(object)
}

// MultipartForm 用来构建 Multipart 表单
type MultipartForm = clientv2.MultipartForm

// GetMultipartFormRequestBody 将数据通过 Multipart 表单作为请求 Body 发送
func GetMultipartFormRequestBody(info *MultipartForm) GetRequestBody {
	return clientv2.GetMultipartFormRequestBody(info)
}

// GetMultipartFormRequestBody 将二进制数据请求 Body 发送
func GetRequestBodyFromReadSeekCloser(r compatible_io.ReadSeekCloser) GetRequestBody {
	return func(*clientv2.RequestParams) (io.ReadCloser, error) {
		_, err := r.Seek(0, io.SeekStart)
		return r, err
	}
}

var (
	defaultBucketHosts = region.Endpoints{
		Preferred:   []string{"uc.qiniuapi.com", "kodo-config.qiniuapi.com"},
		Alternative: []string{"uc.qbox.me"},
	}
)

// DefaultBucketHosts 默认的 Bucket 域名列表
func DefaultBucketHosts() region.Endpoints {
	return defaultBucketHosts.Clone()
}
