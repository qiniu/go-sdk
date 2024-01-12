package http_client

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

// Resolver 域名解析器的接口
type Resolver interface {
	// Resolve 解析域名的 IP 地址
	Resolve(context.Context, string) ([]net.IP, error)
}

// DefaultResolver 默认的域名解析器
type DefaultResolver struct {
}

func NewDefaultResolver() Resolver {
	return &DefaultResolver{}
}

func (resolver *DefaultResolver) Resolve(ctx context.Context, host string) ([]net.IP, error) {
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
		resolver Resolver
		cache    *cache.Cache
	}

	// CacheResolverOptions 缓存域名解析器选项
	CacheResolverOptions struct {
		// 压缩周期（默认：60s）
		CompactInterval time.Duration

		// 持久化路径（默认：$TMPDIR/qiniu-golang-sdk/resolver_01.cache.json）
		PersistentFilePath string

		// 持久化周期（默认：60s）
		PersistentDuration time.Duration

		// 主备域名冻结时间（默认：600s），当一个域名请求失败（单个域名会被重试 RetryMax 次），会被冻结一段时间，使用备用域名进行重试，在冻结时间内，域名不能被使用，当一个操作中所有域名竣备冻结操作不在进行重试，返回最后一次操作的错误。
		HostFreezeDuration time.Duration
	}

	resolverCacheValue struct {
		IPs       []net.IP  `json:"ips"`
		ExpiredAt time.Time `json:"expired_at"`
	}
)

const cacheFileName = "resolver_01.cache.json"

var (
	persistentCaches     map[uint64]*cache.Cache
	persistentCachesLock sync.Mutex
	defaultResolver      Resolver = &DefaultResolver{}
)

// NewCacheResolver 创建带缓存功能的域名解析器
func NewCacheResolver(resolver Resolver, opts *CacheResolverOptions) (Resolver, error) {
	if opts == nil {
		opts = &CacheResolverOptions{}
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
	if resolver == nil {
		resolver = defaultResolver
	}

	persistentCache, err := getPersistentCache(opts)
	if err != nil {
		return nil, err
	}
	return &cacheResolver{
		cache:    persistentCache,
		resolver: resolver,
	}, nil
}

func getPersistentCache(opts *CacheResolverOptions) (*cache.Cache, error) {
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
			reflect.TypeOf(&resolverCacheValue{}),
			opts.PersistentFilePath,
			opts.CompactInterval,
			opts.PersistentDuration,
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
			return &resolverCacheValue{IPs: ips, ExpiredAt: time.Now().Add(5 * time.Minute)}, nil
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

func (opts *CacheResolverOptions) toBytes() []byte {
	bytes := make([]byte, 0, 1024)
	bytes = strconv.AppendInt(bytes, int64(opts.CompactInterval), 36)
	bytes = strconv.AppendInt(bytes, int64(opts.PersistentDuration), 36)
	bytes = append(bytes, []byte(opts.PersistentFilePath)...)
	bytes = append(bytes, byte(0))
	return bytes
}

func calcPersistentCacheCrc64(opts *CacheResolverOptions) uint64 {
	return crc64.Checksum(opts.toBytes(), crc64.MakeTable(crc64.ISO))
}
