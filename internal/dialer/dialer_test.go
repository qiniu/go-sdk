//go:build unit
// +build unit

package dialer_test

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/qiniu/go-sdk/v7/internal/dialer"
)

func TestDialContext(t *testing.T) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9901})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	now := time.Now()
	conn, err := dialer.DialContext(context.Background(), "tcp", []net.IP{net.IPv4(8, 8, 8, 8), net.IPv4(8, 8, 4, 4)}, "9901", dialer.DialOptions{Timeout: 3 * time.Second, KeepAlive: time.Second})
	if err != nil {
		if !os.IsTimeout(err) {
			t.Fatal("Unexpected error", err)
		}
	} else {
		conn.Close()
		t.Fatalf("Returning connection is unexpected")
	}

	if time.Since(now) < 3*time.Second || time.Since(now) > 4*time.Second {
		t.Fatalf("Unexpected time elapsed")
	}

	now = time.Now()
	conn, err = dialer.DialContext(context.Background(), "tcp", []net.IP{net.IPv4(8, 8, 8, 8), net.IPv4(8, 8, 4, 4), net.IPv4(127, 0, 0, 1)}, "9901", dialer.DialOptions{Timeout: 3 * time.Second, KeepAlive: time.Second})
	if err != nil {
		t.Fatal("Unexpected error", err)
	}
	conn.Close()

	if time.Since(now) < 2*time.Second || time.Since(now) > 3*time.Second {
		t.Fatalf("Unexpected time elapsed")
	}
}
