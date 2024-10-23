package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofrs/flock"
	"golang.org/x/sync/singleflight"
)

type (
	CacheValue interface {
		IsEqual(CacheValue) bool
		ShouldRefresh() bool
		IsValid() bool
	}

	Cache struct {
		compactInterval time.Duration
		cacheMap        map[string]cacheValue
		cacheMapMutex   sync.Mutex
		lastCompactTime time.Time
		persistentFile  *persistentFile
		group           singleflight.Group
		flushing        uint32
	}

	persistentFile struct {
		valueType          reflect.Type
		cacheFilePath      string
		persistentDuration time.Duration
		lastPersistentTime time.Time
		handleError        func(error)
	}

	cacheValue struct {
		Value     CacheValue `json:"value"`
		CreatedAt time.Time  `json:"created_at"`
	}

	unmarshalledCacheEntry struct {
		Key       string          `json:"key"`
		Value     json.RawMessage `json:"value"`
		CreatedAt time.Time       `json:"created_at"`
	}
)

func NewCache(compactInterval time.Duration) *Cache {
	return &Cache{
		compactInterval: compactInterval,
		cacheMap:        make(map[string]cacheValue),
		lastCompactTime: time.Now(),
	}
}

func NewPersistentCache(
	valueType reflect.Type,
	persistentFilePath string,
	compactInterval time.Duration,
	persistentDuration time.Duration,
	handleError func(error),
) (*Cache, error) {
	err := os.MkdirAll(filepath.Dir(persistentFilePath), 0700)
	if err != nil {
		return nil, err
	}
	unlockFunc, err := lockCachePersistentFile(persistentFilePath, false, handleError)
	if err != nil {
		return nil, err
	}
	defer unlockFunc()

	file, closeFunc, err := openCachePersistentFile(persistentFilePath, handleError)
	if err != nil {
		return nil, err
	}
	defer closeFunc()

	cacheMap, err := loadCacheMapFrom(valueType, file)
	if err != nil {
		return nil, err
	}

	return &Cache{
		persistentFile: &persistentFile{
			valueType:          valueType,
			cacheFilePath:      persistentFilePath,
			persistentDuration: persistentDuration,
			lastPersistentTime: time.Now(),
			handleError:        handleError,
		},
		cacheMap:        cacheMap,
		compactInterval: compactInterval,
		lastCompactTime: time.Now(),
	}, nil
}

type GetResult uint8

const (
	GetResultFromCache                GetResult = 0
	GetResultFromCacheAndRefreshAsync GetResult = 1
	GetResultFromFallback             GetResult = 2
	GetResultFromInvalidCache         GetResult = 3
	NoResultGot                       GetResult = 4
)

func (cache *Cache) Get(key string, fallback func() (CacheValue, error)) (CacheValue, GetResult) {
	cache.cacheMapMutex.Lock()
	value, ok := cache.cacheMap[key]
	cache.cacheMapMutex.Unlock()

	defer func() {
		go cache.flush()
	}()

	if ok && value.Value.IsValid() {
		if value.Value.ShouldRefresh() {
			cache.doFallbackAsync(key, fallback)
			return value.Value, GetResultFromCacheAndRefreshAsync
		} else {
			return value.Value, GetResultFromCache
		}
	}

	newValue, err := cache.doFallback(key, fallback)
	if err != nil {
		if ok {
			return value.Value, GetResultFromInvalidCache
		} else {
			return nil, NoResultGot
		}
	}
	cache.set(key, newValue, false)
	return newValue, GetResultFromFallback
}

func (cache *Cache) doFallback(key string, fallback func() (CacheValue, error)) (CacheValue, error) {
	newValue, err, _ := cache.group.Do(key, func() (interface{}, error) { return fallback() })
	if err != nil {
		return nil, err
	}
	return newValue.(CacheValue), nil
}

func (cache *Cache) doFallbackAsync(key string, fallback func() (CacheValue, error)) {
	go func() {
		newValue, err := cache.doFallback(key, fallback)
		if err == nil {
			cache.Set(key, newValue)
		}
	}()
}

func (cache *Cache) Set(key string, value CacheValue) {
	cache.set(key, value, true)
}

func (cache *Cache) set(key string, value CacheValue, willFlushAsync bool) {
	cache.checkType(value)
	if !value.IsValid() {
		return
	}

	now := time.Now()
	cache.cacheMapMutex.Lock()
	cache.cacheMap[key] = cacheValue{Value: value, CreatedAt: now}
	cache.cacheMapMutex.Unlock()

	if willFlushAsync {
		go cache.flush()
	}
}

func (cache *Cache) checkType(cacheValue CacheValue) {
	if pf := cache.persistentFile; pf != nil {
		if cacheValueType := reflect.TypeOf(cacheValue); !cacheValueType.AssignableTo(pf.valueType) {
			panic(fmt.Sprintf("cannot assign %s to %s", cacheValueType, pf.valueType))
		}
	}
}

func (cache *Cache) flush() {
	if !atomic.CompareAndSwapUint32(&cache.flushing, 0, 1) {
		return
	}
	defer atomic.StoreUint32(&cache.flushing, 0)

	if cache.lastCompactTime.Add(cache.compactInterval).Before(time.Now()) {
		cache.doCompact()
		cache.lastCompactTime = time.Now()
	}

	if pf := cache.persistentFile; pf != nil {
		if pf.lastPersistentTime.Add(pf.persistentDuration).Before(time.Now()) {
			cache.doPersistent()
			pf.lastPersistentTime = time.Now()
		}
	}
}

func (cache *Cache) doCompact() {
	cache.cacheMapMutex.Lock()
	defer cache.cacheMapMutex.Unlock()

	var toDeleted []string
	for key, value := range cache.cacheMap {
		if !value.Value.IsValid() {
			toDeleted = append(toDeleted, key)
		}
	}
	for _, toDeletedKey := range toDeleted {
		delete(cache.cacheMap, toDeletedKey)
	}
}

func (cache *Cache) doPersistent() {
	var (
		cacheFilePath = cache.persistentFile.cacheFilePath
		handleError   = cache.persistentFile.handleError
		valueType     = cache.persistentFile.valueType
	)

	unlockFunc, err := lockCachePersistentFile(cacheFilePath, true, handleError)
	if err != nil {
		return
	}
	defer unlockFunc()

	cache.cacheMapMutex.Lock()
	defer cache.cacheMapMutex.Unlock()

	file, closeFunc, err := openCachePersistentFile(cacheFilePath, handleError)
	if err != nil {
		return
	}
	defer closeFunc()

	newCacheMap, err := loadCacheMapFrom(valueType, file)
	if err != nil {
		if handleError != nil {
			handleError(err)
		}
		return
	}
	if isCacheMapEqual(cache.cacheMap, newCacheMap) {
		return
	}
	mergeCacheMap(cache.cacheMap, newCacheMap)

	if _, err = file.Seek(0, io.SeekStart); err != nil {
		if handleError != nil {
			handleError(err)
		}
		return
	}
	if err = file.Truncate(0); err != nil {
		if handleError != nil {
			handleError(err)
		}
		return
	}
	if err = saveCacheMapTo(file, cache.cacheMap); err != nil && handleError != nil {
		handleError(err)
	}
}

func loadCacheMapFrom(valueType reflect.Type, r io.Reader) (map[string]cacheValue, error) {
	decoder := json.NewDecoder(r)
	cacheMap := make(map[string]cacheValue)
	for decoder.More() {
		var entry unmarshalledCacheEntry
		if err := decoder.Decode(&entry); err != nil {
			return nil, err
		}
		ptrValue := reflect.New(valueType)
		if err := json.Unmarshal(entry.Value, ptrValue.Interface()); err != nil {
			return nil, err
		}
		value := ptrValue.Elem().Interface().(CacheValue)
		if value.IsValid() {
			cacheMap[entry.Key] = cacheValue{Value: value, CreatedAt: entry.CreatedAt}
		}
	}
	return cacheMap, nil
}

func isCacheMapEqual(left, right map[string]cacheValue) bool {
	if len(left) != len(right) {
		return false
	}
	leftKeys := sort.StringSlice(getCacheMapKeys(left))
	leftKeys.Sort()
	rightKeys := sort.StringSlice(getCacheMapKeys(right))
	rightKeys.Sort()
	if !reflect.DeepEqual(leftKeys, rightKeys) {
		return false
	}
	for _, key := range leftKeys {
		leftValue := left[key]
		rightValue := right[key]
		if !leftValue.Value.IsEqual(rightValue.Value) {
			return false
		}
	}
	return true
}

func getCacheMapKeys(m map[string]cacheValue) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func mergeCacheMap(left, right map[string]cacheValue) {
	for newKey, newValue := range right {
		existedCacheValue, exists := left[newKey]
		if exists && existedCacheValue.CreatedAt.Before(newValue.CreatedAt) || !exists {
			left[newKey] = newValue
		}
	}
}

func saveCacheMapTo(w io.Writer, m map[string]cacheValue) error {
	encoder := json.NewEncoder(w)
	for k, v := range m {
		rawMessage, err := json.Marshal(v.Value)
		if err != nil {
			return err
		}
		if err = encoder.Encode(unmarshalledCacheEntry{
			Key:       k,
			Value:     rawMessage,
			CreatedAt: v.CreatedAt,
		}); err != nil {
			return err
		}
	}
	return nil
}

func lockCachePersistentFile(cacheFilePath string, ex bool, handleError func(error)) (context.CancelFunc, error) {
	var (
		lockFilePath = cacheFilePath + ".lock"
		lockFile     = flock.New(lockFilePath)
		err          error
	)
	if ex {
		err = lockFile.Lock()
	} else {
		err = lockFile.RLock()
	}
	if err != nil {
		if handleError != nil {
			handleError(err)
		}
		return nil, err
	}
	return func() {
		if err := lockFile.Unlock(); err != nil && handleError != nil {
			handleError(err)
		}
	}, nil
}

func openCachePersistentFile(cacheFile string, handleError func(error)) (*os.File, context.CancelFunc, error) {
	file, err := os.OpenFile(cacheFile, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		if handleError != nil {
			handleError(err)
		}
		return nil, nil, err
	}
	return file, func() {
		if err := file.Close(); err != nil && handleError != nil {
			handleError(err)
		}
	}, nil
}
