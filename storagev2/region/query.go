package region

import (
	"context"
	"crypto/md5"
	"fmt"
	"hash/crc64"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/internal/cache"
	"github.com/qiniu/go-sdk/v7/internal/clientv2"
	"github.com/qiniu/go-sdk/v7/internal/hostprovider"
	"github.com/qiniu/go-sdk/v7/internal/log"
	"github.com/qiniu/go-sdk/v7/storagev2/backoff"
	"github.com/qiniu/go-sdk/v7/storagev2/chooser"
	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/resolver"
	"github.com/qiniu/go-sdk/v7/storagev2/retrier"
)

type (
	// BucketRegionsQuery 空间区域查询器
	BucketRegionsQuery interface {
		Query(accessKey, bucketName string) RegionsProvider
	}

	bucketRegionsQuery struct {
		bucketHosts         Endpoints
		cache               *cache.Cache
		client              clientv2.Client
		useHttps            bool
		accelerateUploading bool
	}

	// BucketRegionsQueryOptions 空间区域查询器选项
	BucketRegionsQueryOptions struct {
		// 使用 HTTP 协议
		UseInsecureProtocol bool

		// 是否加速上传
		AccelerateUploading bool

		// 压缩周期（默认：60s）
		CompactInterval time.Duration

		// 持久化路径（默认：$TMPDIR/qiniu-golang-sdk/query_v4_01.cache.json）
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

	bucketRegionsProvider struct {
		accessKey           string
		bucketName          string
		cacheKey            string
		query               *bucketRegionsQuery
		accelerateUploading bool
	}

	v4QueryCacheValue struct {
		Regions      []*Region `json:"regions"`
		RefreshAfter time.Time `json:"refresh_after"`
		ExpiredAt    time.Time `json:"expired_at"`
	}

	v4QueryServiceHosts struct {
		Domains    []string `json:"domains"`
		Old        []string `json:"old"`
		AccDomains []string `json:"acc_domains"`
	}

	v4QueryRegion struct {
		RegionId string              `json:"region"`
		Ttl      int64               `json:"ttl"`
		Io       v4QueryServiceHosts `json:"io"`
		IoSrc    v4QueryServiceHosts `json:"io_src"`
		Up       v4QueryServiceHosts `json:"up"`
		Rs       v4QueryServiceHosts `json:"rs"`
		Rsf      v4QueryServiceHosts `json:"rsf"`
		Api      v4QueryServiceHosts `json:"api"`
		Uc       v4QueryServiceHosts `json:"uc"`
	}

	v4QueryResponse struct {
		Hosts []v4QueryRegion `json:"hosts"`
	}
)

const bucketRegionsQueryCacheFileName = "query_v4_01.cache.json"

var (
	persistentCaches     map[uint64]*cache.Cache
	persistentCachesLock sync.Mutex
)

// NewBucketRegionsQuery 创建空间区域查询器
func NewBucketRegionsQuery(bucketHosts Endpoints, opts *BucketRegionsQueryOptions) (BucketRegionsQuery, error) {
	if opts == nil {
		opts = &BucketRegionsQueryOptions{}
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
		persistentFilePath = filepath.Join(os.TempDir(), "qiniu-golang-sdk", bucketRegionsQueryCacheFileName)
	}
	persistentDuration := opts.PersistentDuration
	if persistentDuration == time.Duration(0) {
		persistentDuration = time.Minute
	}

	persistentCache, err := getPersistentCache(persistentFilePath, compactInterval, persistentDuration)
	if err != nil {
		return nil, err
	}

	return &bucketRegionsQuery{
		bucketHosts: bucketHosts,
		cache:       persistentCache,
		client: makeBucketQueryClient(
			opts.Client, nil,
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
			nil, nil, nil,
		),
		useHttps:            !opts.UseInsecureProtocol,
		accelerateUploading: opts.AccelerateUploading,
	}, nil
}

func getPersistentCache(persistentFilePath string, compactInterval, persistentDuration time.Duration) (*cache.Cache, error) {
	var (
		persistentCache *cache.Cache
		ok              bool
		err             error
	)

	crc64Value := calcPersistentCacheCrc64(persistentFilePath, compactInterval, persistentDuration)
	persistentCachesLock.Lock()
	defer persistentCachesLock.Unlock()

	if persistentCaches == nil {
		persistentCaches = make(map[uint64]*cache.Cache)
	}
	if persistentCache, ok = persistentCaches[crc64Value]; !ok {
		persistentCache, err = cache.NewPersistentCache(
			reflect.TypeOf(&v4QueryCacheValue{}),
			persistentFilePath,
			compactInterval,
			persistentDuration,
			func(err error) {
				log.Warn(fmt.Sprintf("BucketRegionsQuery persist error: %s", err))
			})
		if err != nil {
			return nil, err
		}
		persistentCaches[crc64Value] = persistentCache
	}
	return persistentCache, nil
}

// Query 查询空间区域，返回 region.RegionsProvider
func (query *bucketRegionsQuery) Query(accessKey, bucketName string) RegionsProvider {
	return &bucketRegionsProvider{
		accessKey:           accessKey,
		bucketName:          bucketName,
		query:               query,
		cacheKey:            makeRegionCacheKey(accessKey, bucketName, query.accelerateUploading, query.bucketHosts),
		accelerateUploading: query.accelerateUploading,
	}
}

func (provider *bucketRegionsProvider) GetRegions(ctx context.Context) ([]*Region, error) {
	var err error
	cacheValue, status := provider.query.cache.Get(provider.cacheKey, func() (cache.CacheValue, error) {
		var ret v4QueryResponse
		url := fmt.Sprintf("%s/v4/query?ak=%s&bucket=%s", provider.query.bucketHosts.firstUrl(provider.query.useHttps), provider.accessKey, provider.bucketName)
		if err = clientv2.DoAndDecodeJsonResponse(provider.query.client, clientv2.RequestParams{
			Context:        ctx,
			Method:         clientv2.RequestMethodGet,
			Url:            url,
			BufferResponse: true,
		}, &ret); err != nil {
			return nil, err
		}
		return ret.toCacheValue(provider.accelerateUploading), nil
	})
	if status == cache.NoResultGot {
		return nil, err
	}
	return cacheValue.(*v4QueryCacheValue).Regions, nil
}

func (left *v4QueryCacheValue) IsEqual(rightValue cache.CacheValue) bool {
	if right, ok := rightValue.(*v4QueryCacheValue); ok {
		if len(left.Regions) != len(right.Regions) {
			return false
		}
		for idx := range left.Regions {
			if !left.Regions[idx].IsEqual(right.Regions[idx]) {
				return false
			}
		}
		return true
	}
	return false
}

func (left *v4QueryCacheValue) IsValid() bool {
	return time.Now().Before(left.ExpiredAt)
}

func (left *v4QueryCacheValue) ShouldRefresh() bool {
	return time.Now().After(left.RefreshAfter)
}

func (response *v4QueryResponse) toCacheValue(accelerateUploading bool) *v4QueryCacheValue {
	var (
		minTtl  = int64(math.MaxInt64)
		regions = make([]*Region, 0, len(response.Hosts))
	)
	for _, host := range response.Hosts {
		regions = append(regions, host.toCacheValue(accelerateUploading))
		if host.Ttl < minTtl {
			minTtl = host.Ttl
		}
	}
	now := time.Now()
	return &v4QueryCacheValue{
		Regions:      regions,
		RefreshAfter: now.Add(time.Duration(minTtl) * time.Second / 2),
		ExpiredAt:    now.Add(time.Duration(minTtl) * time.Second),
	}
}

func (response *v4QueryRegion) toCacheValue(accelerateUploading bool) *Region {
	region := Region{
		RegionID: response.RegionId,
		Up:       response.Up.toCacheValue(),
		Io:       response.Io.toCacheValue(),
		IoSrc:    response.IoSrc.toCacheValue(),
		Rs:       response.Rs.toCacheValue(),
		Rsf:      response.Rsf.toCacheValue(),
		Api:      response.Api.toCacheValue(),
		Bucket:   response.Uc.toCacheValue(),
	}
	if !accelerateUploading {
		region.Up.Accelerated = nil
	}

	return &region
}

func (response *v4QueryServiceHosts) toCacheValue() Endpoints {
	return Endpoints{
		Accelerated: response.AccDomains,
		Preferred:   response.Domains,
		Alternative: response.Old,
	}
}

func makeRegionCacheKey(accessKey, bucketName string, accelerateUploading bool, bucketHosts Endpoints) string {
	enableAcceleration := uint8(0)
	if accelerateUploading {
		enableAcceleration = 1
	}
	return fmt.Sprintf("%s:%s:%d:%s", accessKey, bucketName, enableAcceleration, makeBucketHostsCacheKey(bucketHosts))
}

func makeBucketHostsCacheKey(serviceHosts Endpoints) string {
	return fmt.Sprintf("%s:%s:%s", makeHostsCacheKey(serviceHosts.Preferred), makeHostsCacheKey(serviceHosts.Alternative), makeHostsCacheKey(serviceHosts.Accelerated))
}

func makeHostsCacheKey(hosts []string) string {
	sortedHosts := append(make([]string, 0, len(hosts)), hosts...)
	sort.StringSlice(sortedHosts).Sort()
	return fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(sortedHosts, ","))))
}

func makeBucketQueryClient(
	client clientv2.Client,
	credentials credentials.CredentialsProvider,
	bucketHosts Endpoints,
	useHttps bool,
	retryMax int,
	hostFreezeDuration time.Duration,
	r resolver.Resolver,
	cs chooser.Chooser,
	bf backoff.Backoff,
	beforeResolve func(*http.Request),
	afterResolve func(*http.Request, []net.IP),
	resolveError func(*http.Request, error),
	beforeBackoff func(*http.Request, *retrier.RetrierOptions, time.Duration),
	afterBackoff func(*http.Request, *retrier.RetrierOptions, time.Duration),
	beforeRequest func(*http.Request, *retrier.RetrierOptions),
	afterResponse func(*http.Response, *retrier.RetrierOptions, error),
	beforeSign, afterSign func(*http.Request),
	signError func(*http.Request, error),
) clientv2.Client {
	is := []clientv2.Interceptor{
		clientv2.NewAntiHijackingInterceptor(),
		clientv2.NewHostsRetryInterceptor(clientv2.HostsRetryConfig{
			RetryMax:           bucketHosts.HostsLength(),
			HostFreezeDuration: hostFreezeDuration,
			HostProvider:       hostprovider.NewWithHosts(bucketHosts.allUrls(useHttps)),
		}),
		clientv2.NewSimpleRetryInterceptor(clientv2.SimpleRetryConfig{
			RetryMax:      retryMax,
			Backoff:       bf,
			Resolver:      r,
			Chooser:       cs,
			BeforeResolve: beforeResolve,
			AfterResolve:  afterResolve,
			ResolveError:  resolveError,
			BeforeBackoff: beforeBackoff,
			AfterBackoff:  afterBackoff,
			BeforeRequest: beforeRequest,
			AfterResponse: afterResponse,
		}),
		clientv2.NewBufferResponseInterceptor(),
	}
	if credentials != nil {
		is = append(is, clientv2.NewAuthInterceptor(clientv2.AuthConfig{
			Credentials: credentials,
			TokenType:   auth.TokenQiniu,
			BeforeSign:  beforeSign,
			AfterSign:   afterSign,
			SignError:   signError,
		}))
	}
	return clientv2.NewClient(client, is...)
}

func calcPersistentCacheCrc64(persistentFilePath string, compactInterval, persistentDuration time.Duration) uint64 {
	bytes := make([]byte, 0, 1024)
	bytes = strconv.AppendInt(bytes, int64(compactInterval), 36)
	bytes = append(bytes, []byte(persistentFilePath)...)
	bytes = append(bytes, byte(0))
	bytes = strconv.AppendInt(bytes, int64(persistentDuration), 36)
	return crc64.Checksum(bytes, crc64.MakeTable(crc64.ISO))
}
