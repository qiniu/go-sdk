package dialer

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

type (
	DialOptions struct {
		Timeout   time.Duration
		KeepAlive time.Duration
	}

	eitherConnOrError struct {
		conn net.Conn
		err  error
	}

	dialerErrs struct {
		errs []error
	}
)

func DialContext(ctx context.Context, network string, ips []net.IP, port string, dialOptions DialOptions) (net.Conn, error) {
	var wg sync.WaitGroup
	resultsChan := make(chan eitherConnOrError, len(ips))
	cancels := make([]context.CancelFunc, 0, len(ips))
	err := &dialerErrs{errs: make([]error, 0, len(ips))}
	interval := dialOptions.Timeout / time.Duration(len(ips))

	defer func() {
		for _, cancel := range cancels {
			cancel()
		}
		wg.Wait()
		close(resultsChan)
	}()

	if len(ips) > 0 {
		ip := ips[0]
		ips = ips[1:]
		newCtx, newCancel := context.WithCancel(ctx)
		cancels = append(cancels, newCancel)
		wg.Add(1)
		dialContextAsync(newCtx, &wg, network, ip, port, DialOptions{Timeout: dialOptions.Timeout, KeepAlive: dialOptions.KeepAlive}, resultsChan)
	} else {
		return nil, errors.New("no ip could be dialed")
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if len(ips) > 0 {
				ip := ips[0]
				ips = ips[1:]
				newCtx, newCancel := context.WithCancel(ctx)
				cancels = append(cancels, newCancel)
				wg.Add(1)
				dialContextAsync(newCtx, &wg, network, ip, port, DialOptions{Timeout: dialOptions.Timeout - interval*time.Duration(len(cancels)-1), KeepAlive: dialOptions.KeepAlive}, resultsChan)
			} else {
				return nil, err
			}
		case connOrErr := <-resultsChan:
			if connOrErr.err != nil {
				err.errs = append(err.errs, connOrErr.err)
			} else if connOrErr.conn != nil {
				return connOrErr.conn, nil
			}
		}
	}
}

func dialContextSync(ctx context.Context, network string, ip net.IP, port string, dialOptions DialOptions) (net.Conn, error) {
	dialer := net.Dialer{Timeout: dialOptions.Timeout, KeepAlive: dialOptions.KeepAlive}
	newAddr := ip.String()
	if port != "" {
		newAddr = net.JoinHostPort(newAddr, port)
	}
	return dialer.DialContext(ctx, network, newAddr)
}

func dialContextAsync(ctx context.Context, wg *sync.WaitGroup, network string, ip net.IP, port string, dialOptions DialOptions, c chan<- eitherConnOrError) {
	go func() {
		defer wg.Done()
		conn, err := dialContextSync(ctx, network, ip, port, dialOptions)
		if err != nil {
			c <- eitherConnOrError{err: err}
		} else {
			c <- eitherConnOrError{conn: conn}
		}
	}()
}

func (e *dialerErrs) Error() string {
	if len(e.errs) > 0 {
		return e.errs[0].Error()
	} else {
		return context.DeadlineExceeded.Error()
	}
}

func (e *dialerErrs) Unwrap() error {
	if len(e.errs) > 0 {
		return e.errs[0]
	} else {
		return context.DeadlineExceeded
	}
}

func (e *dialerErrs) Timeout() bool {
	if len(e.errs) > 0 {
		if te, ok := e.errs[0].(interface{ Timeout() bool }); ok {
			return te.Timeout()
		} else {
			return false
		}
	} else {
		return true
	}
}
