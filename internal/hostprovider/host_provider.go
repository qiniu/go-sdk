package hostprovider

import (
	"errors"
	"github.com/qiniu/go-sdk/v7/internal/freezer"
)

type HostProvider interface {
	Provider() (string, error)
	Freeze(host string, cause error, duration int64) error
}

func NewWithHosts(hosts []string) HostProvider {
	return &arrayHostProvider{
		hosts:   hosts,
		freezer: freezer.New(),
	}
}

type arrayHostProvider struct {
	hosts         []string
	freezer       freezer.Freezer
	lastFreezeErr error
}

func (a *arrayHostProvider) Provider() (string, error) {
	if len(a.hosts) == 0 {
		return "", errors.New("no host found")
	}

	for _, host := range a.hosts {
		if a.freezer.Available(host) {
			return host, nil
		}
	}
	return "", a.lastFreezeErr
}

func (a *arrayHostProvider) Freeze(host string, cause error, duration int64) error {
	a.lastFreezeErr = cause
	return a.freezer.Freeze(host, duration)
}
