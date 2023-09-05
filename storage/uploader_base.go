package storage

import (
	"errors"
	"time"

	"github.com/qiniu/go-sdk/v7/internal/hostprovider"
)

// retryMax: 为 0，使用默认值，每个域名只请求一次
// hostFreezeDuration: 为 0，使用默认值：50ms ~ 100ms
func getUpHost(config *Config, retryMax int, hostFreezeDuration time.Duration, ak, bucket string) (upHost string, err error) {
	region := config.GetRegion()
	if region == nil {
		if region, err = GetRegionWithOptions(ak, bucket, UCApiOptions{
			RetryMax:           retryMax,
			HostFreezeDuration: hostFreezeDuration,
		}); err != nil {
			return "", err
		}
	}

	if region == nil {
		return "", errors.New("can't get region with bucket:" + bucket)
	}

	if config.UseCdnDomains {
		if len(region.CdnUpHosts) == 0 {
			return "", errors.New("can't get region cdn host with bucket:" + bucket)
		}

		return endpoint(config.UseHTTPS, region.CdnUpHosts[0]), nil
	}

	if len(region.SrcUpHosts) == 0 {
		return "", errors.New("can't get region src host with bucket:" + bucket)
	}

	return endpoint(config.UseHTTPS, region.SrcUpHosts[0]), nil
}

// retryMax: 为 0，使用默认值，每个域名只请求一次
// hostFreezeDuration: 为 0，使用默认值：50ms ~ 100ms
func getUpHostProvider(config *Config, retryMax int, hostFreezeDuration time.Duration, ak, bucket string) (hostprovider.HostProvider, error) {
	region := config.GetRegion()
	var err error
	if region == nil {
		if region, err = GetRegionWithOptions(ak, bucket, UCApiOptions{
			RetryMax:           retryMax,
			HostFreezeDuration: hostFreezeDuration,
		}); err != nil {
			return nil, err
		}
	}

	hosts := make([]string, 0)
	if config.UseCdnDomains && len(region.CdnUpHosts) > 0 {
		hosts = append(hosts, region.CdnUpHosts...)
	} else if len(region.SrcUpHosts) > 0 {
		hosts = append(hosts, region.SrcUpHosts...)
	}

	for i := 0; i < len(hosts); i++ {
		hosts[i] = endpoint(config.UseHTTPS, hosts[i])
	}

	return hostprovider.NewWithHosts(hosts), nil
}
