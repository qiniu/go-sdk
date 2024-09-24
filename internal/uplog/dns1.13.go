//go:build 1.13
// +build 1.13

package uplog

import "net"

func isDnsNotFoundError(dnsError *net.DNSError) bool {
	return dnsError.IsNotFound
}
