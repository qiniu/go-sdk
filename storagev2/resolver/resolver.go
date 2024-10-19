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

		// FeedbackGood 反馈一批 IP 地址请求成功
		FeedbackGood(context.Context, string, []net.IP)

		// FeedbackBad 反馈一批 IP 地址请求失败
		FeedbackBad(context.Context, string, []net.IP)
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

func (resolver customizedResolver) FeedbackGood(context.Context, string, []net.IP) {}
func (resolver customizedResolver) FeedbackBad(context.Context, string, []net.IP)  {}

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

func (resolver defaultResolver) FeedbackGood(context.Context, string, []net.IP) {}
func (resolver defaultResolver) FeedbackBad(context.Context, string, []net.IP)  {}

type (
	cacheResolver struct {
		resolver          Resolver
		cache             *cache.Cache
		cacheLifetime     time.Duration
		cacheRefreshAfter time.Duration
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

		// 缓存刷新时间（默认：80s）
		CacheRefreshAfter time.Duration
	}

	resolverCacheValueIP struct {
		IP        net.IP    `json:"ip"`
		ExpiredAt time.Time `json:"expired_at"`
	}

	resolverCacheValue struct {
		IPs          []resolverCacheValueIP `json:"ips"`
		RefreshAfter time.Time              `json:"refresh_after"`
		ExpiredAt    time.Time              `json:"expired_at"`
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
	cacheRefreshAfter := opts.CacheRefreshAfter
	if cacheRefreshAfter == time.Duration(0) {
		cacheRefreshAfter = 80 * time.Second
	}
	if resolver == nil {
		resolver = staticDefaultResolver
	}

	persistentCache, err := getPersistentCache(persistentFilePath, compactInterval, persistentDuration)
	if err != nil {
		return nil, err
	}
	return &cacheResolver{
		cache:             persistentCache,
		resolver:          resolver,
		cacheLifetime:     cacheLifetime,
		cacheRefreshAfter: cacheRefreshAfter,
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

func (resolver *cacheResolver) resolve(ctx context.Context, host string) (*resolverCacheValue, error) {
	if ips, err := resolver.resolver.Resolve(ctx, host); err != nil {
		return nil, err
	} else {
		now := time.Now()
		cacheValueIPs := make([]resolverCacheValueIP, len(ips))
		for i, ip := range ips {
			cacheValueIPs[i] = resolverCacheValueIP{IP: ip, ExpiredAt: now.Add(resolver.cacheLifetime)}
		}
		return &resolverCacheValue{
			IPs:          cacheValueIPs,
			RefreshAfter: now.Add(resolver.cacheRefreshAfter),
			ExpiredAt:    now.Add(resolver.cacheLifetime),
		}, nil
	}
}

func (resolver *cacheResolver) Resolve(ctx context.Context, host string) ([]net.IP, error) {
	lip, err := resolver.localIp()
	if err != nil {
		return nil, err
	}
	cacheKey := lip + ":" + host
	var rcv *resolverCacheValue
	if shouldByPassResolveCache(ctx) {
		if rcv, err = resolver.resolve(ctx, host); err != nil {
			return nil, err
		}
	} else {
		cacheValue, status := resolver.cache.Get(cacheKey, func() (cache.CacheValue, error) {
			var cacheValue cache.CacheValue
			cacheValue, err = resolver.resolve(ctx, host)
			return cacheValue, err
		})
		if status == cache.NoResultGot || status == cache.GetResultFromInvalidCache {
			return nil, err
		}
		rcv = cacheValue.(*resolverCacheValue)
	}
	now := time.Now()
	ips := make([]net.IP, 0, len(rcv.IPs))
	for _, cacheValueIP := range rcv.IPs {
		if cacheValueIP.ExpiredAt.After(now) {
			ips = append(ips, cacheValueIP.IP)
		}
	}
	if len(ips) < len(rcv.IPs) {
		newCacheValue := &resolverCacheValue{
			IPs:          make([]resolverCacheValueIP, 0, len(rcv.IPs)),
			RefreshAfter: rcv.RefreshAfter,
			ExpiredAt:    rcv.ExpiredAt,
		}
		for _, cacheValueIP := range rcv.IPs {
			if cacheValueIP.ExpiredAt.After(now) {
				newCacheValue.IPs = append(newCacheValue.IPs, cacheValueIP)
			}
		}
		resolver.cache.Set(cacheKey, newCacheValue)
	}

	return ips, nil
}

func (resolver cacheResolver) FeedbackGood(ctx context.Context, host string, ips []net.IP) {
	lip, err := resolver.localIp()
	if err != nil {
		return
	}
	cacheKey := lip + ":" + host
	cacheValue, status := resolver.cache.Get(cacheKey, func() (cache.CacheValue, error) {
		return nil, context.Canceled
	})
	if status == cache.GetResultFromCache || status == cache.GetResultFromCacheAndRefreshAsync {
		rcv := cacheValue.(*resolverCacheValue)
		now := time.Now()
		anyIPLiveLonger := false
		for i := range rcv.IPs {
			if isIPContains(ips, rcv.IPs[i].IP) {
				rcv.IPs[i].ExpiredAt = now.Add(resolver.cacheLifetime)
				anyIPLiveLonger = true
			}
		}
		if anyIPLiveLonger {
			rcv.RefreshAfter = now.Add(resolver.cacheRefreshAfter)
			rcv.ExpiredAt = now.Add(resolver.cacheLifetime)
			resolver.cache.Set(cacheKey, rcv)
		}
	}
}

func (resolver cacheResolver) FeedbackBad(context.Context, string, []net.IP) {}

func (left *resolverCacheValue) IsEqual(rightValue cache.CacheValue) bool {
	if right, ok := rightValue.(*resolverCacheValue); ok {
		if len(left.IPs) != len(right.IPs) {
			return false
		}
		for idx := range left.IPs {
			if !left.IPs[idx].IP.Equal(right.IPs[idx].IP) || !left.IPs[idx].ExpiredAt.Equal(right.IPs[idx].ExpiredAt) {
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

func (left *resolverCacheValue) ShouldRefresh() bool {
	return time.Now().After(left.RefreshAfter)
}

func (*cacheResolver) localIp() (string, error) {
	conn, err := net.Dial("udp", "223.5.5.5:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	return conn.LocalAddr().(*net.UDPAddr).IP.String(), nil
}

type (
	byPassResolverCacheContextKey   struct{}
	byPassResolverCacheContextValue struct{}
)

// WithByPassResolverCache 设置 Context 绕过 Resolver 内部缓存
func WithByPassResolverCache(ctx context.Context) context.Context {
	return context.WithValue(ctx, byPassResolverCacheContextKey{}, byPassResolverCacheContextValue{})
}

func shouldByPassResolveCache(ctx context.Context) bool {
	_, ok := ctx.Value(byPassResolverCacheContextKey{}).(byPassResolverCacheContextValue)
	return ok
}

func calcPersistentCacheCrc64(persistentFilePath string, compactInterval, persistentDuration time.Duration) uint64 {
	bytes := make([]byte, 0, 1024)
	bytes = strconv.AppendInt(bytes, int64(compactInterval), 36)
	bytes = strconv.AppendInt(bytes, int64(persistentDuration), 36)
	bytes = append(bytes, []byte(persistentFilePath)...)
	bytes = append(bytes, byte(0))
	return crc64.Checksum(bytes, crc64.MakeTable(crc64.ISO))
}

func isIPContains(t []net.IP, i net.IP) bool {
	for _, tt := range t {
		if tt.Equal(i) {
			return true
		}
	}
	return false
}
