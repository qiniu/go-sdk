//go:build unit
// +build unit

package cache

import (
	"errors"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"
)

type integerCacheValue struct {
	Value        int       `json:"value"`
	RefreshAfter time.Time `json:"refresh_after"`
	ExpiredAt    time.Time `json:"expired_at"`
}

func (v integerCacheValue) ShouldRefresh() bool {
	return time.Now().After(v.RefreshAfter)
}

func (v integerCacheValue) IsValid() bool {
	return time.Now().Before(v.ExpiredAt)
}

func (left integerCacheValue) IsEqual(rightV CacheValue) bool {
	right, ok := rightV.(integerCacheValue)
	if !ok {
		return false
	}
	return left.Value == right.Value
}

func TestCache(t *testing.T) {
	cache := NewCache(200 * time.Millisecond)
	if value, result := cache.Get("key_1", func() (CacheValue, error) {
		return integerCacheValue{
			Value:        1,
			RefreshAfter: time.Now().Add(50 * time.Millisecond),
			ExpiredAt:    time.Now().Add(100 * time.Millisecond),
		}, nil
	}); result != GetResultFromFallback {
		t.Fatalf("unexpected result: %v", result)
	} else if v := value.(integerCacheValue).Value; v != 1 {
		t.Fatalf("unexpected cache value: %v", v)
	}

	if value, result := cache.Get("key_1", func() (CacheValue, error) {
		t.Fatal("should not call this fallback")
		return nil, nil
	}); result != GetResultFromCache {
		t.Fatalf("unexpected result: %v", result)
	} else if v := value.(integerCacheValue).Value; v != 1 {
		t.Fatalf("unexpected cache value: %v", v)
	}

	time.Sleep(50 * time.Millisecond)
	if value, result := cache.Get("key_1", func() (CacheValue, error) {
		return integerCacheValue{
			Value:        2,
			RefreshAfter: time.Now().Add(50 * time.Millisecond),
			ExpiredAt:    time.Now().Add(100 * time.Millisecond),
		}, nil
	}); result != GetResultFromCacheAndRefreshAsync {
		t.Fatalf("unexpected result: %v", result)
	} else if v := value.(integerCacheValue).Value; v != 1 {
		t.Fatalf("unexpected cache value: %v", v)
	}

	time.Sleep(250 * time.Millisecond)
	if value, result := cache.Get("key_1", func() (CacheValue, error) {
		return nil, errors.New("test error")
	}); result != GetResultFromInvalidCache {
		t.Fatalf("unexpected result: %v", result)
	} else if v := value.(integerCacheValue).Value; v != 2 {
		t.Fatalf("unexpected cache value: %v", v)
	}
	time.Sleep(150 * time.Millisecond)
	cache.flush()
	if _, result := cache.Get("key_1", func() (CacheValue, error) {
		return nil, errors.New("test error")
	}); result != NoResultGot {
		t.Fatalf("unexpected result: %v", result)
	}

	if value, result := cache.Get("key_2", func() (CacheValue, error) {
		return integerCacheValue{Value: 3, ExpiredAt: time.Now().Add(-100 * time.Millisecond)}, nil
	}); result != GetResultFromFallback {
		t.Fatalf("unexpected result: %v", result)
	} else if v := value.(integerCacheValue).Value; v != 3 {
		t.Fatalf("unexpected cache value: %v", v)
	}

	if value, result := cache.Get("key_3", func() (CacheValue, error) {
		return integerCacheValue{Value: 4, ExpiredAt: time.Now().Add(100 * time.Millisecond)}, nil
	}); result != GetResultFromFallback {
		t.Fatal("unexpected ok")
	} else if v := value.(integerCacheValue).Value; v != 4 {
		t.Fatalf("unexpected cache value: %v", v)
	}
}

func TestCachePersist(t *testing.T) {
	cacheFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer cacheFile.Close()
	defer os.Remove(cacheFile.Name())

	valueType := reflect.TypeOf(integerCacheValue{})
	cache, err := NewPersistentCache(
		valueType,
		cacheFile.Name(),
		100*time.Millisecond,
		100*time.Millisecond,
		func(err error) {
			t.Fatalf("no error are expected: %s", err)
		})
	if err != nil {
		t.Fatal(err)
	}
	if value, result := cache.Get("key_1", func() (CacheValue, error) {
		return integerCacheValue{
			Value:        1,
			RefreshAfter: time.Now().Add(100 * time.Millisecond),
			ExpiredAt:    time.Now().Add(200 * time.Millisecond),
		}, nil
	}); result != GetResultFromFallback {
		t.Fatal("unexpected ok")
	} else if v := value.(integerCacheValue).Value; v != 1 {
		t.Fatalf("unexpected cache value: %v", v)
	}

	time.Sleep(100 * time.Millisecond)
	cache.flush()

	cacheMap, err := loadCacheMapFrom(valueType, cacheFile)
	if err != nil {
		t.Fatal(err)
	}
	if value, ok := cacheMap["key_1"]; !ok {
		t.Fatalf("key_1 should be existed")
	} else if v := value.Value.(integerCacheValue).Value; v != 1 {
		t.Fatalf("unexpected cache value: %v", v)
	}

	time.Sleep(100 * time.Millisecond)
	cache.flush()

	cacheMap, err = loadCacheMapFrom(valueType, cacheFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(cacheMap) != 0 {
		t.Fatalf("key_1 should be deleted")
	}
}
