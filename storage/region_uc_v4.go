package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/qiniu/go-sdk/v7/storagev2/apis"
	"github.com/qiniu/go-sdk/v7/storagev2/apis/query_bucket_v4"
)

var ucQueryV4Group singleflight.Group

type regionV4CacheValue struct {
	Regions  []*Region `json:"regions"`
	Deadline time.Time `json:"deadline"`
}

func (r *regionV4CacheValue) getRegions() []*Region {
	if r == nil {
		return nil
	}
	return r.Regions
}

type regionV4CacheMap map[string]regionV4CacheValue

const regionV4CacheFileName = "query_v4_00.cache.json"

var (
	regionV4CachePath     = filepath.Join(os.TempDir(), "qiniu-golang-sdk", regionV4CacheFileName)
	regionV4Cache         sync.Map
	regionV4CacheLock     sync.RWMutex
	regionV4CacheSyncLock sync.Mutex
	regionV4CacheLoaded   bool = false
)

func setRegionV4CachePath(newPath string) {
	cacheDir := filepath.Dir(newPath)
	if len(cacheDir) == 0 {
		return
	}

	regionV4CacheLock.Lock()
	defer regionV4CacheLock.Unlock()

	regionV4CachePath = filepath.Join(cacheDir, regionV4CacheFileName)
	regionV4CacheLoaded = false
}

func loadRegionV4Cache() {
	cacheFile, err := os.Open(regionV4CachePath)
	if err != nil {
		return
	}
	defer cacheFile.Close()

	var cacheMap regionV4CacheMap
	if err = json.NewDecoder(cacheFile).Decode(&cacheMap); err != nil {
		return
	}
	for cacheKey, cacheValue := range cacheMap {
		regionV4Cache.Store(cacheKey, cacheValue)
	}
}

func storeRegionV4Cache() {
	err := os.MkdirAll(filepath.Dir(regionV4CachePath), 0700)
	if err != nil {
		return
	}

	cacheFile, err := os.OpenFile(regionV4CachePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return
	}
	defer cacheFile.Close()

	cacheMap := make(regionV4CacheMap)
	regionV4Cache.Range(func(cacheKey, cacheValue interface{}) bool {
		cacheMap[cacheKey.(string)] = cacheValue.(regionV4CacheValue)
		return true
	})
	if err = json.NewEncoder(cacheFile).Encode(cacheMap); err != nil {
		return
	}
}

func getRegionByV4(ak, bucket string, options UCApiOptions) (*RegionGroup, error) {
	regionV4CacheLock.RLock()
	if regionV4CacheLoaded {
		regionV4CacheLock.RUnlock()
	} else {
		regionV4CacheLock.RUnlock()
		func() {
			regionV4CacheLock.Lock()
			defer regionV4CacheLock.Unlock()

			if !regionV4CacheLoaded {
				loadRegionV4Cache()
				regionV4CacheLoaded = true
			}
		}()
	}

	regionCacheKey := makeRegionCacheKey(ak, bucket, options.Hosts, options.AccelerateUploading)
	// check from cache
	if v, ok := regionV4Cache.Load(regionCacheKey); ok && time.Now().Before(v.(regionV4CacheValue).Deadline) {
		cacheValue, _ := v.(regionV4CacheValue)
		return NewRegionGroup(cacheValue.getRegions()...), nil
	}

	newRegion, err, _ := ucQueryV4Group.Do(regionCacheKey, func() (interface{}, error) {
		regions, ttl, err := _getRegionByV4WithoutCache(ak, bucket, options)
		if err != nil {
			return nil, fmt.Errorf("query region error, %s", err.Error())
		}
		regionV4Cache.Store(regionCacheKey, regionV4CacheValue{
			Regions:  regions,
			Deadline: time.Now().Add(time.Duration(ttl) * time.Second),
		})

		regionV4CacheSyncLock.Lock()
		defer regionV4CacheSyncLock.Unlock()

		storeRegionV4Cache()

		return NewRegionGroup(regions...), nil
	})

	if newRegion == nil {
		return nil, err
	}

	return newRegion.(*RegionGroup), err
}

func _getRegionByV4WithoutCache(ak, bucket string, options UCApiOptions) ([]*Region, int64, error) {
	toRegion := func(r *query_bucket_v4.BucketQueryHost) *Region {
		var rsHost, rsfHost, apiHost, ioVipHost, ioSrcHost string
		upDomains := make([]string, 0, len(r.UpDomains.AcceleratedUpDomains)+len(r.UpDomains.PreferedUpDomains)+len(r.UpDomains.AlternativeUpDomains))
		if options.AccelerateUploading && len(r.UpDomains.AcceleratedUpDomains) > 0 {
			upDomains = append(upDomains, r.UpDomains.AcceleratedUpDomains...)
		}
		upDomains = append(upDomains, r.UpDomains.PreferedUpDomains...)
		upDomains = append(upDomains, r.UpDomains.AlternativeUpDomains...)
		if len(r.RsDomains.PreferedRsDomains) > 0 {
			rsHost = r.RsDomains.PreferedRsDomains[0]
		}
		if len(r.RsfDomains.PreferedRsfDomains) > 0 {
			rsfHost = r.RsfDomains.PreferedRsfDomains[0]
		}
		if len(r.ApiDomains.PreferedApiDomains) > 0 {
			apiHost = r.ApiDomains.PreferedApiDomains[0]
		}
		if len(r.IoDomains.PreferedIoDomains) > 0 {
			ioVipHost = r.IoDomains.PreferedIoDomains[0]
		}
		if len(r.IoSrcDomains.PreferedIoSrcDomains) > 0 {
			ioSrcHost = r.IoSrcDomains.PreferedIoSrcDomains[0]
		}
		return &Region{
			SrcUpHosts: upDomains,
			CdnUpHosts: upDomains,
			RsHost:     rsHost,
			RsfHost:    rsfHost,
			ApiHost:    apiHost,
			IovipHost:  ioVipHost,
			IoSrcHost:  ioSrcHost,
		}
	}
	response, err := options.getApiStorageClient().QueryBucketV4(
		context.Background(),
		&apis.QueryBucketV4Request{
			Bucket:    bucket,
			AccessKey: ak,
		},
		options.getApiOptions(),
	)
	if err != nil {
		return nil, 0, err
	}
	regions := make([]*Region, 0, len(response.Hosts))
	var ttl int64 = math.MaxInt64
	for _, host := range response.Hosts {
		regions = append(regions, toRegion(&host))
		if ttl > host.TimeToLive {
			ttl = host.TimeToLive
		}
	}
	return regions, ttl, nil
}
