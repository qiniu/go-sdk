package storage

import "github.com/qiniu/go-sdk/v7/internal/hostprovider"

func getUpHost(config *Config, ak, bucket string) (upHost string, err error) {
	var zone *Zone
	if config.Zone != nil {
		zone = config.Zone
	} else if zone, err = GetZone(ak, bucket); err != nil {
		return
	}

	host := zone.SrcUpHosts[0]
	if config.UseCdnDomains {
		host = zone.CdnUpHosts[0]
	}

	upHost = endpoint(config.UseHTTPS, host)
	return
}

func getUpHostProvider(config *Config, ak, bucket string) (hostprovider.HostProvider, error) {
	var zone *Zone
	var err error
	if config.Zone != nil {
		zone = config.Zone
	} else if zone, err = GetZone(ak, bucket); err != nil {
		return nil, err
	}

	hosts := make([]string, 0, 0)
	if config.UseCdnDomains && len(zone.CdnUpHosts) > 0 {
		hosts = append(hosts, zone.CdnUpHosts...)
	} else if len(zone.SrcUpHosts) > 0 {
		hosts = append(hosts, zone.SrcUpHosts...)
	}

	for i := 0; i < len(hosts); i++ {
		hosts[i] = endpoint(config.UseHTTPS, hosts[i])
	}

	return hostprovider.NewWithHosts(hosts), nil
}
