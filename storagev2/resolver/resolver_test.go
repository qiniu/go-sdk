//go:build unit
// +build unit

package resolver_test

import (
	"context"
	"io/ioutil"
	"net"
	"os"
	"testing"

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

func TestCacheResolver(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	mr := &mockResolver{m: map[string][]net.IP{"upload.qiniup.com": {net.IPv4(1, 1, 1, 1)}}, c: make(map[string]int)}
	resolver, err := resolver.NewCacheResolver(mr, &resolver.CacheResolverConfig{
		PersistentFilePath: tmpFile.Name(),
	})

	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		ips, err := resolver.Resolve(context.Background(), "upload.qiniup.com")
		if err != nil {
			t.Fatal(err)
		}
		if len(ips) != 1 || !ips[0].Equal(net.IPv4(1, 1, 1, 1)) {
			t.Fatal("Unexpected ips")
		}
	}
	if mr.c["upload.qiniup.com"] != 1 {
		t.Fatal("Unexpected cache")
	}
}
