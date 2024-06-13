package resolver

import (
	"context"
	"fmt"
	"hash/crc64"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/qiniu/go-sdk/v7/internal/cache"
	"github.com/qiniu/go-sdk/v7/internal/log"
)

type (
	// Resolver 域名解析器的接口
	Resolver interface {
		// Resolve 解析域名的 IP 地址
		Resolve(context.Context, string) ([]net.IP, error)
	}

	defaultResolver    struct{}
	customizedResolver struct {
		resolveFn func(context.Context, string) ([]net.IP, error)
	}
)

// NewResolver 创建自定义的域名解析器
func NewResolver(fn func(context.Context, string) ([]net.IP, error)) Resolver {
	return customizedResolver{resolveFn: fn}
}

func (resolver customizedResolver) Resolve(ctx context.Context, host string) ([]net.IP, error) {
	return resolver.resolveFn(ctx, host)
}

// NewDefaultResolver 创建默认的域名解析器
func NewDefaultResolver() Resolver {
	return &defaultResolver{}
}

func (resolver *defaultResolver) Resolve(ctx context.Context, host string) ([]net.IP, error) {
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	ips := make([]net.IP, len(addrs))
	for i, ia := range addrs {
		ips[i] = ia.IP
	}
	return ips, nil
}

type (
	cacheResolver struct {
		resolver      Resolver
		cache         *cache.Cache
		cacheLifetime time.Duration
	}

	// CacheResolverConfig 缓存域名解析器选项
	CacheResolverConfig struct {
		// 压缩周期（默认：60s）
		CompactInterval time.Duration

		// 持久化路径（默认：$TMPDIR/qiniu-golang-sdk/resolver_01.cache.json）
		PersistentFilePath string

		// 持久化周期（默认：60s）
		PersistentDuration time.Duration

		// 缓存有效期（默认：120s）
		CacheLifetime time.Duration
	}

	resolverCacheValue struct {
		IPs       []net.IP  `json:"ips"`
		ExpiredAt time.Time `json:"expired_at"`
	}
)

const cacheFileName = "resolver_01.cache.json"

var (
	persistentCaches      map[uint64]*cache.Cache
	persistentCachesLock  sync.Mutex
	staticDefaultResolver Resolver = &defaultResolver{}
)

// NewCacheResolver 创建带缓存功能的域名解析器
func NewCacheResolver(resolver Resolver, opts *CacheResolverConfig) (Resolver, error) {
	if opts == nil {
		opts = &CacheResolverConfig{}
	}
	compactInterval := opts.CompactInterval
	if compactInterval == time.Duration(0) {
		compactInterval = 60 * time.Second
	}
	persistentFilePath := opts.PersistentFilePath
	if persistentFilePath == "" {
		persistentFilePath = filepath.Join(os.TempDir(), "qiniu-golang-sdk", cacheFileName)
	}
	persistentDuration := opts.PersistentDuration
	if persistentDuration == time.Duration(0) {
		persistentDuration = 60 * time.Second
	}
	cacheLifetime := opts.CacheLifetime
	if cacheLifetime == time.Duration(0) {
		cacheLifetime = 120 * time.Second
	}
	if resolver == nil {
		resolver = staticDefaultResolver
	}

	persistentCache, err := getPersistentCache(persistentFilePath, compactInterval, persistentDuration)
	if err != nil {
		return nil, err
	}
	return &cacheResolver{
		cache:         persistentCache,
		resolver:      resolver,
		cacheLifetime: opts.CacheLifetime,
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
			reflect.TypeOf(&resolverCacheValue{}),
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

func (resolver *cacheResolver) Resolve(ctx context.Context, host string) ([]net.IP, error) {
	lip, err := resolver.localIp(host)
	if err != nil {
		return nil, err
	}
	cacheValue, status := resolver.cache.Get(lip+":"+host, func() (cache.CacheValue, error) {
		var ips []net.IP
		if ips, err = resolver.resolver.Resolve(ctx, host); err != nil {
			return nil, err
		} else {
			return &resolverCacheValue{IPs: ips, ExpiredAt: time.Now().Add(resolver.cacheLifetime)}, nil
		}
	})
	if status == cache.NoResultGot {
		return nil, err
	}
	return cacheValue.(*resolverCacheValue).IPs, nil
}

func (left *resolverCacheValue) IsEqual(rightValue cache.CacheValue) bool {
	if right, ok := rightValue.(*resolverCacheValue); ok {
		if len(left.IPs) != len(right.IPs) {
			return false
		}
		for idx := range left.IPs {
			if !left.IPs[idx].Equal(right.IPs[idx]) {
				return false
			}
		}
		return true
	}
	return false
}

func (left *resolverCacheValue) IsValid() bool {
	return time.Now().Before(left.ExpiredAt)
}

func (*cacheResolver) localIp(host string) (string, error) {
	conn, err := net.Dial("udp", host+":80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	return conn.LocalAddr().(*net.UDPAddr).IP.String(), nil
}

func calcPersistentCacheCrc64(persistentFilePath string, compactInterval, persistentDuration time.Duration) uint64 {
	bytes := make([]byte, 0, 1024)
	bytes = strconv.AppendInt(bytes, int64(compactInterval), 36)
	bytes = strconv.AppendInt(bytes, int64(persistentDuration), 36)
	bytes = append(bytes, []byte(persistentFilePath)...)
	bytes = append(bytes, byte(0))
	return crc64.Checksum(bytes, crc64.MakeTable(crc64.ISO))
}
