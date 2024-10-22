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
	simpleURLsIter struct {
		urls, used []*url.URL
	}

	signedURLsIter struct {
		ctx      context.Context
		urlsIter URLsIter
		signer   Signer
		options  *SignOptions
		cache    *cache.Cache
	}

	staticDomainBasedURLsProvider struct {
		domains []string
	}

	defaultSrcURLsProvider struct {
		accessKey   string
		query       region.BucketRegionsQuery
		queryOnce   sync.Once
		bucketHosts region.Endpoints
		options     *DefaultSrcURLsProviderOptions
	}

	domainsQueryURLsProvider struct {
		storage           *apis.Storage
		cache             *cache.Cache
		credentials       credentials.CredentialsProvider
		cacheTTL          time.Duration
		cacheRefreshAfter time.Duration
	}

	combinedDownloadURLsProviders struct {
		providers []DownloadURLsProvider
	}

	combinedURLsIter struct {
		iters, used []URLsIter
	}

	signedDownloadURLsProviders struct {
		provider DownloadURLsProvider
		signer   Signer
		options  *SignOptions
	}

	// 默认源站域名下载 URL 生成器选项
	DefaultSrcURLsProviderOptions struct {
		region.BucketRegionsQueryOptions

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

		// 缓存刷新时间（默认：1800s）
		CacheRefreshAfter time.Duration
	}

	domainCacheValue struct {
		Domains      []string  `json:"domains"`
		RefreshAfter time.Time `json:"refresh_after"`
		ExpiredAt    time.Time `json:"expired_at"`
	}

	signingCacheValue struct {
		url       *url.URL
		expiredAt time.Time
	}
)

// 将 URL 列表转换为迭代器
func NewURLsIter(urls []*url.URL) URLsIter {
	return &simpleURLsIter{urls: urls, used: make([]*url.URL, 0, len(urls))}
}

func (s *simpleURLsIter) Peek(u *url.URL) (bool, error) {
	if len(s.urls) > 0 {
		*u = *s.urls[0]
		return true, nil
	}
	return false, nil
}

func (s *simpleURLsIter) Next() {
	if len(s.urls) > 0 {
		s.used = append(s.used, s.urls[0])
		s.urls = s.urls[1:]
	}
}

func (s *simpleURLsIter) Reset() {
	s.urls = append(s.used, s.urls...)
	s.used = make([]*url.URL, 0, cap(s.urls))
}

func (s *simpleURLsIter) Clone() URLsIter {
	return &simpleURLsIter{
		urls: append(make([]*url.URL, 0, cap(s.urls)), s.urls...),
		used: append(make([]*url.URL, 0, cap(s.used)), s.used...),
	}
}

// 为 URL 列表签名
func SignURLs(ctx context.Context, urlsIter URLsIter, signer Signer, options *SignOptions) URLsIter {
	return &signedURLsIter{ctx: ctx, urlsIter: urlsIter, signer: signer, options: options, cache: cache.NewCache(1 * time.Second)}
}

func (s *signedURLsIter) Peek(u *url.URL) (bool, error) {
	var unsignedURL url.URL
	if ok, err := s.urlsIter.Peek(&unsignedURL); err != nil {
		return ok, err
	} else if ok {
		var err error
		cacheValue, status := s.cache.Get(unsignedURL.String(), func() (cache.CacheValue, error) {
			signedURL := unsignedURL
			if err = s.signer.Sign(s.ctx, &signedURL, s.options); err != nil {
				return nil, err
			}
			return signingCacheValue{&signedURL, time.Now().Add(1 * time.Second)}, nil
		})
		if status == cache.NoResultGot {
			return false, err
		}
		*u = *cacheValue.(signingCacheValue).url
		return true, nil
	}
	return false, nil
}

func (s *signedURLsIter) Next() {
	s.urlsIter.Next()
}

func (s *signedURLsIter) Reset() {
	s.urlsIter.Reset()
}

func (s *signedURLsIter) Clone() URLsIter {
	return &signedURLsIter{
		ctx:      s.ctx,
		urlsIter: s.urlsIter.Clone(),
		signer:   s.signer,
		options:  s.options,
		cache:    s.cache,
	}
}

func (scv signingCacheValue) IsEqual(cv cache.CacheValue) bool {
	return scv.url.String() == cv.(signingCacheValue).url.String()
}

func (scv signingCacheValue) IsValid() bool {
	return time.Now().Before(scv.expiredAt)
}

func (scv signingCacheValue) ShouldRefresh() bool {
	return false
}

// 创建静态域名下载 URL 生成器
func NewStaticDomainBasedURLsProvider(domains []string) DownloadURLsProvider {
	return &staticDomainBasedURLsProvider{domains}
}

func (g *staticDomainBasedURLsProvider) GetURLsIter(_ context.Context, objectName string, options *GenerateOptions) (URLsIter, error) {
	if options == nil {
		options = &GenerateOptions{}
	}
	urls := make([]*url.URL, 0, len(g.domains))
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
		urls = append(urls, u)
	}
	return NewURLsIter(urls), nil
}

// 创建默认源站域名下载 URL 生成器
func NewDefaultSrcURLsProvider(accessKey string, options *DefaultSrcURLsProviderOptions) DownloadURLsProvider {
	if options == nil {
		options = &DefaultSrcURLsProviderOptions{}
	}
	bucketHosts := options.BucketHosts
	if bucketHosts.IsEmpty() {
		bucketHosts = http_client.DefaultBucketHosts()
	}
	return &defaultSrcURLsProvider{accessKey: accessKey, bucketHosts: bucketHosts, options: options}
}

func (g *defaultSrcURLsProvider) GetURLsIter(ctx context.Context, objectName string, options *GenerateOptions) (URLsIter, error) {
	if options == nil {
		options = &GenerateOptions{}
	}
	if options.BucketName == "" {
		return nil, errors.MissingRequiredFieldError{Name: "BucketName"}
	}

	var err error
	g.queryOnce.Do(func() {
		g.query, err = region.NewBucketRegionsQuery(g.bucketHosts, &g.options.BucketRegionsQueryOptions)
	})
	if err != nil {
		return nil, err
	}

	accessKey := g.accessKey
	if accessKey == "" {
		if defaultCreds := credentials.Default(); defaultCreds != nil {
			accessKey = defaultCreds.AccessKey
		}
	}

	regions, err := g.query.Query(accessKey, options.BucketName).GetRegions(ctx)
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
	return NewStaticDomainBasedURLsProvider(ioSrcDomains).GetURLsIter(ctx, objectName, options)
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
	creds := options.Credentials
	if creds == nil {
		if defaultCreds := credentials.Default(); defaultCreds != nil {
			creds = defaultCreds
		}
	}
	if creds == nil {
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
	cacheRefreshAfter := options.CacheRefreshAfter
	if cacheRefreshAfter == time.Duration(0) {
		cacheRefreshAfter = time.Hour / 2
	}
	persistentCache, err := getPersistentCache(persistentFilePath, compactInterval, persistentDuration)
	if err != nil {
		return nil, err
	}

	storage := apis.NewStorage(&options.Options)
	return &domainsQueryURLsProvider{storage, persistentCache, creds, cacheTTL, cacheRefreshAfter}, nil
}

func (g *domainsQueryURLsProvider) GetURLsIter(ctx context.Context, objectName string, options *GenerateOptions) (URLsIter, error) {
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
			now := time.Now()
			return &domainCacheValue{
				Domains:      response.Domains,
				RefreshAfter: now.Add(g.cacheRefreshAfter),
				ExpiredAt:    now.Add(g.cacheTTL),
			}, nil
		}
	})
	if status == cache.NoResultGot {
		return nil, err
	}
	domains := cacheValue.(*domainCacheValue).Domains
	return NewStaticDomainBasedURLsProvider(domains).GetURLsIter(ctx, objectName, options)
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

func (left *domainCacheValue) ShouldRefresh() bool {
	return time.Now().After(left.RefreshAfter)
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

func (g combinedDownloadURLsProviders) GetURLsIter(ctx context.Context, objectName string, options *GenerateOptions) (URLsIter, error) {
	urlIters := make([]URLsIter, 0, len(g.providers))
	for _, downloadURLsProvider := range g.providers {
		urlsIter, err := downloadURLsProvider.GetURLsIter(ctx, objectName, options)
		if err != nil {
			return nil, err
		}
		urlIters = append(urlIters, urlsIter)
	}
	return &combinedURLsIter{iters: urlIters, used: make([]URLsIter, 0, len(urlIters))}, nil
}

func (c *combinedURLsIter) Peek(u *url.URL) (bool, error) {
	for len(c.iters) > 0 {
		iter := c.iters[0]
		if ok, err := iter.Peek(u); err != nil {
			return ok, err
		} else if ok {
			return true, nil
		} else {
			c.used = append(c.used, c.iters[0])
			c.iters = c.iters[1:]
		}
	}
	return false, nil
}

func (c *combinedURLsIter) Next() {
	if len(c.iters) > 0 {
		iter := c.iters[0]
		iter.Next()
	}
}

func (c *combinedURLsIter) Reset() {
	c.iters = append(c.used, c.iters...)
	c.used = make([]URLsIter, 0, cap(c.iters))
}

func (c *combinedURLsIter) Clone() URLsIter {
	new := combinedURLsIter{
		iters: make([]URLsIter, 0, cap(c.iters)),
		used:  make([]URLsIter, 0, cap(c.used)),
	}
	for _, iter := range c.iters {
		new.iters = append(new.iters, iter.Clone())
	}
	for _, iter := range c.used {
		new.used = append(new.used, iter.Clone())
	}
	return &new
}

// 为下载 URL 获取结果签名
func SignURLsProvider(provider DownloadURLsProvider, signer Signer, options *SignOptions) DownloadURLsProvider {
	return signedDownloadURLsProviders{provider, signer, options}
}

func (provider signedDownloadURLsProviders) GetURLsIter(ctx context.Context, objectName string, options *GenerateOptions) (URLsIter, error) {
	if urlsIter, err := provider.provider.GetURLsIter(ctx, objectName, options); err == nil {
		return SignURLs(ctx, urlsIter, provider.signer, provider.options), nil
	} else {
		return urlsIter, err
	}
}
