package hostprovider

import (
	"errors"
	"time"

	"github.com/qiniu/go-sdk/v7/internal/freezer"
)

var (
	ErrNoHostFound    = errors.New("no host found")
	ErrAllHostsFrozen = errors.New("all hosts are frozen")
)

type (
	HostProvider interface {
		Provider() (string, error)
		Freeze(host string, cause error, duration time.Duration) error
	}

	arrayHostProvider struct {
		hosts         []string
		freezer       freezer.Freezer
		lastFreezeErr error
	}
)

func NewWithHosts(hosts []string) HostProvider {
	return &arrayHostProvider{
		hosts:   hosts,
		freezer: freezer.New(),
	}
}

func (a *arrayHostProvider) Provider() (string, error) {
	if len(a.hosts) == 0 {
		return "", ErrNoHostFound
	}

	for _, host := range a.hosts {
		if a.freezer.Available(host) {
			return host, nil
		}
	}

	if a.lastFreezeErr != nil {
		return "", a.lastFreezeErr
	} else {
		return "", ErrAllHostsFrozen
	}
}

func (a *arrayHostProvider) Freeze(host string, cause error, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}

	a.lastFreezeErr = cause
	return a.freezer.Freeze(host, duration)
}
