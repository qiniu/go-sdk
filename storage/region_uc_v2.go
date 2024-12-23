package storage

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/qiniu/go-sdk/v7/client"
	"github.com/qiniu/go-sdk/v7/internal/clientv2"
	"github.com/qiniu/go-sdk/v7/storagev2/apis"
	"github.com/qiniu/go-sdk/v7/storagev2/backoff"
	"github.com/qiniu/go-sdk/v7/storagev2/chooser"
	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/resolver"
	"github.com/qiniu/go-sdk/v7/storagev2/retrier"
)

// 此处废弃，但为了兼容老版本，单独放置一个文件

// UcQueryRet 为查询请求的回复
type UcQueryRet struct {
	TTL       int                            `json:"ttl"`
	Io        map[string]map[string][]string `json:"-"`
	IoInfo    map[string]UcQueryIo           `json:"io"`
	IoSrcInfo map[string]UcQueryIo           `json:"io_src"`
	Up        map[string]UcQueryUp           `json:"up"`
	RsInfo    map[string]UcQueryServerInfo   `json:"rs"`
	RsfInfo   map[string]UcQueryServerInfo   `json:"rsf"`
	ApiInfo   map[string]UcQueryServerInfo   `json:"api"`
}

type tmpUcQueryRet UcQueryRet

func (uc *UcQueryRet) UnmarshalJSON(data []byte) error {
	var tmp tmpUcQueryRet
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	uc.TTL = tmp.TTL
	uc.IoInfo = tmp.IoInfo
	uc.IoSrcInfo = tmp.IoSrcInfo
	uc.Up = tmp.Up
	uc.RsInfo = tmp.RsInfo
	uc.RsfInfo = tmp.RsfInfo
	uc.ApiInfo = tmp.ApiInfo
	uc.setup()
	return nil
}

func (uc *UcQueryRet) setup() {
	if uc.Io != nil || uc.IoInfo == nil {
		return
	}

	uc.Io = make(map[string]map[string][]string)
	ioSrc := uc.IoInfo["src"].toMapWithoutInfo()
	if len(ioSrc) > 0 {
		uc.Io["src"] = ioSrc
	}

	ioOldSrc := uc.IoInfo["old_src"].toMapWithoutInfo()
	if len(ioOldSrc) > 0 {
		uc.Io["old_src"] = ioOldSrc
	}
}

type UcQueryUp = UcQueryServerInfo
type UcQueryIo = UcQueryServerInfo

// UcQueryServerInfo 为查询请求回复中的上传域名信息
type UcQueryServerInfo struct {
	Main   []string `json:"main,omitempty"`
	Backup []string `json:"backup,omitempty"`
	Info   string   `json:"info,omitempty"`
}

func (io UcQueryServerInfo) toMapWithoutInfo() map[string][]string {

	ret := make(map[string][]string)
	if len(io.Main) > 0 {
		ret["main"] = io.Main
	}

	if len(io.Backup) > 0 {
		ret["backup"] = io.Backup
	}

	return ret
}

var ucQueryV2Group singleflight.Group

type regionV2CacheValue struct {
	Region   *Region   `json:"region"`
	Deadline time.Time `json:"deadline"`
}

type regionV2CacheMap map[string]regionV2CacheValue

const regionV2CacheFileName = "query_v2_00.cache.json"

var (
	regionV2CachePath     = filepath.Join(os.TempDir(), "qiniu-golang-sdk", regionV2CacheFileName)
	regionV2Cache         sync.Map
	regionV2CacheLock     sync.RWMutex
	regionV2CacheSyncLock sync.Mutex
	regionV2CacheLoaded   bool = false
)

func setRegionV2CachePath(newPath string) {
	cacheDir := filepath.Dir(newPath)
	if len(cacheDir) == 0 {
		return
	}

	regionV2CacheLock.Lock()
	defer regionV2CacheLock.Unlock()

	regionV2CachePath = filepath.Join(cacheDir, regionV2CacheFileName)
	regionV2CacheLoaded = false
}

func loadRegionV2Cache() {
	cacheFile, err := os.Open(regionV2CachePath)
	if err != nil {
		return
	}
	defer cacheFile.Close()

	var cacheMap regionV2CacheMap
	if err = json.NewDecoder(cacheFile).Decode(&cacheMap); err != nil {
		return
	}
	for cacheKey, cacheValue := range cacheMap {
		regionV2Cache.Store(cacheKey, cacheValue)
	}
}

func storeRegionV2Cache() {
	err := os.MkdirAll(filepath.Dir(regionV2CachePath), 0700)
	if err != nil {
		return
	}

	cacheFile, err := os.OpenFile(regionV2CachePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return
	}
	defer cacheFile.Close()

	cacheMap := make(regionV2CacheMap)
	regionV2Cache.Range(func(cacheKey, cacheValue interface{}) bool {
		cacheMap[cacheKey.(string)] = cacheValue.(regionV2CacheValue)
		return true
	})
	if err = json.NewEncoder(cacheFile).Encode(cacheMap); err != nil {
		return
	}
}

type UCApiOptions struct {
	// 是否使用 HTTPS 协议
	UseHttps bool

	// 是否加速上传
	AccelerateUploading bool

	// 单域名重试次数
	RetryMax int

	// api 请求的域名
	Hosts []string

	// 主备域名冻结时间（默认：600s），当一个域名请求失败（单个域名会被重试 TryTimes 次），会被冻结一段时间，使用备用域名进行重试，在冻结时间内，域名不能被使用，当一个操作中所有域名竣备冻结操作不在进行重试，返回最后一次操作的错误。
	HostFreezeDuration time.Duration

	// api 请求使用的 client
	Client *client.Client

	// api 使用的域名解析器
	Resolver resolver.Resolver

	// api 使用的 IP 选择器
	Chooser chooser.Chooser

	// api 使用的退避器
	Backoff backoff.Backoff

	// api 使用的重试器
	Retrier retrier.Retrier

	// 签名前回调函数
	BeforeSign func(*http.Request)

	// 签名后回调函数
	AfterSign func(*http.Request)

	// 签名失败回调函数
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

func (options *UCApiOptions) getApiStorageClient() *apis.Storage {
	return apis.NewStorage(&http_client.Options{
		Interceptors:        []clientv2.Interceptor{},
		UseInsecureProtocol: !options.UseHttps,
		AccelerateUploading: options.AccelerateUploading,
		Resolver:            options.Resolver,
		Chooser:             options.Chooser,
		HostRetryConfig:     &clientv2.RetryConfig{RetryMax: options.RetryMax, Retrier: options.Retrier},
		HostsRetryConfig:    &clientv2.RetryConfig{Retrier: options.Retrier},
		HostFreezeDuration:  options.HostFreezeDuration,
		BeforeSign:          options.BeforeSign,
		AfterSign:           options.AfterSign,
		SignError:           options.SignError,
		BeforeResolve:       options.BeforeResolve,
		AfterResolve:        options.AfterResolve,
		ResolveError:        options.ResolveError,
		BeforeBackoff:       options.BeforeBackoff,
		AfterBackoff:        options.AfterBackoff,
		BeforeRequest:       options.BeforeRequest,
		AfterResponse:       options.AfterResponse,
	})
}

func (options *UCApiOptions) getApiOptions() *apis.Options {
	return &apis.Options{OverwrittenBucketHosts: getUcEndpointProvider(options.UseHttps, options.Hosts)}
}

func DefaultUCApiOptions() UCApiOptions {
	return UCApiOptions{
		UseHttps: true,
	}
}

func getRegionByV2(ak, bucket string, options UCApiOptions) (*Region, error) {
	regionV2CacheLock.RLock()
	if regionV2CacheLoaded {
		regionV2CacheLock.RUnlock()
	} else {
		regionV2CacheLock.RUnlock()
		func() {
			regionV2CacheLock.Lock()
			defer regionV2CacheLock.Unlock()

			if !regionV2CacheLoaded {
				loadRegionV2Cache()
				regionV2CacheLoaded = true
			}
		}()
	}

	regionCacheKey := makeRegionCacheKey(ak, bucket, options.Hosts, options.AccelerateUploading)
	// check from cache
	if v, ok := regionV2Cache.Load(regionCacheKey); ok && time.Now().Before(v.(regionV2CacheValue).Deadline) {
		return v.(regionV2CacheValue).Region, nil
	}

	newRegion, err, _ := ucQueryV2Group.Do(regionCacheKey, func() (interface{}, error) {
		region, ttl, err := _getRegionByV2WithoutCache(ak, bucket, options)
		if err != nil {
			return nil, fmt.Errorf("query region error, %s", err.Error())
		}

		regionV2Cache.Store(regionCacheKey, regionV2CacheValue{
			Region:   region,
			Deadline: time.Now().Add(time.Duration(ttl) * time.Second),
		})

		regionV2CacheSyncLock.Lock()
		defer regionV2CacheSyncLock.Unlock()

		storeRegionV2Cache()
		return region, nil
	})
	if newRegion == nil {
		return nil, err
	}

	return newRegion.(*Region), err
}

func makeRegionCacheKey(ak, bucket string, ucHosts []string, accelerateUploading bool) string {
	hostStrings := fmt.Sprintf("%v", ucHosts)
	s := fmt.Sprintf("%s:%s:%x", ak, bucket, md5.Sum([]byte(hostStrings)))
	if accelerateUploading {
		s += ":1"
	}
	return s
}

func _getRegionByV2WithoutCache(ak, bucket string, options UCApiOptions) (*Region, int64, error) {
	response, err := options.getApiStorageClient().QueryBucketV2(
		context.Background(),
		&apis.QueryBucketV2Request{
			Bucket:    bucket,
			AccessKey: ak,
		},
		options.getApiOptions(),
	)
	if err != nil {
		return nil, 0, err
	}
	var srcUpHosts, cdnUpHosts []string
	var rsHost, rsfHost, apiHost, ioVipHost, ioSrcHost string
	if options.AccelerateUploading && len(response.UpDomains.AcceleratedUpDomains.MainAcceleratedUpDomains) > 0 {
		srcUpHosts = make([]string, 0,
			len(response.UpDomains.AcceleratedUpDomains.MainAcceleratedUpDomains)+
				len(response.UpDomains.AcceleratedUpDomains.BackupAcceleratedUpDomains)+
				len(response.UpDomains.OldAcceleratedDomains.OldMainAcceleratedUpDomains)+
				len(response.UpDomains.SourceUpDomains.MainSourceUpDomains)+
				len(response.UpDomains.SourceUpDomains.BackupSourceUpDomains)+
				len(response.UpDomains.OldSourceDomains.OldMainSourceUpDomains))
		srcUpHosts = append(srcUpHosts, response.UpDomains.AcceleratedUpDomains.MainAcceleratedUpDomains...)
		srcUpHosts = append(srcUpHosts, response.UpDomains.AcceleratedUpDomains.BackupAcceleratedUpDomains...)
		srcUpHosts = append(srcUpHosts, response.UpDomains.OldAcceleratedDomains.OldMainAcceleratedUpDomains...)
		srcUpHosts = append(srcUpHosts, response.UpDomains.SourceUpDomains.MainSourceUpDomains...)
		srcUpHosts = append(srcUpHosts, response.UpDomains.SourceUpDomains.BackupSourceUpDomains...)
		srcUpHosts = append(srcUpHosts, response.UpDomains.OldSourceDomains.OldMainSourceUpDomains...)
		cdnUpHosts = make([]string, 0, len(response.UpDomains.AcceleratedUpDomains.MainAcceleratedUpDomains))
		cdnUpHosts = append(cdnUpHosts, response.UpDomains.AcceleratedUpDomains.MainAcceleratedUpDomains...)
	} else {
		srcUpHosts = make([]string, 0,
			len(response.UpDomains.SourceUpDomains.MainSourceUpDomains)+
				len(response.UpDomains.SourceUpDomains.BackupSourceUpDomains)+
				len(response.UpDomains.OldSourceDomains.OldMainSourceUpDomains))
		srcUpHosts = append(srcUpHosts, response.UpDomains.SourceUpDomains.MainSourceUpDomains...)
		srcUpHosts = append(srcUpHosts, response.UpDomains.SourceUpDomains.BackupSourceUpDomains...)
		srcUpHosts = append(srcUpHosts, response.UpDomains.OldSourceDomains.OldMainSourceUpDomains...)
		cdnUpHosts = make([]string, 0,
			len(response.UpDomains.AcceleratedUpDomains.MainAcceleratedUpDomains)+
				len(response.UpDomains.AcceleratedUpDomains.BackupAcceleratedUpDomains)+
				len(response.UpDomains.OldAcceleratedDomains.OldMainAcceleratedUpDomains))
		cdnUpHosts = append(cdnUpHosts, response.UpDomains.AcceleratedUpDomains.MainAcceleratedUpDomains...)
		cdnUpHosts = append(cdnUpHosts, response.UpDomains.AcceleratedUpDomains.BackupAcceleratedUpDomains...)
		cdnUpHosts = append(cdnUpHosts, response.UpDomains.OldAcceleratedDomains.OldMainAcceleratedUpDomains...)
	}
	if len(response.RsDomains.AcceleratedRsDomains.MainAcceleratedRsDomains) > 0 {
		rsHost = response.RsDomains.AcceleratedRsDomains.MainAcceleratedRsDomains[0]
	}
	if len(response.RsfDomains.AcceleratedRsfDomains.MainAcceleratedRsfDomains) > 0 {
		rsfHost = response.RsfDomains.AcceleratedRsfDomains.MainAcceleratedRsfDomains[0]
	}
	if len(response.ApiDomains.AcceleratedApiDomains.MainAcceleratedApiDomains) > 0 {
		apiHost = response.ApiDomains.AcceleratedApiDomains.MainAcceleratedApiDomains[0]
	}
	if len(response.IoDomains.SourceIoDomains.MainSourceIoDomains) > 0 {
		ioVipHost = response.IoDomains.SourceIoDomains.MainSourceIoDomains[0]
	}
	if len(response.IoSrcDomains.SourceIoSrcDomains.MainSourceIoSrcDomains) > 0 {
		ioSrcHost = response.IoSrcDomains.SourceIoSrcDomains.MainSourceIoSrcDomains[0]
	}
	return &Region{
		SrcUpHosts: srcUpHosts,
		CdnUpHosts: cdnUpHosts,
		RsHost:     rsHost,
		RsfHost:    rsfHost,
		ApiHost:    apiHost,
		IovipHost:  ioVipHost,
		IoSrcHost:  ioSrcHost,
	}, response.TimeToLive, nil
}
