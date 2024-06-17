package downloader

import (
	"context"
	"fmt"
	"hash/crc64"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/qiniu/go-sdk/v7/internal/cache"
	"github.com/qiniu/go-sdk/v7/internal/log"
	"github.com/qiniu/go-sdk/v7/storagev2/apis"
	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/errors"
	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/region"
)

type (
	unsignedURL struct {
		url *url.URL
	}

	signedURL struct {
		ctx     context.Context
		url     URLProvider
		signer  Signer
		options *SignOptions
		cache   *cache.Cache
	}

	staticDomainBasedURLsProvider struct {
		domains []string
	}

	defaultSrcURLsProvider struct {
		credentials credentials.CredentialsProvider
		query       region.BucketRegionsQuery
		ttl         time.Duration
	}

	domainsQueryURLsProvider struct {
		storage     *apis.Storage
		cache       *cache.Cache
		credentials credentials.CredentialsProvider
		cacheTTL    time.Duration
	}

	combinedDownloadURLsProviders struct {
		providers []DownloadURLsProvider
	}

	signedDownloadURLsProviders struct {
		provider DownloadURLsProvider
		signer   Signer
		options  *SignOptions
	}

	// 默认源站域名下载 URL 生成器选项
	DefaultSrcURLsProviderOptions struct {
		region.BucketRegionsQueryOptions
		SignOptions

		// Bucket 服务器地址
		BucketHosts region.Endpoints
	}

	// 基于域名查询的下载 URL 生成器选项
	DomainsQueryURLsProviderOptions struct {
		http_client.Options

		// 压缩周期（默认：60s）
		CompactInterval time.Duration

		// 持久化路径（默认：$TMPDIR/qiniu-golang-sdk/domain_v2_01.cache.json）
		PersistentFilePath string

		// 持久化周期（默认：60s）
		PersistentDuration time.Duration

		// 缓存有效周期（默认：3600s）
		CacheTTL time.Duration
	}

	domainCacheValue struct {
		Domains   []string  `json:"domains"`
		ExpiredAt time.Time `json:"expired_at"`
	}

	signingCacheValue struct {
		url       *url.URL
		expiredAt time.Time
	}
)

// 转换 URL
func NewURLProvider(u *url.URL) URLProvider {
	return &unsignedURL{u}
}

func (uu *unsignedURL) GetURL(u *url.URL) error {
	*u = *uu.url
	return nil
}

// 为 URL 列表签名
func SignURLs(ctx context.Context, url URLProvider, signer Signer, options *SignOptions) URLProvider {
	return &signedURL{ctx: ctx, url: url, signer: signer, options: options, cache: cache.NewCache(1 * time.Second)}
}

func (su *signedURL) GetURL(u *url.URL) error {
	key := u.String()
	var err error
	cacheValue, status := su.cache.Get(key, func() (cache.CacheValue, error) {
		var (
			signedURL url.URL
		)
		if err = su.url.GetURL(&signedURL); err != nil {
			return nil, err
		}
		if err = su.signer.Sign(su.ctx, &signedURL, su.options); err != nil {
			return nil, err
		}
		return signingCacheValue{&signedURL, time.Now().Add(1 * time.Second)}, nil
	})
	if status == cache.NoResultGot {
		return err
	}
	*u = *cacheValue.(signingCacheValue).url
	return nil
}

func (scv signingCacheValue) IsEqual(cv cache.CacheValue) bool {
	return scv.url.String() == cv.(signingCacheValue).url.String()
}

func (scv signingCacheValue) IsValid() bool {
	return scv.expiredAt.After(time.Now())
}

// 创建静态域名下载 URL 生成器
func NewStaticDomainBasedURLsProvider(domains []string) DownloadURLsProvider {
	return &staticDomainBasedURLsProvider{domains}
}

func (g *staticDomainBasedURLsProvider) GetURLs(_ context.Context, objectName string, options *GenerateOptions) ([]URLProvider, error) {
	if options == nil {
		options = &GenerateOptions{}
	}
	urls := make([]URLProvider, 0, len(g.domains))
	for _, domain := range g.domains {
		if !strings.Contains(domain, "://") {
			if options.UseInsecureProtocol {
				domain = "http://" + domain
			} else {
				domain = "https://" + domain
			}
		}
		u, err := url.Parse(domain)
		if err != nil {
			return nil, err
		}
		u.Path = "/" + objectName
		u.RawPath = ""
		u.RawQuery = options.Command
		urls = append(urls, NewURLProvider(u))
	}
	return urls, nil
}

// 创建默认源站域名下载 URL 生成器
func NewDefaultSrcURLsProvider(credentials credentials.CredentialsProvider, options *DefaultSrcURLsProviderOptions) (DownloadURLsProvider, error) {
	if options == nil {
		options = &DefaultSrcURLsProviderOptions{}
	}
	bucketHosts := options.BucketHosts
	if bucketHosts.IsEmpty() {
		bucketHosts = http_client.DefaultBucketHosts()
	}
	ttl := options.Ttl
	if ttl == 0 {
		ttl = 3 * time.Minute
	}
	query, err := region.NewBucketRegionsQuery(bucketHosts, &options.BucketRegionsQueryOptions)
	if err != nil {
		return nil, err
	}
	return &defaultSrcURLsProvider{credentials, query, ttl}, nil
}

func (g *defaultSrcURLsProvider) GetURLs(ctx context.Context, objectName string, options *GenerateOptions) ([]URLProvider, error) {
	if options == nil {
		options = &GenerateOptions{}
	}
	if options.BucketName == "" {
		return nil, errors.MissingRequiredFieldError{Name: "BucketName"}
	}
	cred, err := g.credentials.Get(ctx)
	if err != nil {
		return nil, err
	}
	regions, err := g.query.Query(cred.AccessKey, options.BucketName).GetRegions(ctx)
	if err != nil {
		return nil, err
	}
	if len(regions) == 0 {
		return nil, http_client.ErrNoRegion
	}
	region := regions[0]
	ioSrcDomains := make([]string, 0, len(region.IoSrc.Preferred)+len(region.IoSrc.Alternative))
	ioSrcDomains = append(ioSrcDomains, region.IoSrc.Preferred...)
	ioSrcDomains = append(ioSrcDomains, region.IoSrc.Alternative...)
	return SignURLsProvider(NewStaticDomainBasedURLsProvider(ioSrcDomains), NewCredentialsSigner(g.credentials), &SignOptions{g.ttl}).GetURLs(ctx, objectName, options)
}

const cacheFileName = "domain_v2_01.cache.json"

var (
	persistentCaches     map[uint64]*cache.Cache
	persistentCachesLock sync.Mutex
)

// 创建基于域名查询的下载 URL 生成器
func NewDomainsQueryURLsProvider(options *DomainsQueryURLsProviderOptions) (DownloadURLsProvider, error) {
	if options == nil {
		options = &DomainsQueryURLsProviderOptions{}
	}
	if options.Credentials == nil {
		return nil, errors.MissingRequiredFieldError{Name: "Credentials"}
	}
	compactInterval := options.CompactInterval
	if compactInterval == time.Duration(0) {
		compactInterval = time.Minute
	}
	persistentFilePath := options.PersistentFilePath
	if persistentFilePath == "" {
		persistentFilePath = filepath.Join(os.TempDir(), "qiniu-golang-sdk", cacheFileName)
	}
	persistentDuration := options.PersistentDuration
	if persistentDuration == time.Duration(0) {
		persistentDuration = time.Minute
	}
	cacheTTL := options.CacheTTL
	if cacheTTL == time.Duration(0) {
		cacheTTL = time.Hour
	}
	persistentCache, err := getPersistentCache(persistentFilePath, compactInterval, persistentDuration)
	if err != nil {
		return nil, err
	}

	storage := apis.NewStorage(&options.Options)
	return &domainsQueryURLsProvider{storage, persistentCache, options.Credentials, cacheTTL}, nil
}

func (g *domainsQueryURLsProvider) GetURLs(ctx context.Context, objectName string, options *GenerateOptions) ([]URLProvider, error) {
	var (
		creds *credentials.Credentials
		err   error
	)

	if options == nil {
		options = &GenerateOptions{}
	}
	if options.BucketName == "" {
		return nil, errors.MissingRequiredFieldError{Name: "BucketName"}
	}
	if creds, err = g.credentials.Get(ctx); err != nil {
		return nil, err
	}
	cacheKey := fmt.Sprintf("%s:%s", creds.AccessKey, options.BucketName)

	cacheValue, status := g.cache.Get(cacheKey, func() (cache.CacheValue, error) {
		response, err := g.storage.GetBucketDomains(ctx, &apis.GetBucketDomainsRequest{BucketName: options.BucketName}, nil)
		if err != nil {
			return nil, err
		} else {
			return &domainCacheValue{Domains: response.Domains, ExpiredAt: time.Now().Add(g.cacheTTL)}, nil
		}
	})
	if status == cache.NoResultGot {
		return nil, err
	}
	domains := cacheValue.(*domainCacheValue).Domains
	return NewStaticDomainBasedURLsProvider(domains).GetURLs(ctx, objectName, options)
}

func (left *domainCacheValue) IsEqual(rightValue cache.CacheValue) bool {
	if right, ok := rightValue.(*domainCacheValue); ok {
		if len(left.Domains) != len(right.Domains) {
			return false
		}
		for idx := range left.Domains {
			if left.Domains[idx] != right.Domains[idx] {
				return false
			}
		}
		return true
	}
	return false
}

func (left *domainCacheValue) IsValid() bool {
	return time.Now().Before(left.ExpiredAt)
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
			reflect.TypeOf(&domainCacheValue{}),
			persistentFilePath,
			compactInterval,
			persistentDuration,
			func(err error) {
				log.Warn(fmt.Sprintf("DomainsURLsProvider persist error: %s", err))
			})
		if err != nil {
			return nil, err
		}
		persistentCaches[crc64Value] = persistentCache
	}
	return persistentCache, nil
}

func calcPersistentCacheCrc64(persistentFilePath string, compactInterval, persistentDuration time.Duration) uint64 {
	bytes := make([]byte, 0, 1024)
	bytes = strconv.AppendInt(bytes, int64(compactInterval), 36)
	bytes = append(bytes, []byte(persistentFilePath)...)
	bytes = append(bytes, byte(0))
	bytes = strconv.AppendInt(bytes, int64(persistentDuration), 36)
	return crc64.Checksum(bytes, crc64.MakeTable(crc64.ISO))
}

// 合并多个下载 URL 生成器
func CombineDownloadURLsProviders(providers ...DownloadURLsProvider) DownloadURLsProvider {
	return combinedDownloadURLsProviders{providers}
}

func (g combinedDownloadURLsProviders) GetURLs(ctx context.Context, objectName string, options *GenerateOptions) ([]URLProvider, error) {
	urlProviders := make([]URLProvider, 0, len(g.providers))
	for _, downloadURLsProvider := range g.providers {
		ups, err := downloadURLsProvider.GetURLs(ctx, objectName, options)
		if err != nil {
			return nil, err
		}
		urlProviders = append(urlProviders, ups...)
	}
	return urlProviders, nil
}

// 为下载 URL 获取结果签名
func SignURLsProvider(provider DownloadURLsProvider, signer Signer, options *SignOptions) DownloadURLsProvider {
	return signedDownloadURLsProviders{provider, signer, options}
}

func (provider signedDownloadURLsProviders) GetURLs(ctx context.Context, objectName string, options *GenerateOptions) ([]URLProvider, error) {
	urls, err := provider.provider.GetURLs(ctx, objectName, options)
	if err != nil {
		return nil, err
	}
	for i := range urls {
		urls[i] = SignURLs(ctx, urls[i], provider.signer, provider.options)
	}
	return urls, nil
}
