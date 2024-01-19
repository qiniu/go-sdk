//go:build unit
// +build unit

package chooser_test

import (
	"context"
	"net"
	"testing"

	"github.com/qiniu/go-sdk/v7/storagev2/chooser"
)

func TestDirectChooser(t *testing.T) {
	cs := chooser.NewDirectChooser()
	ips := cs.Choose(context.Background(), &chooser.ChooseOptions{
		IPs:    []net.IP{net.IPv4(1, 2, 3, 4), net.IPv4(5, 6, 7, 8)},
		Domain: "www.qiniu.com",
	})
	assertIPs(t, ips, []net.IP{net.IPv4(1, 2, 3, 4), net.IPv4(5, 6, 7, 8)})
}

func assertIPs(t *testing.T, ips []net.IP, expected []net.IP) {
	if len(ips) != len(expected) {
		t.Fatalf("unexpected ips count: actual=%v, expected=%v", ips, expected)
	}
	for i := range ips {
		if !ips[i].Equal(expected[i]) {
			t.Fatalf("unexpected ip: actual=%v, expected=%v", ips, expected)
		}
	}
}
