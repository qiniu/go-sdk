package region

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/qiniu/go-sdk/v7/internal/freezer"
	"github.com/qiniu/go-sdk/v7/internal/hostprovider"
)

type (
	// 服务地址
	//
	// 可以存储域名或 IP，端口和协议可选
	Endpoints struct {
		Preferred   []string `json:"preferred,omitempty"`
		Alternative []string `json:"alternative,omitempty"`
	}

	// 区域信息
	//
	// 可能有多个机房信息，每个机房可能有多个服务地址
	//
	// 如果使用公有云，建议使用 GetRegionByID 方法直接获取获取区域实例，不建议手动设置服务地址
	Region struct {
		RegionID string    `json:"region_id,omitempty"` // 区域 ID
		Up       Endpoints `json:"up,omitempty"`        // Up 服务域名
		Io       Endpoints `json:"io,omitempty"`        // Io 服务域名
		IoSrc    Endpoints `json:"io_src,omitempty"`    // IoSrc 服务域名
		Rs       Endpoints `json:"rs,omitempty"`        // Rs 服务域名
		Rsf      Endpoints `json:"rsf,omitempty"`       // Rsf 服务域名
		Api      Endpoints `json:"api,omitempty"`       // Api 服务域名
		Bucket   Endpoints `json:"bucket,omitempty"`    // Bucket 服务域名
	}

	// 区域提供者
	RegionsProvider interface {
		GetRegions(context.Context) ([]*Region, error)
	}

	// 服务名称
	ServiceName string

	// 服务地址迭代器
	EndpointsIter struct {
		endpoints     Endpoints
		index         int
		isAlternative bool
	}

	// 服务地址提供者
	EndpointsProvider interface {
		GetEndpoints(context.Context) (Endpoints, error)
	}

	endpointsHostProvider struct {
		iter          *EndpointsIter
		freezer       freezer.Freezer
		lastFreezeErr error
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
func GetRegionByID(regionID string, useHttps bool) *Region {
	region := &Region{RegionID: regionID}
	if regionID == "z0" {
		region.Up.Preferred = []string{makeHost("upload.qiniup.com", useHttps), makeHost("upload-z0.qiniup.com", useHttps), makeHost("up.qiniup.com", useHttps), makeHost("up-z0.qiniup.com", useHttps)}
		region.Up.Alternative = []string{makeHost("up.qbox.me", useHttps), makeHost("up-z0.qbox.me", useHttps)}
		region.Io.Preferred = []string{makeHost("iovip.qiniuio.com", useHttps), makeHost("iovip-z0.qiniuio.com", useHttps)}
		region.Io.Alternative = []string{makeHost("iovip.qbox.me", useHttps), makeHost("iovip-z0.qbox.me", useHttps)}
	} else {
		region.Up.Preferred = []string{makeHost(fmt.Sprintf("upload-%s.qiniup.com", regionID), useHttps), makeHost(fmt.Sprintf("up-%s.qiniup.com", regionID), useHttps)}
		region.Io.Preferred = []string{makeHost(fmt.Sprintf("iovip-%s.qiniuio.com", regionID), useHttps)}
	}
	region.Rs.Preferred = []string{makeHost(fmt.Sprintf("rs-%s.qiniuapi.com", regionID), useHttps)}
	region.Rsf.Preferred = []string{makeHost(fmt.Sprintf("rsf-%s.qiniuapi.com", regionID), useHttps)}
	region.Api.Preferred = []string{makeHost(fmt.Sprintf("api-%s.qiniuapi.com", regionID), useHttps)}
	region.Bucket.Preferred = []string{makeHost("uc.qiniuapi.com", useHttps), makeHost("kodo-config.qiniuapi.com", useHttps)}
	region.Bucket.Alternative = []string{makeHost("uc.qbox.me", useHttps)}
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

func (region *Region) Endpoints(serviceNames []ServiceName) (Endpoints, error) {
	var endpoint Endpoints
	for _, serviceName := range serviceNames {
		switch serviceName {
		case ServiceUp:
			endpoint = endpoint.Join(region.Up)
		case ServiceIo:
			endpoint = endpoint.Join(region.Io)
		case ServiceIoSrc:
			endpoint = endpoint.Join(region.IoSrc)
		case ServiceRs:
			endpoint = endpoint.Join(region.Rs)
		case ServiceRsf:
			endpoint = endpoint.Join(region.Rsf)
		case ServiceApi:
			endpoint = endpoint.Join(region.Api)
		case ServiceBucket:
			endpoint = endpoint.Join(region.Bucket)
		default:
			return endpoint, ErrUnrecognizedServiceName
		}
	}
	return endpoint, nil
}

func (region *Region) EndpointsIter(serviceNames []ServiceName) (*EndpointsIter, error) {
	endpoints, err := region.Endpoints(serviceNames)
	if err != nil {
		return nil, err
	}
	return endpoints.Iter(), nil
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

func (left Endpoints) Join(rights ...Endpoints) Endpoints {
	newEndpoint := left
	for _, right := range rights {
		if len(newEndpoint.Preferred) == 0 {
			newEndpoint.Preferred = right.Preferred
		} else {
			newEndpoint.Preferred = append(newEndpoint.Preferred, right.Preferred...)
		}
		if len(newEndpoint.Alternative) == 0 {
			newEndpoint.Alternative = right.Alternative
		} else {
			newEndpoint.Alternative = append(newEndpoint.Alternative, right.Alternative...)
		}
	}

	return newEndpoint
}

func (left Endpoints) IsEqual(right Endpoints) bool {
	return reflect.DeepEqual(left.Preferred, right.Preferred) &&
		reflect.DeepEqual(left.Alternative, right.Alternative)
}

func (hosts Endpoints) Iter() *EndpointsIter {
	return &EndpointsIter{endpoints: hosts}
}

func (endpoints Endpoints) IsEmpty() bool {
	return len(endpoints.Preferred) == 0 && len(endpoints.Alternative) == 0
}

func (endpoints Endpoints) firstUrl(useHttps bool) string {
	for _, preferred := range endpoints.Preferred {
		return makeUrlFromHost(preferred, useHttps)
	}
	for _, alternative := range endpoints.Alternative {
		return makeUrlFromHost(alternative, useHttps)
	}
	return ""
}

func (endpoints Endpoints) GetEndpoints(context.Context) (Endpoints, error) {
	return endpoints, nil
}

func (endpoints Endpoints) allUrls(useHttps bool) []string {
	allHosts := make([]string, 0, len(endpoints.Preferred)+len(endpoints.Alternative))
	for _, preferred := range endpoints.Preferred {
		allHosts = append(allHosts, makeUrlFromHost(preferred, useHttps))
	}
	for _, alternative := range endpoints.Alternative {
		allHosts = append(allHosts, makeUrlFromHost(alternative, useHttps))
	}
	return allHosts
}

func (endpoints Endpoints) ToHostProvider() hostprovider.HostProvider {
	return &endpointsHostProvider{
		iter:    endpoints.Iter(),
		freezer: freezer.New(),
	}
}

func (endpoints Endpoints) Clone() Endpoints {
	return Endpoints{
		Preferred:   append([]string{}, endpoints.Preferred...),
		Alternative: append([]string{}, endpoints.Alternative...),
	}
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

func (iter *EndpointsIter) Next(nextHost *string) bool {
	if iter.isAlternative {
		if iter.index >= len(iter.endpoints.Alternative) {
			return false
		}
		host := iter.endpoints.Alternative[iter.index]
		iter.index += 1
		*nextHost = host
		return true
	}
	if iter.index >= len(iter.endpoints.Preferred) {
		iter.isAlternative = true
		iter.index = 0
		return iter.Next(nextHost)
	}
	host := iter.endpoints.Preferred[iter.index]
	iter.index += 1
	*nextHost = host
	return true
}

func (iter *EndpointsIter) More() bool {
	if iter.isAlternative {
		return iter.index < len(iter.endpoints.Alternative)
	} else if iter.index >= len(iter.endpoints.Preferred) {
		return len(iter.endpoints.Alternative) > 0
	}
	return true
}

func (iter *EndpointsIter) SwitchToAlternative() {
	if len(iter.endpoints.Alternative) > 0 && !iter.isAlternative {
		iter.isAlternative = true
		iter.index = 0
	}
}

func (provider *endpointsHostProvider) Freeze(host string, cause error, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}

	provider.lastFreezeErr = cause
	return provider.freezer.Freeze(host, duration)
}

func (provider *endpointsHostProvider) Provider() (string, error) {
	if provider.iter.endpoints.IsEmpty() {
		return "", hostprovider.ErrNoHostFound
	}

	var host string
	for provider.iter.Next(&host) {
		if provider.freezer.Available(host) {
			return host, nil
		}
	}

	if provider.lastFreezeErr != nil {
		return "", provider.lastFreezeErr
	} else {
		return "", hostprovider.ErrAllHostsFrozen
	}
}
