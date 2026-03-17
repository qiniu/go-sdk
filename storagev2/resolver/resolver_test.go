//go:build unit
// +build unit

package resolver_test

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/resolver"
)

func TestDefaultResolver(t *testing.T) {
	ips, err := resolver.NewDefaultResolver().Resolve(context.Background(), "upload.qiniup.com")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	} else if len(ips) == 0 {
		t.Fatal("Unexpected empty ips")
	}
}

type mockResolver struct {
	m map[string][]net.IP
	c map[string]int
}

func (mr *mockResolver) Resolve(ctx context.Context, host string) ([]net.IP, error) {
	mr.c[host]++
	return mr.m[host], nil
}

func (mr *mockResolver) FeedbackGood(context.Context, string, []net.IP) {}

func (mr *mockResolver) FeedbackBad(context.Context, string, []net.IP) {}

func TestCacheResolver(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	mr := &mockResolver{m: map[string][]net.IP{"upload.qiniup.com": {net.IPv4(1, 1, 1, 1), net.IPv4(1, 1, 2, 2)}}, c: make(map[string]int)}
	resolver, err := resolver.NewCacheResolver(mr, &resolver.CacheResolverConfig{
		PersistentFilePath: tmpFile.Name(),
		CacheRefreshAfter:  3 * time.Second,
		CacheLifetime:      2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		ips, err := resolver.Resolve(context.Background(), "upload.qiniup.com")
		if err != nil {
			t.Fatal(err)
		}
		if len(ips) != 2 || !ips[0].Equal(net.IPv4(1, 1, 1, 1)) || !ips[1].Equal(net.IPv4(1, 1, 2, 2)) {
			t.Fatal("Unexpected ips")
		}
	}
	if mr.c["upload.qiniup.com"] != 1 {
		t.Fatal("Unexpected cache")
	}

	time.Sleep(1000 * time.Millisecond)
	resolver.FeedbackGood(context.Background(), "upload.qiniup.com", []net.IP{net.IPv4(1, 1, 1, 1)})
	time.Sleep(1500 * time.Millisecond)
	ips, err := resolver.Resolve(context.Background(), "upload.qiniup.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(ips) != 1 || !ips[0].Equal(net.IPv4(1, 1, 1, 1)) {
		t.Fatal("Unexpected ips")
	}
	if mr.c["upload.qiniup.com"] != 1 {
		t.Fatal("Unexpected cache")
	}
}

func TestCacheResolverMaxLifetime(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	mr := &mockResolver{
		m: map[string][]net.IP{"upload.qiniup.com": {net.IPv4(1, 1, 1, 1)}},
		c: make(map[string]int),
	}
	r, err := resolver.NewCacheResolver(mr, &resolver.CacheResolverConfig{
		PersistentFilePath: tmpFile.Name(),
		CacheRefreshAfter:  2 * time.Second,
		CacheLifetime:      5 * time.Second,
		CacheMaxLifetime:   3 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	// 首次解析，触发 DNS 查询
	ips, err := r.Resolve(context.Background(), "upload.qiniup.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(ips) != 1 || !ips[0].Equal(net.IPv4(1, 1, 1, 1)) {
		t.Fatal("Unexpected ips")
	}
	if mr.c["upload.qiniup.com"] != 1 {
		t.Fatalf("Expected 1 resolve call, got %d", mr.c["upload.qiniup.com"])
	}

	// 每隔 500ms 调用 FeedbackGood 延长缓存，持续 4 秒（超过 CacheMaxLifetime 3s）
	for i := 0; i < 8; i++ {
		time.Sleep(500 * time.Millisecond)
		r.FeedbackGood(context.Background(), "upload.qiniup.com", []net.IP{net.IPv4(1, 1, 1, 1)})
	}

	// 此时 CacheMaxLifetime（3s）已过，FeedbackGood 无法继续推迟 RefreshAfter
	// 下次 Resolve 应触发异步 DNS 重新查询
	ips, err = r.Resolve(context.Background(), "upload.qiniup.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(ips) != 1 || !ips[0].Equal(net.IPv4(1, 1, 1, 1)) {
		t.Fatal("Unexpected ips")
	}

	// 等待异步刷新完成
	time.Sleep(1 * time.Second)

	if mr.c["upload.qiniup.com"] != 2 {
		t.Fatalf("Expected 2 resolve calls after max lifetime, got %d", mr.c["upload.qiniup.com"])
	}
}
