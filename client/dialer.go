package client

import (
	"context"
	"net"
	"time"
)

type (
	resolverContextKey          struct{}
	dialTimeoutContextKey       struct{}
	keepAliveIntervalContextKey struct{}
	resolverContextValue        struct {
		domain string
		ips    []net.IP
	}
)

func defaultDialFunc(ctx context.Context, network string, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		host = address
	}

	dialTimeout, ok := ctx.Value(dialTimeoutContextKey{}).(time.Duration)
	if !ok {
		dialTimeout = 30 * time.Second
	}
	keepAliveInterval, ok := ctx.Value(keepAliveIntervalContextKey{}).(time.Duration)
	if !ok {
		keepAliveInterval = 15 * time.Second
	}
	if resolved, ok := ctx.Value(resolverContextKey{}).(resolverContextValue); ok && len(resolved.ips) > 0 && resolved.domain == host {
		dialer := net.Dialer{Timeout: dialTimeout / time.Duration(len(resolved.ips)), KeepAlive: keepAliveInterval}
		for _, ip := range resolved.ips {
			newAddr := ip.String()
			if port != "" {
				newAddr = net.JoinHostPort(newAddr, port)
			}
			if conn, err := dialer.DialContext(ctx, network, newAddr); err == nil {
				return conn, nil
			}
		}
	}
	return (&net.Dialer{Timeout: dialTimeout, KeepAlive: keepAliveInterval}).DialContext(ctx, network, address)
}

func WithResolvedIPs(ctx context.Context, domain string, ips []net.IP) context.Context {
	return context.WithValue(ctx, resolverContextKey{}, resolverContextValue{domain: domain, ips: ips})
}

func WithDialTimeout(ctx context.Context, timeout time.Duration) context.Context {
	return context.WithValue(ctx, dialTimeoutContextKey{}, timeout)
}

func WithKeepAliveInterval(ctx context.Context, interval time.Duration) context.Context {
	return context.WithValue(ctx, keepAliveIntervalContextKey{}, interval)
}
