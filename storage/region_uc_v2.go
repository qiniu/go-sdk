package storage

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/qiniu/go-sdk/v7/internal/clientv2"
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

func (uc *UcQueryRet) getOneHostFromInfo(info map[string]UcQueryIo) string {
	hosts := uc.getSrcHostsFromInfo(info)
	if len(hosts) > 0 {
		return hosts[0]
	}

	hosts = uc.getAccHostsFromInfo(info)
	if len(hosts) > 0 {
		return hosts[0]
	}

	return ""
}

func (uc *UcQueryRet) getSrcHostsFromInfo(info map[string]UcQueryIo) []string {
	ret := make([]string, 0)
	if info == nil {
		return ret
	}

	if len(info["src"].Main) > 0 {
		ret = append(ret, info["src"].Main...)
	}

	if len(info["src"].Backup) > 0 {
		ret = append(ret, info["src"].Backup...)
	}

	return ret
}

func (uc *UcQueryRet) getAccHostsFromInfo(info map[string]UcQueryIo) []string {
	ret := make([]string, 0)
	if info == nil {
		return ret
	}

	if len(info["acc"].Main) > 0 {
		ret = append(ret, info["acc"].Main...)
	}

	if len(info["acc"].Backup) > 0 {
		ret = append(ret, info["acc"].Backup...)
	}

	return ret
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
	if io.Main != nil && len(io.Main) > 0 {
		ret["main"] = io.Main
	}

	if io.Backup != nil && len(io.Backup) > 0 {
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
	UseHttps bool //
	RetryMax int  // 单域名重试次数
	// 主备域名冻结时间（默认：600s），当一个域名请求失败（单个域名会被重试 TryTimes 次），会被冻结一段时间，使用备用域名进行重试，在冻结时间内，域名不能被使用，当一个操作中所有域名竣备冻结操作不在进行重试，返回最后一次操作的错误。
	HostFreezeDuration time.Duration
}

func DefaultUCApiOptions() UCApiOptions {
	return UCApiOptions{
		UseHttps:           true,
		RetryMax:           0,
		HostFreezeDuration: 0,
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

	regionCacheKey := makeRegionCacheKey(ak, bucket)
	//check from cache
	if v, ok := regionV2Cache.Load(regionCacheKey); ok && time.Now().Before(v.(regionV2CacheValue).Deadline) {
		return v.(regionV2CacheValue).Region, nil
	}

	newRegion, err, _ := ucQueryV2Group.Do(regionCacheKey, func() (interface{}, error) {
		reqURL := fmt.Sprintf("%s/v2/query?ak=%s&bucket=%s", getUcHost(options.UseHttps), ak, bucket)

		var ret UcQueryRet
		c := getUCClient(ucClientConfig{
			IsUcQueryApi:       true,
			RetryMax:           options.RetryMax,
			HostFreezeDuration: options.HostFreezeDuration,
		}, nil)
		err := clientv2.DoAndDecodeJsonResponse(c, clientv2.RequestParams{
			Context:     context.Background(),
			Method:      clientv2.RequestMethodGet,
			Url:         reqURL,
			Header:      nil,
			BodyCreator: nil,
		}, &ret)
		if err != nil {
			return nil, fmt.Errorf("query region error, %s", err.Error())
		}

		region := &Region{
			SrcUpHosts: ret.getSrcHostsFromInfo(ret.Up),
			CdnUpHosts: ret.getAccHostsFromInfo(ret.Up),
			IovipHost:  ret.getOneHostFromInfo(ret.IoInfo),
			RsHost:     ret.getOneHostFromInfo(ret.RsInfo),
			RsfHost:    ret.getOneHostFromInfo(ret.RsfInfo),
			ApiHost:    ret.getOneHostFromInfo(ret.ApiInfo),
			IoSrcHost:  ret.getOneHostFromInfo(ret.IoSrcInfo),
		}

		regionV2Cache.Store(regionCacheKey, regionV2CacheValue{
			Region:   region,
			Deadline: time.Now().Add(time.Duration(ret.TTL) * time.Second),
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

func makeRegionCacheKey(ak, bucket string) string {
	return fmt.Sprintf("%s:%s:%x", ak, bucket, md5.Sum([]byte(getUcHost(false))))
}
