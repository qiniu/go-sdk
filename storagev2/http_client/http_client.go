package http_client

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
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
	"github.com/qiniu/go-sdk/v7/storagev2/retrier"
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
		useHttps            bool
		accelerateUploading bool
		basicHTTPClient     BasicHTTPClient
		bucketQuery         region.BucketRegionsQuery
		allRegions          region.RegionsProvider
		regions             region.RegionsProvider
		credentials         credentials.CredentialsProvider
		resolver            resolver.Resolver
		chooser             chooser.Chooser
		hostRetryConfig     *RetryConfig
		hostsRetryConfig    *RetryConfig
		hostFreezeDuration  time.Duration
		shouldFreezeHost    func(req *http.Request, resp *http.Response, err error) bool
		beforeSign          func(req *http.Request)
		afterSign           func(req *http.Request)
		signError           func(req *http.Request, err error)
		beforeResolve       func(*http.Request)
		afterResolve        func(*http.Request, []net.IP)
		resolveError        func(*http.Request, error)
		beforeBackoff       func(*http.Request, *retrier.RetrierOptions, time.Duration)
		afterBackoff        func(*http.Request, *retrier.RetrierOptions, time.Duration)
		beforeRequest       func(*http.Request, *retrier.RetrierOptions)
		afterResponse       func(*http.Response, *retrier.RetrierOptions, error)
	}

	// Options 为构建 Client 提供了可选参数
	Options struct {
		// 基础 HTTP 客户端
		BasicHTTPClient BasicHTTPClient

		// 空间区域查询器
		BucketQuery region.BucketRegionsQuery

		// 所有区域提供者
		AllRegions region.RegionsProvider

		// 区域提供者
		Regions region.RegionsProvider

		// 凭证信息提供者
		Credentials credentials.CredentialsProvider

		// 拦截器
		Interceptors []Interceptor

		// 是否使用 HTTP 协议
		UseInsecureProtocol bool

		// 域名解析器
		Resolver resolver.Resolver

		// 域名选择器
		Chooser chooser.Chooser

		// 单域名重试配置
		HostRetryConfig *RetryConfig

		// 主备域名重试配置
		HostsRetryConfig *RetryConfig

		// 主备域名冻结时间
		HostFreezeDuration time.Duration

		// 主备域名冻结判断函数
		ShouldFreezeHost func(*http.Request, *http.Response, error) bool

		// 签名前回调函数
		BeforeSign func(*http.Request)

		// 签名后回调函数
		AfterSign func(*http.Request)

		// 签名错误回调函数
		SignError func(*http.Request, error)

		// 域名解析前回调函数
		BeforeResolve func(*http.Request)

		// 域名解析后回调函数
		AfterResolve func(*http.Request, []net.IP)

		// 域名解析错误回调函数
		ResolveError func(*http.Request, error)

		// 退避前回调函数
		BeforeBackoff func(*http.Request, *retrier.RetrierOptions, time.Duration)

		// 退避后回调函数
		AfterBackoff func(*http.Request, *retrier.RetrierOptions, time.Duration)

		// 请求前回调函数
		BeforeRequest func(*http.Request, *retrier.RetrierOptions)

		// 请求后回调函数
		AfterResponse func(*http.Response, *retrier.RetrierOptions, error)

		// 是否加速上传
		AccelerateUploading bool
	}

	// Request 包含一个具体的 HTTP 请求的参数
	Request struct {
		// 请求方法
		Method string

		// 请求服务名
		ServiceNames []region.ServiceName

		// 服务地址提供者
		Endpoints region.EndpointsProvider

		// 区域提供者
		Region region.RegionsProvider

		// 请求路径
		Path string

		// 原始请求查询参数
		RawQuery string

		// 请求查询参数
		Query url.Values

		// 请求头
		Header http.Header

		// 请求 Body 获取函数
		RequestBody GetRequestBody

		// 凭证信息提供者
		Credentials credentials.CredentialsProvider

		// 授权类型
		AuthType auth.TokenType

		// 上传凭证接口
		UpToken uptoken.UpTokenProvider

		// 是否缓存响应
		BufferResponse bool

		// 拦截器追加列表
		Interceptors []Interceptor

		// 请求进度回调函数
		OnRequestProgress func(uint64, uint64)
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
	hostFreezeDuration := options.HostFreezeDuration
	if hostFreezeDuration < time.Millisecond {
		hostFreezeDuration = 600 * time.Second
	}
	shouldFreezeHost := options.ShouldFreezeHost
	if shouldFreezeHost == nil {
		shouldFreezeHost = defaultShouldFreezeHost
	}
	creds := options.Credentials
	if creds == nil {
		if defaultCreds := credentials.Default(); defaultCreds != nil {
			creds = defaultCreds
		}
	}

	return &Client{
		useHttps:            !options.UseInsecureProtocol,
		accelerateUploading: options.AccelerateUploading,
		basicHTTPClient:     clientv2.NewClient(options.BasicHTTPClient, options.Interceptors...),
		bucketQuery:         options.BucketQuery,
		allRegions:          options.AllRegions,
		regions:             options.Regions,
		credentials:         creds,
		resolver:            options.Resolver,
		chooser:             options.Chooser,
		hostRetryConfig:     options.HostRetryConfig,
		hostsRetryConfig:    options.HostsRetryConfig,
		hostFreezeDuration:  hostFreezeDuration,
		shouldFreezeHost:    shouldFreezeHost,
		beforeSign:          options.BeforeSign,
		afterSign:           options.AfterSign,
		signError:           options.SignError,
		beforeResolve:       options.BeforeResolve,
		afterResolve:        options.AfterResolve,
		resolveError:        options.ResolveError,
		beforeBackoff:       options.BeforeBackoff,
		afterBackoff:        options.AfterBackoff,
		beforeRequest:       options.BeforeRequest,
		afterResponse:       options.AfterResponse,
	}
}

// Do 发送 HTTP 请求
func (httpClient *Client) Do(ctx context.Context, request *Request) (*http.Response, error) {
	req, err := httpClient.makeReq(ctx, request)
	if err != nil {
		return nil, err
	}
	req = clientv2.WithInterceptors(req, clientv2.NewAntiHijackingInterceptor())
	if !isSignatureDisabled(ctx) {
		if upTokenProvider := request.UpToken; upTokenProvider != nil {
			req = clientv2.WithInterceptors(req, clientv2.NewUpTokenInterceptor(clientv2.UpTokenConfig{
				UpToken: upTokenProvider,
			}))
		} else {
			credentialsProvider := request.Credentials
			if credentialsProvider == nil {
				credentialsProvider = httpClient.credentials
			}
			if credentialsProvider != nil {
				req = clientv2.WithInterceptors(req, clientv2.NewAuthInterceptor(clientv2.AuthConfig{
					Credentials: credentialsProvider,
					TokenType:   request.AuthType,
					BeforeSign:  httpClient.beforeSign,
					AfterSign:   httpClient.afterSign,
					SignError:   httpClient.signError,
				}))
			}
		}
	}
	if len(request.Interceptors) > 0 {
		req = clientv2.WithInterceptors(req, request.Interceptors...)
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

func (httpClient *Client) AccelerateUploadingEnabled() bool {
	return httpClient.accelerateUploading
}

func (httpClient *Client) GetBucketQuery() region.BucketRegionsQuery {
	return httpClient.bucketQuery
}

func (httpClient *Client) GetCredentials() credentials.CredentialsProvider {
	return httpClient.credentials
}

func (httpClient *Client) GetAllRegions() region.RegionsProvider {
	return httpClient.allRegions
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

func (httpClient *Client) GetResolver() resolver.Resolver {
	return httpClient.resolver
}

func (httpClient *Client) GetChooser() chooser.Chooser {
	return httpClient.chooser
}

func (httpClient *Client) GetBeforeSignCallback() func(*http.Request) {
	return httpClient.beforeSign
}

func (httpClient *Client) GetAfterSignCallback() func(*http.Request) {
	return httpClient.afterSign
}

func (httpClient *Client) GetSignErrorCallback() func(*http.Request, error) {
	return httpClient.signError
}

func (httpClient *Client) GetBeforeResolveCallback() func(*http.Request) {
	return httpClient.beforeResolve
}

func (httpClient *Client) GetAfterResolveCallback() func(*http.Request, []net.IP) {
	return httpClient.afterResolve
}

func (httpClient *Client) GetResolveErrorCallback() func(*http.Request, error) {
	return httpClient.resolveError
}

func (httpClient *Client) GetBeforeBackoffCallback() func(*http.Request, *retrier.RetrierOptions, time.Duration) {
	return httpClient.beforeBackoff
}

func (httpClient *Client) GetAfterBackoffCallback() func(*http.Request, *retrier.RetrierOptions, time.Duration) {
	return httpClient.afterBackoff
}

func (httpClient *Client) GetBeforeRequestCallback() func(*http.Request, *retrier.RetrierOptions) {
	return httpClient.beforeRequest
}

func (httpClient *Client) GetAfterResponseCallback() func(*http.Response, *retrier.RetrierOptions, error) {
	return httpClient.afterResponse
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
		return r.Endpoints(serviceNames)
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

	interceptors := make([]Interceptor, 0, 3)

	var hostsRetryConfig, hostRetryConfig clientv2.RetryConfig
	if httpClient.hostsRetryConfig != nil {
		hostsRetryConfig = *httpClient.hostsRetryConfig
	}
	if hostsRetryConfig.RetryMax <= 0 {
		hostsRetryConfig.RetryMax = endpoints.HostsLength()
	}
	if hostsRetryConfig.Retrier == nil {
		hostsRetryConfig.Retrier = retrier.NewErrorRetrier()
	}

	if httpClient.hostRetryConfig != nil {
		hostRetryConfig = *httpClient.hostRetryConfig
	}
	if hostRetryConfig.RetryMax <= 0 {
		hostRetryConfig.RetryMax = 3
	}
	if hostRetryConfig.Retrier == nil {
		hostRetryConfig.Retrier = retrier.NewErrorRetrier()
	}

	interceptors = append(interceptors, clientv2.NewBufferResponseInterceptor())
	interceptors = append(interceptors, clientv2.NewHostsRetryInterceptor(clientv2.HostsRetryConfig{
		RetryMax:           hostsRetryConfig.RetryMax,
		ShouldRetry:        hostsRetryConfig.ShouldRetry,
		Retrier:            hostsRetryConfig.Retrier,
		HostFreezeDuration: httpClient.hostFreezeDuration,
		ShouldFreezeHost:   httpClient.shouldFreezeHost,
		HostProvider:       hostProvider,
	}))
	interceptors = append(interceptors, clientv2.NewSimpleRetryInterceptor(
		clientv2.SimpleRetryConfig{
			RetryMax:      hostRetryConfig.RetryMax,
			RetryInterval: hostRetryConfig.RetryInterval,
			Backoff:       hostRetryConfig.Backoff,
			ShouldRetry:   hostRetryConfig.ShouldRetry,
			Retrier:       hostRetryConfig.Retrier,
			Resolver:      httpClient.resolver,
			Chooser:       httpClient.chooser,
			BeforeResolve: httpClient.beforeResolve,
			AfterResolve:  httpClient.afterResolve,
			ResolveError:  httpClient.resolveError,
			BeforeBackoff: httpClient.beforeBackoff,
			AfterBackoff:  httpClient.afterBackoff,
			BeforeRequest: httpClient.beforeRequest,
			AfterResponse: httpClient.afterResponse,
		},
	))
	req, err := clientv2.NewRequest(clientv2.RequestParams{
		Context:           ctx,
		Method:            request.Method,
		Url:               url,
		Header:            request.Header,
		GetBody:           request.RequestBody,
		BufferResponse:    request.BufferResponse,
		OnRequestProgress: request.OnRequestProgress,
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
	return func(params *clientv2.RequestParams) (io.ReadCloser, error) {
		params.Header.Set("Content-Type", "application/octet-stream")
		totalSize, err := r.Seek(0, io.SeekEnd)
		if err != nil {
			return r, err
		}
		params.Header.Set("Content-Length", strconv.FormatInt(totalSize, 10))
		_, err = r.Seek(0, io.SeekStart)
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
