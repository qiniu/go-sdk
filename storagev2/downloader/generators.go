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
	staticDomainBasedURLsGenerator struct {
		domains []string
	}

	defaultSrcURLsGenerator struct {
		credentials credentials.CredentialsProvider
		query       region.BucketRegionsQuery
		ttl         time.Duration
	}

	domainsQueryURLsGenerator struct {
		storage     *apis.Storage
		cache       *cache.Cache
		credentials credentials.CredentialsProvider
		cacheTTL    time.Duration
	}

	combinedGenerators struct {
		generators []DownloadURLsGenerator
	}

	// 基于域名查询的下载 URL 生成器选项
	DomainsQueryURLsGeneratorOptions struct {
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
)

// 创建静态域名下载 URL 生成器
func NewStaticDomainBasedURLsGenerator(domains []string) DownloadURLsGenerator {
	return &staticDomainBasedURLsGenerator{domains}
}

func (g *staticDomainBasedURLsGenerator) GenerateURLs(_ context.Context, objectName string, options *GenerateOptions) ([]*url.URL, error) {
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
		u.RawFragment = ""
		urls = append(urls, u)
	}
	return urls, nil
}

// 创建默认源站域名下载 URL 生成器
func NewDefaultSrcURLsGenerator(credentials credentials.CredentialsProvider, bucketHosts region.Endpoints, ttl time.Duration, options *region.BucketRegionsQueryOptions) (DownloadURLsGenerator, error) {
	query, err := region.NewBucketRegionsQuery(bucketHosts, options)
	if err != nil {
		return nil, err
	}
	return &defaultSrcURLsGenerator{credentials, query, ttl}, nil
}

func (g *defaultSrcURLsGenerator) GenerateURLs(ctx context.Context, objectName string, options *GenerateOptions) ([]*url.URL, error) {
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
	urls, err := NewStaticDomainBasedURLsGenerator(ioSrcDomains).GenerateURLs(ctx, objectName, options)
	if err != nil {
		return nil, err
	}
	for _, u := range urls {
		u.RawQuery += signURL(u.String(), cred, time.Now().Add(g.ttl).Unix())
	}
	return urls, nil
}

const cacheFileName = "domain_v2_01.cache.json"

var (
	persistentCaches     map[uint64]*cache.Cache
	persistentCachesLock sync.Mutex
)

// 创建基于域名查询的下载 URL 生成器
func NewDomainsQueryURLsGenerator(options *DomainsQueryURLsGeneratorOptions) (DownloadURLsGenerator, error) {
	if options == nil {
		options = &DomainsQueryURLsGeneratorOptions{}
	}
	if options.Credentials == nil {
		return nil, errors.MissingRequiredFieldError{Name: "Credentials"}
	}
	if options.CompactInterval == time.Duration(0) {
		options.CompactInterval = time.Minute
	}
	if options.PersistentFilePath == "" {
		options.PersistentFilePath = filepath.Join(os.TempDir(), "qiniu-golang-sdk", cacheFileName)
	}
	if options.PersistentDuration == time.Duration(0) {
		options.PersistentDuration = time.Minute
	}
	if options.CacheTTL == time.Duration(0) {
		options.CacheTTL = time.Hour
	}
	persistentCache, err := getPersistentCache(options)
	if err != nil {
		return nil, err
	}

	storage := apis.NewStorage(&options.Options)
	return &domainsQueryURLsGenerator{storage, persistentCache, options.Credentials, options.CacheTTL}, nil
}

func (g *domainsQueryURLsGenerator) GenerateURLs(ctx context.Context, objectName string, options *GenerateOptions) ([]*url.URL, error) {
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
	return NewStaticDomainBasedURLsGenerator(domains).GenerateURLs(ctx, objectName, options)
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

func getPersistentCache(opts *DomainsQueryURLsGeneratorOptions) (*cache.Cache, error) {
	var (
		persistentCache *cache.Cache
		ok              bool
		err             error
	)

	crc64Value := calcPersistentCacheCrc64(opts)
	persistentCachesLock.Lock()
	defer persistentCachesLock.Unlock()

	if persistentCaches == nil {
		persistentCaches = make(map[uint64]*cache.Cache)
	}
	if persistentCache, ok = persistentCaches[crc64Value]; !ok {
		persistentCache, err = cache.NewPersistentCache(
			reflect.TypeOf(&domainCacheValue{}),
			opts.PersistentFilePath,
			opts.CompactInterval,
			opts.PersistentDuration,
			func(err error) {
				log.Warn(fmt.Sprintf("DomainsURLsGenerator persist error: %s", err))
			})
		if err != nil {
			return nil, err
		}
		persistentCaches[crc64Value] = persistentCache
	}
	return persistentCache, nil
}

func (opts *DomainsQueryURLsGeneratorOptions) toBytes() []byte {
	bytes := make([]byte, 0, 1024)
	bytes = strconv.AppendInt(bytes, int64(opts.CompactInterval), 36)
	bytes = append(bytes, []byte(opts.PersistentFilePath)...)
	bytes = append(bytes, byte(0))
	bytes = strconv.AppendInt(bytes, int64(opts.PersistentDuration), 36)
	return bytes
}

func calcPersistentCacheCrc64(opts *DomainsQueryURLsGeneratorOptions) uint64 {
	return crc64.Checksum(opts.toBytes(), crc64.MakeTable(crc64.ISO))
}

// 合并多个下载 URL 生成器
func CombineGenerators(generators []DownloadURLsGenerator) DownloadURLsGenerator {
	return &combinedGenerators{generators}
}

func (g *combinedGenerators) GenerateURLs(ctx context.Context, objectName string, options *GenerateOptions) ([]*url.URL, error) {
	all := make([]*url.URL, 0, 16)
	for _, generator := range g.generators {
		urls, err := generator.GenerateURLs(ctx, objectName, options)
		if err != nil {
			return nil, err
		}
		all = append(all, urls...)
	}
	return all, nil
}
