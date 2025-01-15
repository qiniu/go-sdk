package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/internal/clientv2"
	"github.com/qiniu/go-sdk/v7/storagev2/apis"
	"github.com/qiniu/go-sdk/v7/storagev2/defaults"
	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	region_v2 "github.com/qiniu/go-sdk/v7/storagev2/region"
)

// 存储所在的地区，例如华东，华南，华北
// 每个存储区域可能有多个机房信息，每个机房可能有多个上传入口
type Region struct {
	// 上传入口
	SrcUpHosts []string `json:"src_up,omitempty"`

	// 加速上传入口
	CdnUpHosts []string `json:"cdn_up,omitempty"`

	// 获取文件信息入口
	RsHost string `json:"rs,omitempty"`

	// bucket列举入口
	RsfHost string `json:"rsf,omitempty"`

	ApiHost string `json:"api,omitempty"`

	// 存储io 入口
	IovipHost string `json:"io,omitempty"`

	// 源站下载入口
	IoSrcHost string `json:"io_src,omitempty"`
}

type RegionID string

// GetDefaultReion 根据RegionID获取对应的Region信息
func GetRegionByID(regionID RegionID) (Region, bool) {
	if r, ok := regionMap[regionID]; ok {
		return r, ok
	}
	return Region{}, false
}

func (r *Region) String() string {
	str := ""
	str += fmt.Sprintf("SrcUpHosts: %v\n", r.SrcUpHosts)
	str += fmt.Sprintf("CdnUpHosts: %v\n", r.CdnUpHosts)
	str += fmt.Sprintf("IovipHost: %s\n", r.IovipHost)
	str += fmt.Sprintf("RsHost: %s\n", r.RsHost)
	str += fmt.Sprintf("RsfHost: %s\n", r.RsfHost)
	str += fmt.Sprintf("ApiHost: %s\n", r.ApiHost)
	return str
}

func endpoint(useHttps bool, host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}

	if strings.HasPrefix(host, "http://") ||
		strings.HasPrefix(host, "https://") {
		return host
	}

	scheme := "http://"
	if useHttps {
		scheme = "https://"
	}
	return fmt.Sprintf("%s%s", scheme, host)
}

// 获取rsfHost
func (r *Region) GetRsfHost(useHttps bool) string {
	return endpoint(useHttps, r.RsfHost)
}

// 获取io host
func (r *Region) GetIoHost(useHttps bool) string {
	return endpoint(useHttps, r.IovipHost)
}

// 获取RsHost
func (r *Region) GetRsHost(useHttps bool) string {
	return endpoint(useHttps, r.RsHost)
}

// 获取api host
func (r *Region) GetApiHost(useHttps bool) string {
	return endpoint(useHttps, r.ApiHost)
}

func (r *Region) GetRegions(ctx context.Context) ([]*region_v2.Region, error) {
	newRegion := &region_v2.Region{
		Up: region_v2.Endpoints{Preferred: append(r.CdnUpHosts, r.SrcUpHosts...)},
	}
	if host := r.IovipHost; host != "" {
		newRegion.Io = region_v2.Endpoints{Preferred: []string{host}}
	}
	if host := r.IoSrcHost; host != "" {
		newRegion.IoSrc = region_v2.Endpoints{Preferred: []string{host}}
	}
	if host := r.RsHost; host != "" {
		newRegion.Rs = region_v2.Endpoints{Preferred: []string{host}}
	}
	if host := r.RsfHost; host != "" {
		newRegion.Rsf = region_v2.Endpoints{Preferred: []string{host}}
	}
	if host := r.ApiHost; host != "" {
		newRegion.Api = region_v2.Endpoints{Preferred: []string{host}}
	}
	return []*region_v2.Region{newRegion}, nil
}

var (
	// regionHuadong 表示华东机房
	regionHuadong = Region{
		SrcUpHosts: []string{
			"up.qiniup.com",
		},
		CdnUpHosts: []string{
			"upload.qiniup.com",
		},
		RsHost:    "rs.qbox.me",
		RsfHost:   "rsf.qbox.me",
		ApiHost:   "api.qiniu.com",
		IovipHost: "iovip.qbox.me",
	}

	// regionHuadongZhejiang2 表示华东-浙江2
	regionHuadongZhejiang2 = Region{
		SrcUpHosts: []string{
			"up-cn-east-2.qiniup.com",
		},
		CdnUpHosts: []string{
			"upload-cn-east-2.qiniup.com",
		},
		RsHost:    "rs-cn-east-2.qiniuapi.com",
		RsfHost:   "rsf-cn-east-2.qiniuapi.com",
		ApiHost:   "api-cn-east-2.qiniuapi.com",
		IovipHost: "iovip-cn-east-2.qiniuio.com",
	}

	// regionHuabei 表示华北机房
	regionHuabei = Region{
		SrcUpHosts: []string{
			"up-z1.qiniup.com",
		},
		CdnUpHosts: []string{
			"upload-z1.qiniup.com",
		},
		RsHost:    "rs-z1.qbox.me",
		RsfHost:   "rsf-z1.qbox.me",
		ApiHost:   "api-z1.qiniuapi.com",
		IovipHost: "iovip-z1.qbox.me",
	}

	// regionHuanan 表示华南机房
	regionHuanan = Region{
		SrcUpHosts: []string{
			"up-z2.qiniup.com",
		},
		CdnUpHosts: []string{
			"upload-z2.qiniup.com",
		},
		RsHost:    "rs-z2.qbox.me",
		RsfHost:   "rsf-z2.qbox.me",
		ApiHost:   "api-z2.qiniuapi.com",
		IovipHost: "iovip-z2.qbox.me",
	}

	// regionNorthAmerica 表示北美机房
	regionNorthAmerica = Region{
		SrcUpHosts: []string{
			"up-na0.qiniup.com",
		},
		CdnUpHosts: []string{
			"upload-na0.qiniup.com",
		},
		RsHost:    "rs-na0.qbox.me",
		RsfHost:   "rsf-na0.qbox.me",
		ApiHost:   "api-na0.qiniuapi.com",
		IovipHost: "iovip-na0.qbox.me",
	}

	// regionSingapore 表示新加坡机房
	regionSingapore = Region{
		SrcUpHosts: []string{
			"up-as0.qiniup.com",
		},
		CdnUpHosts: []string{
			"upload-as0.qiniup.com",
		},
		RsHost:    "rs-as0.qbox.me",
		RsfHost:   "rsf-as0.qbox.me",
		ApiHost:   "api-as0.qiniuapi.com",
		IovipHost: "iovip-as0.qbox.me",
	}
)

const (
	// region code
	RIDHuadong          = RegionID("z0")
	RIDHuadongZheJiang2 = RegionID("cn-east-2")
	RIDHuabei           = RegionID("z1")
	RIDHuanan           = RegionID("z2")
	RIDNorthAmerica     = RegionID("na0")
	RIDSingapore        = RegionID("as0")
)

// regionMap 是RegionID到具体的Region的映射
var regionMap = map[RegionID]Region{
	RIDHuadong:          regionHuadong,
	RIDHuadongZheJiang2: regionHuadongZhejiang2,
	RIDHuanan:           regionHuanan,
	RIDHuabei:           regionHuabei,
	RIDSingapore:        regionSingapore,
	RIDNorthAmerica:     regionNorthAmerica,
}

const (
	defaultUcHost0 = "uc.qiniuapi.com"
	defaultUcHost1 = "kodo-config.qiniuapi.com"
	defaultUcHost2 = "uc.qbox.me"
)

// UcHost 为查询空间相关域名的 API 服务地址
// 设置 UcHost 时，如果不指定 scheme 默认会使用 https
// Deprecated 使用 SetUcHosts 替换
var UcHost = ""

var ucHosts = []string{defaultUcHost0, defaultUcHost1, defaultUcHost2}

func init() {
	if defaultUcHosts, err := defaults.BucketURLs(); err == nil && len(defaultUcHosts) > 0 {
		ucHosts = defaultUcHosts
	}
}

// SetUcHost
// Deprecated 使用 SetUcHosts 替换
func SetUcHost(host string, useHttps bool) {
	if len(host) == 0 {
		return
	}
	host = endpoint(useHttps, host)
	ucHosts = []string{host}
}

// SetUcHosts 配置多个 UC 域名
func SetUcHosts(hosts ...string) {
	var newHosts []string
	for _, host := range hosts {
		if len(host) > 0 {
			newHosts = append(newHosts, host)
		}
	}
	ucHosts = newHosts
}

func getUcEndpointProvider(useHttps bool, hosts []string) region_v2.EndpointsProvider {
	if len(hosts) == 0 {
		if len(UcHost) > 0 {
			hosts = append(hosts, endpoint(useHttps, UcHost))
		}

		for _, host := range ucHosts {
			if len(host) > 0 {
				hosts = append(hosts, endpoint(useHttps, host))
			}
		}
	}

	if len(hosts) > 0 {
		return region_v2.Endpoints{Preferred: hosts}
	} else {
		return nil
	}
}

func getUcEndpoint(useHttps bool, hosts []string) region_v2.Endpoints {
	provider := getUcEndpointProvider(useHttps, hosts)
	if provider == nil {
		return region_v2.Endpoints{}
	}

	if p, ok := provider.(region_v2.Endpoints); ok {
		return p
	} else {
		return region_v2.Endpoints{}
	}
}

// GetRegion 用来根据ak和bucket来获取空间相关的机房信息
// 延用 v2, v2 结构和 v4 结构不同且暂不可替代
// Deprecated 使用 GetRegionWithOptions 替换
func GetRegion(ak, bucket string) (*Region, error) {
	return GetRegionWithOptions(ak, bucket, DefaultUCApiOptions())
}

// GetRegionWithOptions 用来根据ak和bucket来获取空间相关的机房信息
func GetRegionWithOptions(ak, bucket string, options UCApiOptions) (*Region, error) {
	return getRegionByV2(ak, bucket, options)
}

// 使用 v4
//
//lint:ignore U1000 Used by test case
func getRegionGroup(ak, bucket string) (*RegionGroup, error) {
	return getRegionByV4(ak, bucket, DefaultUCApiOptions())
}

func getRegionGroupWithOptions(ak, bucket string, options UCApiOptions) (*RegionGroup, error) {
	return getRegionByV4(ak, bucket, options)
}

type RegionInfo struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

func SetRegionCachePath(newPath string) {
	setRegionV2CachePath(newPath)
	setRegionV4CachePath(newPath)
}

// GetRegionsInfo Deprecated and use GetRegionsInfoWithOptions instead
// Deprecated
func GetRegionsInfo(mac *auth.Credentials) ([]RegionInfo, error) {
	return GetRegionsInfoWithOptions(mac, DefaultUCApiOptions())
}

func GetRegionsInfoWithOptions(mac *auth.Credentials, options UCApiOptions) ([]RegionInfo, error) {
	var httpClient clientv2.Client
	if options.Client != nil {
		httpClient = options.Client.Client
	}

	if mac == nil {
		mac = auth.Default()
	}

	response, err := apis.NewStorage(&http_client.Options{
		BasicHTTPClient:    httpClient,
		HostFreezeDuration: options.HostFreezeDuration,
		HostRetryConfig: &clientv2.RetryConfig{
			RetryMax: options.RetryMax,
		},
	}).GetRegions(
		context.Background(),
		&apis.GetRegionsRequest{Credentials: mac},
		&apis.Options{OverwrittenBucketHosts: getUcEndpointProvider(options.UseHttps, options.Hosts)},
	)
	if err != nil {
		return nil, err
	}

	regions := make([]RegionInfo, 0, len(response.Regions))
	for _, region := range response.Regions {
		regions = append(regions, RegionInfo{
			ID:          region.Id,
			Description: region.Description,
		})
	}
	return regions, nil
}
