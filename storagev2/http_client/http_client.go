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
	"github.com/qiniu/go-sdk/v7/storagev2/chooser"
	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/defaults"
	"github.com/qiniu/go-sdk/v7/storagev2/region"
	"github.com/qiniu/go-sdk/v7/storagev2/resolver"
	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"
)

var (
	ErrNoRegion              = errors.New("no region from provider")
	ErrNoEndpointsConfigured = errors.New("no endpoints configured")
)

type (
	InterceptorPriority = clientv2.InterceptorPriority
	Interceptor         = clientv2.Interceptor
	BasicHTTPClient     = clientv2.Client
	GetRequestBody      = clientv2.GetRequestBody
	RetryConfig         = clientv2.RetryConfig
	Handler             = clientv2.Handler

	// Client 提供了对七牛 HTTP 客户端
	Client struct {
		useHttps           bool
		basicHTTPClient    BasicHTTPClient
		bucketQuery        region.BucketRegionsQuery
		regions            region.RegionsProvider
		credentials        credentials.CredentialsProvider
		resolver           resolver.Resolver
		chooser            chooser.Chooser
		hostRetryConfig    *RetryConfig
		hostsRetryConfig   *RetryConfig
		hostFreezeDuration time.Duration
		shouldFreezeHost   func(req *http.Request, resp *http.Response, err error) bool
	}

	// Options 为构建 Client 提供了可选参数
	Options struct {
		BasicHTTPClient     BasicHTTPClient
		BucketQuery         region.BucketRegionsQuery
		Regions             region.RegionsProvider
		Credentials         credentials.CredentialsProvider
		Interceptors        []Interceptor
		UseInsecureProtocol bool
		Resolver            resolver.Resolver
		Chooser             chooser.Chooser
		HostRetryConfig     *RetryConfig
		HostsRetryConfig    *RetryConfig
		HostFreezeDuration  time.Duration
		ShouldFreezeHost    func(req *http.Request, resp *http.Response, err error) bool
	}

	// Request 包含一个具体的 HTTP 请求的参数
	Request struct {
		Method         string
		ServiceNames   []region.ServiceName
		Endpoints      region.EndpointsProvider
		Region         region.RegionsProvider
		Path           string
		RawQuery       string
		Query          url.Values
		Header         http.Header
		RequestBody    GetRequestBody
		Credentials    credentials.CredentialsProvider
		AuthType       auth.TokenType
		UpToken        uptoken.UpTokenProvider
		BufferResponse bool
	}
)

// NewClient 用来构建一个新的七牛 HTTP 客户端
func NewClient(options *Options) *Client {
	if options == nil {
		options = &Options{}
		if isDisabled, err := defaults.DisableSecureProtocol(); err == nil {
			options.UseInsecureProtocol = isDisabled
		}
	}
	if options.HostFreezeDuration < time.Millisecond {
		options.HostFreezeDuration = 600 * time.Second
	}
	if options.ShouldFreezeHost == nil {
		options.ShouldFreezeHost = defaultShouldFreezeHost
	}
	if options.Credentials == nil {
		options.Credentials = auth.Default()
	}

	return &Client{
		basicHTTPClient:    clientv2.NewClient(options.BasicHTTPClient, options.Interceptors...),
		useHttps:           !options.UseInsecureProtocol,
		bucketQuery:        options.BucketQuery,
		regions:            options.Regions,
		credentials:        options.Credentials,
		resolver:           options.Resolver,
		chooser:            options.Chooser,
		hostRetryConfig:    options.HostRetryConfig,
		hostsRetryConfig:   options.HostsRetryConfig,
		hostFreezeDuration: options.HostFreezeDuration,
		shouldFreezeHost:   options.ShouldFreezeHost,
	}
}

// Do 发送 HTTP 请求
func (httpClient *Client) Do(ctx context.Context, request *Request) (*http.Response, error) {
	req, err := httpClient.makeReq(ctx, request)
	if err != nil {
		return nil, err
	}
	if upTokenProvider := request.UpToken; upTokenProvider != nil {
		if upToken, err := upTokenProvider.GetUpToken(ctx); err != nil {
			return nil, err
		} else {
			req.Header.Set("Authorization", "UpToken "+upToken)
		}
	} else {
		credentialsProvider := request.Credentials
		if credentialsProvider == nil {
			credentialsProvider = httpClient.credentials
		}
		if credentialsProvider != nil {
			if credentials, err := credentialsProvider.Get(ctx); err != nil {
				return nil, err
			} else {
				req = clientv2.WithInterceptors(req, clientv2.NewAuthInterceptor(clientv2.AuthConfig{
					Credentials: credentials,
					TokenType:   request.AuthType,
				}))
			}
		}
	}
	return httpClient.basicHTTPClient.Do(req)
}

// DoAndAcceptJSON 发送 HTTP 请求并接收 JSON 响应
func (httpClient *Client) DoAndAcceptJSON(ctx context.Context, request *Request, ret interface{}) error {
	if resp, err := httpClient.Do(ctx, request); err != nil {
		return err
	} else {
		return clientv1.DecodeJsonFromReader(resp.Body, ret)
	}
}

func (httpClient *Client) GetBucketQuery() region.BucketRegionsQuery {
	return httpClient.bucketQuery
}

func (httpClient *Client) GetCredentials() credentials.CredentialsProvider {
	return httpClient.credentials
}

func (httpClient *Client) GetRegions() region.RegionsProvider {
	return httpClient.regions
}

func (httpClient *Client) GetClient() BasicHTTPClient {
	return httpClient.basicHTTPClient
}

func (httpClient *Client) UseInsecureProtocol() bool {
	return !httpClient.useHttps
}

func (httpClient *Client) GetHostFreezeDuration() time.Duration {
	return httpClient.hostFreezeDuration
}

func (httpClient *Client) GetHostRetryConfig() *RetryConfig {
	return httpClient.hostRetryConfig
}

func (httpClient *Client) GetHostsRetryConfig() *RetryConfig {
	return httpClient.hostsRetryConfig
}

func (httpClient *Client) getEndpoints(ctx context.Context, request *Request) (region.Endpoints, error) {
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
	} else if httpClient.regions != nil && len(request.ServiceNames) > 0 {
		return getEndpointsFromRegionsProvider(ctx, httpClient.regions, request.ServiceNames)
	}
	return region.Endpoints{}, ErrNoEndpointsConfigured
}

func (httpClient *Client) makeReq(ctx context.Context, request *Request) (*http.Request, error) {
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
		hostsRetryConfig = &RetryConfig{
			RetryMax: len(endpoints.Preferred) + len(endpoints.Alternative),
		}
	}
	r := httpClient.resolver
	if r == nil {
		if r, err = resolver.NewCacheResolver(nil, nil); err != nil {
			return nil, err
		}
	}
	cs := httpClient.chooser
	if cs == nil {
		cs = chooser.NewShuffleChooser(chooser.NewSmartIPChooser(nil))
	}
	interceptors = append(interceptors, clientv2.NewHostsRetryInterceptor(clientv2.HostsRetryConfig{
		RetryMax:           hostsRetryConfig.RetryMax,
		ShouldRetry:        hostsRetryConfig.ShouldRetry,
		HostFreezeDuration: httpClient.hostFreezeDuration,
		HostProvider:       hostProvider,
		ShouldFreezeHost:   httpClient.shouldFreezeHost,
	}))
	if httpClient.hostRetryConfig != nil {
		interceptors = append(interceptors, clientv2.NewSimpleRetryInterceptor(
			clientv2.SimpleRetryConfig{
				RetryMax:      httpClient.hostRetryConfig.RetryMax,
				RetryInterval: httpClient.hostRetryConfig.RetryInterval,
				Backoff:       httpClient.hostRetryConfig.Backoff,
				ShouldRetry:   httpClient.hostRetryConfig.ShouldRetry,
				Resolver:      r,
				Chooser:       cs,
			},
		))
	}
	req, err := clientv2.NewRequest(clientv2.RequestParams{
		Context:        ctx,
		Method:         request.Method,
		Url:            url,
		Header:         request.Header,
		GetBody:        request.RequestBody,
		BufferResponse: request.BufferResponse,
	})
	if err != nil {
		return nil, err
	}
	return clientv2.WithInterceptors(req, interceptors...), nil
}

func (httpClient *Client) generateUrl(request *Request, hostProvider hostprovider.HostProvider) (string, error) {
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

func (options *Options) SetBucketHosts(bucketHosts region.Endpoints) (err error) {
	options.BucketQuery, err = region.NewBucketRegionsQuery(bucketHosts, nil)
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

var defaultBucketHosts = region.Endpoints{
	Preferred:   []string{"uc.qiniuapi.com", "kodo-config.qiniuapi.com"},
	Alternative: []string{"uc.qbox.me"},
}

func init() {
	if bucketUrls, err := defaults.BucketURLs(); err == nil && len(bucketUrls) > 0 {
		defaultBucketHosts = region.Endpoints{Preferred: bucketUrls}
	}
}

// DefaultBucketHosts 默认的 Bucket 域名列表
func DefaultBucketHosts() region.Endpoints {
	return defaultBucketHosts.Clone()
}

func defaultShouldFreezeHost(*http.Request, *http.Response, error) bool {
	return true
}
