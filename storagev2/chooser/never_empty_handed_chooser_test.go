//go:build unit
// +build unit

package chooser_test

import (
	"context"
	"net"
	"testing"

	"github.com/alex-ant/gomath/rational"
	"github.com/qiniu/go-sdk/v7/storagev2/chooser"
)

func TestNeverEmptyHandedChooser(t *testing.T) {
	cs := chooser.NewNeverEmptyHandedChooser(chooser.NewIPChooser(nil), rational.New(1, 2))
	ips := cs.Choose(context.Background(), &chooser.ChooseOptions{
		IPs:    []net.IP{net.IPv4(1, 2, 3, 4), net.IPv4(1, 2, 3, 5), net.IPv4(5, 6, 7, 8), net.IPv4(5, 6, 7, 9)},
		Domain: "www.qiniu.com",
	})
	assertIPs(t, ips, []net.IP{net.IPv4(1, 2, 3, 4), net.IPv4(1, 2, 3, 5), net.IPv4(5, 6, 7, 8), net.IPv4(5, 6, 7, 9)})

	cs.FeedbackBad(context.Background(), &chooser.FeedbackOptions{
		IPs:    []net.IP{net.IPv4(1, 2, 3, 4), net.IPv4(1, 2, 3, 5), net.IPv4(5, 6, 7, 8), net.IPv4(5, 6, 7, 9)},
		Domain: "www.qiniu.com",
	})

	ips = cs.Choose(context.Background(), &chooser.ChooseOptions{
		IPs:    []net.IP{net.IPv4(1, 2, 3, 4), net.IPv4(1, 2, 3, 5), net.IPv4(5, 6, 7, 8), net.IPv4(5, 6, 7, 9)},
		Domain: "www.qiniu.com",
	})
	if len(ips) != 2 {
		t.Fatal("unexpected ips count")
	}
}
