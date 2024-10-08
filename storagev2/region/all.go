package region

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/qiniu/go-sdk/v7/internal/cache"
	"github.com/qiniu/go-sdk/v7/internal/clientv2"
	"github.com/qiniu/go-sdk/v7/storagev2/backoff"
	"github.com/qiniu/go-sdk/v7/storagev2/chooser"
	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/resolver"
	"github.com/qiniu/go-sdk/v7/storagev2/retrier"
)

type (
	allRegionsProvider struct {
		credentials credentials.CredentialsProvider
		bucketHosts Endpoints
		cache       *cache.Cache
		client      clientv2.Client
		useHttps    bool
	}

	// AllRegionsProviderOptions 所有区域提供者选项
	AllRegionsProviderOptions struct {
		// 使用 HTTP 协议
		UseInsecureProtocol bool

		// 压缩周期（默认：60s）
		CompactInterval time.Duration

		// 持久化路径（默认：$TMPDIR/qiniu-golang-sdk/regions_v4_01.cache.json）
		PersistentFilePath string

		// 持久化周期（默认：60s）
		PersistentDuration time.Duration

		// 单域名重试次数（默认：2）
		RetryMax int

		// 主备域名冻结时间（默认：600s），当一个域名请求失败（单个域名会被重试 RetryMax 次），会被冻结一段时间，使用备用域名进行重试，在冻结时间内，域名不能被使用，当一个操作中所有域名竣备冻结操作不在进行重试，返回最后一次操作的错误。
		HostFreezeDuration time.Duration

		// HTTP 客户端，如果不配置则使用默认的 HTTP 客户端
		Client clientv2.Client

		// 域名解析器，如果不配置则使用默认的域名解析器
		Resolver resolver.Resolver

		// 域名选择器，如果不配置则使用默认的域名选择器
		Chooser chooser.Chooser

		// 退避器，如果不配置则使用默认的退避器
		Backoff backoff.Backoff

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
	}

	singleRegion struct {
		ID  string              `json:"id"`
		Ttl int64               `json:"ttl"`
		Io  v4QueryServiceHosts `json:"io"`
		Up  v4QueryServiceHosts `json:"up"`
		Rs  v4QueryServiceHosts `json:"rs"`
		Rsf v4QueryServiceHosts `json:"rsf"`
		Api v4QueryServiceHosts `json:"api"`
		Uc  v4QueryServiceHosts `json:"uc"`
	}

	regionsResponse struct {
		Regions []singleRegion `json:"regions"`
	}
)

const allRegionsProviderCacheFileName = "regions_v4_01.cache.json"

// NewAllRegionsProvider 创建所有空间提供者
func NewAllRegionsProvider(credentials credentials.CredentialsProvider, bucketHosts Endpoints, opts *AllRegionsProviderOptions) (RegionsProvider, error) {
	if opts == nil {
		opts = &AllRegionsProviderOptions{}
	}
	retryMax := opts.RetryMax
	if retryMax <= 0 {
		retryMax = 2
	}
	compactInterval := opts.CompactInterval
	if compactInterval == time.Duration(0) {
		compactInterval = time.Minute
	}
	persistentFilePath := opts.PersistentFilePath
	if persistentFilePath == "" {
		persistentFilePath = filepath.Join(os.TempDir(), "qiniu-golang-sdk", allRegionsProviderCacheFileName)
	}
	persistentDuration := opts.PersistentDuration
	if persistentDuration == time.Duration(0) {
		persistentDuration = time.Minute
	}

	persistentCache, err := getPersistentCache(persistentFilePath, compactInterval, persistentDuration)
	if err != nil {
		return nil, err
	}

	return &allRegionsProvider{
		credentials: credentials,
		bucketHosts: bucketHosts,
		cache:       persistentCache,
		client: makeBucketQueryClient(
			opts.Client,
			credentials,
			bucketHosts,
			!opts.UseInsecureProtocol,
			retryMax,
			opts.HostFreezeDuration,
			opts.Resolver,
			opts.Chooser,
			opts.Backoff,
			opts.BeforeResolve,
			opts.AfterResolve,
			opts.ResolveError,
			opts.BeforeBackoff,
			opts.AfterBackoff,
			opts.BeforeRequest,
			opts.AfterResponse,
			opts.BeforeSign,
			opts.AfterSign,
			opts.SignError,
		),
		useHttps: !opts.UseInsecureProtocol,
	}, nil
}

func (provider *allRegionsProvider) GetRegions(ctx context.Context) ([]*Region, error) {
	creds, err := provider.credentials.Get(ctx)
	if err != nil {
		return nil, err
	}
	cacheValue, status := provider.cache.Get(makeRegionsProviderCacheKey(creds.AccessKey, provider.bucketHosts), func() (cache.CacheValue, error) {
		var ret regionsResponse
		url := provider.bucketHosts.firstUrl(provider.useHttps) + "/regions"
		if err = clientv2.DoAndDecodeJsonResponse(provider.client, clientv2.RequestParams{
			Context:        ctx,
			Method:         clientv2.RequestMethodGet,
			Url:            url,
			BufferResponse: true,
		}, &ret); err != nil {
			return nil, err
		}
		return ret.toCacheValue(), nil
	})
	if status == cache.NoResultGot {
		return nil, err
	}
	return cacheValue.(*v4QueryCacheValue).Regions, nil
}

func (response *regionsResponse) toCacheValue() *v4QueryCacheValue {
	var (
		minTtl  = int64(math.MaxInt64)
		regions = make([]*Region, 0, len(response.Regions))
	)
	for _, host := range response.Regions {
		regions = append(regions, host.toCacheValue())
		if host.Ttl < minTtl {
			minTtl = host.Ttl
		}
	}
	now := time.Now()
	return &v4QueryCacheValue{
		Regions:      regions,
		RefreshAfter: now.Add(time.Duration(minTtl/2) * time.Second),
		ExpiredAt:    now.Add(time.Duration(minTtl) * time.Second),
	}
}

func (response *singleRegion) toCacheValue() *Region {
	return &Region{
		RegionID: response.ID,
		Up:       response.Up.toCacheValue(),
		Io:       response.Io.toCacheValue(),
		Rs:       response.Rs.toCacheValue(),
		Rsf:      response.Rsf.toCacheValue(),
		Api:      response.Api.toCacheValue(),
		Bucket:   response.Uc.toCacheValue(),
	}
}

func makeRegionsProviderCacheKey(accessKey string, bucketHosts Endpoints) string {
	return fmt.Sprintf("%s:%s", accessKey, makeBucketHostsCacheKey(bucketHosts))
}
