package region

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type (
	// 区域 ID
	RegionID string

	// 服务地址
	//
	// 可以存储域名或 IP，端口和协议可选
	ServiceHosts struct {
		Preferred   []string `json:"preferred,omitempty"`
		Alternative []string `json:"alternative,omitempty"`
	}

	// 区域信息
	//
	// 可能有多个机房信息，每个机房可能有多个服务地址
	Region struct {
		RegionID RegionID     `json:"region_id,omitempty"`
		Up       ServiceHosts `json:"up,omitempty"`
		Io       ServiceHosts `json:"io,omitempty"`
		IoSrc    ServiceHosts `json:"io_src,omitempty"`
		Rs       ServiceHosts `json:"rs,omitempty"`
		Rsf      ServiceHosts `json:"rsf,omitempty"`
		Api      ServiceHosts `json:"api,omitempty"`
		Bucket   ServiceHosts `json:"bucket,omitempty"`
	}

	// 服务名称
	ServiceName string

	// 服务地址迭代器
	ServiceHostsIter struct {
		hosts         ServiceHosts
		index         int
		isAlternative bool
	}
)

const (
	// Up 服务
	ServiceUp ServiceName = "up"
	// Io 服务
	ServiceIo ServiceName = "io"
	// IoSrc 服务
	ServiceIoSrc ServiceName = "io_src"
	// Rs 服务
	ServiceRs ServiceName = "rs"
	// Rsf 服务
	ServiceRsf ServiceName = "rsf"
	// Api 服务
	ServiceApi ServiceName = "api"
	// Bucket 服务
	ServiceBucket ServiceName = "bucket"
)

var (
	ErrUnrecognizedServiceName = errors.New("unrecognized service name")
)

// 根据 RegionID 获取公有云区域信息
func GetRegionByID(regionID RegionID, useHttps bool) *Region {
	region := &Region{RegionID: regionID}
	if regionID == "z0" {
		region.Up.Preferred = []string{makeHost("upload.qiniup.com", useHttps), makeHost("up.qiniup.com", useHttps)}
		region.Up.Alternative = []string{makeHost("up.qbox.me", useHttps)}
		region.Io.Preferred = []string{makeHost("iovip.qiniuio.com", useHttps)}
		region.Io.Alternative = []string{makeHost("iovip.qbox.me", useHttps)}
	} else {
		region.Up.Preferred = []string{makeHost(fmt.Sprintf("upload-%s.qiniup.com", regionID), useHttps), makeHost(fmt.Sprintf("up-%s.qiniup.com", regionID), useHttps)}
		region.Io.Preferred = []string{makeHost(fmt.Sprintf("iovip-%s.qiniuio.com", regionID), useHttps)}
	}
	region.Rs.Preferred = []string{makeHost(fmt.Sprintf("rs-%s.qiniuapi.com", regionID), useHttps)}
	region.Rsf.Preferred = []string{makeHost(fmt.Sprintf("rsf-%s.qiniuapi.com", regionID), useHttps)}
	region.Api.Preferred = []string{makeHost(fmt.Sprintf("api-%s.qiniuapi.com", regionID), useHttps)}
	region.Bucket.Preferred = []string{makeHost("uc.qbox.me", useHttps)}
	return region
}

func makeHost(domain string, useHttps bool) string {
	if useHttps {
		return "https://" + domain
	} else {
		return "http://" + domain
	}
}

func (region *Region) GetRegions(context.Context) ([]*Region, error) {
	return []*Region{region}, nil
}

func (region *Region) IterServiceHosts(serviceName ServiceName) (*ServiceHostsIter, error) {
	switch serviceName {
	case ServiceUp:
		return region.Up.Iter(), nil
	case ServiceIo:
		return region.Io.Iter(), nil
	case ServiceIoSrc:
		return region.IoSrc.Iter(), nil
	case ServiceRs:
		return region.Rs.Iter(), nil
	case ServiceRsf:
		return region.Rsf.Iter(), nil
	case ServiceApi:
		return region.Api.Iter(), nil
	case ServiceBucket:
		return region.Bucket.Iter(), nil
	default:
		return nil, ErrUnrecognizedServiceName
	}
}

func (left *Region) IsEqual(right *Region) bool {
	return left.RegionID == right.RegionID &&
		left.Up.IsEqual(right.Up) &&
		left.Io.IsEqual(right.Io) &&
		left.IoSrc.IsEqual(right.IoSrc) &&
		left.Rs.IsEqual(right.Rs) &&
		left.Rsf.IsEqual(right.Rsf) &&
		left.Api.IsEqual(right.Api) &&
		left.Bucket.IsEqual(right.Bucket)
}

func (left ServiceHosts) IsEqual(right ServiceHosts) bool {
	return reflect.DeepEqual(left.Preferred, right.Preferred) &&
		reflect.DeepEqual(left.Alternative, right.Alternative)
}

func (hosts ServiceHosts) Iter() *ServiceHostsIter {
	return &ServiceHostsIter{hosts: hosts}
}

func (hosts ServiceHosts) firstUrl(useHttps bool) string {
	for _, preferred := range hosts.Preferred {
		return makeUrlFromHost(preferred, useHttps)
	}
	for _, alternative := range hosts.Alternative {
		return makeUrlFromHost(alternative, useHttps)
	}
	return ""
}

func (hosts ServiceHosts) allUrls(useHttps bool) []string {
	allHosts := make([]string, 0, len(hosts.Preferred)+len(hosts.Alternative))
	for _, preferred := range hosts.Preferred {
		allHosts = append(allHosts, makeUrlFromHost(preferred, useHttps))
	}
	for _, alternative := range hosts.Alternative {
		allHosts = append(allHosts, makeUrlFromHost(alternative, useHttps))
	}
	return allHosts
}

func makeUrlFromHost(host string, useHttps bool) string {
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		return host
	}
	if useHttps {
		return "https://" + host
	} else {
		return "http://" + host
	}
}

func (iter *ServiceHostsIter) Next(nextHost *string) bool {
	if iter.isAlternative {
		if iter.index >= len(iter.hosts.Alternative) {
			return false
		}
		host := iter.hosts.Alternative[iter.index]
		iter.index += 1
		*nextHost = host
		return true
	}
	if iter.index >= len(iter.hosts.Preferred) {
		iter.isAlternative = true
		iter.index = 0
		return iter.Next(nextHost)
	}
	host := iter.hosts.Preferred[iter.index]
	iter.index += 1
	*nextHost = host
	return true
}

func (iter *ServiceHostsIter) More() bool {
	if iter.isAlternative {
		return iter.index < len(iter.hosts.Alternative)
	} else if iter.index >= len(iter.hosts.Preferred) {
		return len(iter.hosts.Alternative) > 0
	}
	return true
}

func (iter *ServiceHostsIter) SwitchToAlternative() {
	if len(iter.hosts.Alternative) > 0 && !iter.isAlternative {
		iter.isAlternative = true
		iter.index = 0
	}
}
