package region

import (
	"context"
	"crypto/md5"
	"fmt"
	"hash/crc64"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/qiniu/go-sdk/v7/internal/cache"
	"github.com/qiniu/go-sdk/v7/internal/clientv2"
	"github.com/qiniu/go-sdk/v7/internal/hostprovider"
	"github.com/qiniu/go-sdk/v7/internal/log"
)

type (
	// BucketRegionsQueryer 空间区域查询器
	BucketRegionsQueryer interface {
		Query(accessKey, bucketName string) RegionsProvider
	}

	bucketRegionsQueryer struct {
		bucketHosts Endpoints
		cache       *cache.Cache
		client      clientv2.Client
		useHttps    bool
	}

	// BucketRegionsQueryer 空间区域查询器选项
	BucketRegionsQueryerOptions struct {
		// 使用 HTTP 协议
		UseInsecureProtocol bool

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
	}

	bucketRegionsProvider struct {
		accessKey  string
		bucketName string
		cacheKey   string
		queryer    *bucketRegionsQueryer
	}

	v4QueryCacheValue struct {
		Regions   []*Region `json:"regions"`
		ExpiredAt time.Time `json:"expired_at"`
	}

	v4QueryServiceHosts struct {
		Domains []string `json:"domains"`
		Old     []string `json:"old"`
	}

	v4QueryRegion struct {
		RegionId RegionID            `json:"region"`
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

const cacheFileName = "query_v4_01.cache.json"

var (
	queryerCaches     map[uint64]BucketRegionsQueryer
	queryerCachesLock sync.Mutex
)

// NewBucketRegionsQueryer 创建空间区域查询器
func NewBucketRegionsQueryer(bucketHosts Endpoints, opts *BucketRegionsQueryerOptions) (BucketRegionsQueryer, error) {
	if opts == nil {
		opts = &BucketRegionsQueryerOptions{}
	}
	if opts.RetryMax <= 0 {
		opts.RetryMax = 2
	}
	if opts.CompactInterval == time.Duration(0) {
		opts.CompactInterval = time.Minute
	}
	if opts.PersistentFilePath == "" {
		opts.PersistentFilePath = filepath.Join(os.TempDir(), "qiniu-golang-sdk", cacheFileName)
	}
	if opts.PersistentDuration == time.Duration(0) {
		opts.PersistentDuration = time.Minute
	}

	crc64Value := calcBucketRegionsQueryerCrc64(bucketHosts, opts)
	queryerCachesLock.Lock()
	defer queryerCachesLock.Unlock()

	if queryerCaches == nil {
		queryerCaches = make(map[uint64]BucketRegionsQueryer)
	}

	if queryer, ok := queryerCaches[crc64Value]; ok {
		return queryer, nil
	} else {
		persistentCache, err := cache.NewPersistentCache(reflect.TypeOf(&v4QueryCacheValue{}), opts.PersistentFilePath, opts.CompactInterval, opts.PersistentDuration, func(err error) {
			log.Warn(fmt.Sprintf("BucketRegionsQueryer persist error: %s", err))
		})
		if err != nil {
			return nil, err
		}
		queryer = &bucketRegionsQueryer{
			bucketHosts: bucketHosts,
			cache:       persistentCache,
			client:      makeBucketQueryClient(opts.Client, bucketHosts, !opts.UseInsecureProtocol, opts.RetryMax, opts.HostFreezeDuration),
			useHttps:    !opts.UseInsecureProtocol,
		}
		queryerCaches[crc64Value] = queryer
		return queryer, nil
	}
}

// Query 查询空间区域，返回 region.RegionsProvider
func (queryer *bucketRegionsQueryer) Query(accessKey, bucketName string) RegionsProvider {
	return &bucketRegionsProvider{
		accessKey:  accessKey,
		bucketName: bucketName,
		queryer:    queryer,
		cacheKey:   makeRegionCacheKey(accessKey, bucketName, queryer.bucketHosts),
	}
}

func (provider *bucketRegionsProvider) GetRegions(ctx context.Context) ([]*Region, error) {
	var err error
	cacheValue, status := provider.queryer.cache.Get(provider.cacheKey, func() (cache.CacheValue, error) {
		var ret v4QueryResponse
		url := fmt.Sprintf("%s/v4/query?ak=%s&bucket=%s", provider.queryer.bucketHosts.firstUrl(provider.queryer.useHttps), provider.accessKey, provider.bucketName)
		if err = clientv2.DoAndDecodeJsonResponse(provider.queryer.client, clientv2.RequestParams{
			Context: ctx,
			Method:  clientv2.RequestMethodGet,
			Url:     url,
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

func (response *v4QueryResponse) toCacheValue() *v4QueryCacheValue {
	var (
		minTtl  = int64(math.MaxInt64)
		regions = make([]*Region, 0, len(response.Hosts))
	)
	for _, host := range response.Hosts {
		regions = append(regions, host.toCacheValue())
		if host.Ttl < minTtl {
			minTtl = host.Ttl
		}
	}
	return &v4QueryCacheValue{
		Regions:   regions,
		ExpiredAt: time.Now().Add(time.Duration(minTtl) * time.Second),
	}
}

func (response *v4QueryRegion) toCacheValue() *Region {
	return &Region{
		RegionID: response.RegionId,
		Up:       response.Up.toCacheValue(),
		Io:       response.Io.toCacheValue(),
		IoSrc:    response.IoSrc.toCacheValue(),
		Rs:       response.Rs.toCacheValue(),
		Rsf:      response.Rsf.toCacheValue(),
		Api:      response.Api.toCacheValue(),
		Bucket:   response.Uc.toCacheValue(),
	}
}

func (response *v4QueryServiceHosts) toCacheValue() Endpoints {
	return Endpoints{
		Preferred:   response.Domains,
		Alternative: response.Old,
	}
}

func makeRegionCacheKey(accessKey, bucketName string, bucketHosts Endpoints) string {
	return fmt.Sprintf("%s:%s:%s", accessKey, bucketName, makeBucketHostsCacheKey(bucketHosts))
}

func makeBucketHostsCacheKey(serviceHosts Endpoints) string {
	return fmt.Sprintf("%s:%s", makeHostsCacheKey(serviceHosts.Preferred), makeHostsCacheKey(serviceHosts.Alternative))
}

func makeHostsCacheKey(hosts []string) string {
	sortedHosts := append(make([]string, 0, len(hosts)), hosts...)
	sort.StringSlice(sortedHosts).Sort()
	return fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(sortedHosts, ","))))
}

func makeBucketQueryClient(client clientv2.Client, bucketHosts Endpoints, useHttps bool, retryMax int, hostFreezeDuration time.Duration) clientv2.Client {
	is := []clientv2.Interceptor{
		clientv2.NewHostsRetryInterceptor(clientv2.HostsRetryConfig{
			RetryConfig: clientv2.RetryConfig{
				RetryMax:      len(bucketHosts.Preferred) + len(bucketHosts.Alternative),
				RetryInterval: nil,
				ShouldRetry:   nil,
			},
			ShouldFreezeHost:   nil,
			HostFreezeDuration: hostFreezeDuration,
			HostProvider:       hostprovider.NewWithHosts(bucketHosts.allUrls(useHttps)),
		}),
		clientv2.NewSimpleRetryInterceptor(clientv2.RetryConfig{
			RetryMax:      retryMax,
			RetryInterval: nil,
			ShouldRetry:   nil,
		}),
	}
	return clientv2.NewClient(client, is...)
}

func (opts *BucketRegionsQueryerOptions) toBytes() []byte {
	bytes := make([]byte, 0, 1024)
	bytes = strconv.AppendBool(bytes, opts.UseInsecureProtocol)
	bytes = strconv.AppendInt(bytes, int64(opts.CompactInterval), 36)
	bytes = append(bytes, []byte(opts.PersistentFilePath)...)
	bytes = append(bytes, byte(0))
	bytes = strconv.AppendInt(bytes, int64(opts.PersistentDuration), 36)
	bytes = strconv.AppendInt(bytes, int64(opts.RetryMax), 36)
	bytes = strconv.AppendInt(bytes, int64(opts.HostFreezeDuration), 36)
	if opts.Client != nil {
		bytes = strconv.AppendUint(bytes, uint64(uintptr(unsafe.Pointer(&opts.Client))), 10)
	} else {
		bytes = strconv.AppendUint(bytes, 0, 10)
	}
	return bytes
}

func calcBucketRegionsQueryerCrc64(bucketHosts Endpoints, opts *BucketRegionsQueryerOptions) uint64 {
	hasher := crc64.New(crc64.MakeTable(crc64.ISO))
	hasher.Write(bucketHosts.toBytes())
	hasher.Write(opts.toBytes())
	return hasher.Sum64()
}
