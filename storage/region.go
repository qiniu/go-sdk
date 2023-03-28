package storage

import (
	"context"
	"fmt"
	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/client"
	"strings"
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
	host = strings.TrimLeft(host, "http://")
	host = strings.TrimLeft(host, "https://")
	if host == "" {
		return ""
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

	// regionApNortheast1 表示亚太-首尔机房
	regionApNortheast1 = Region{
		SrcUpHosts: []string{
			"up-ap-northeast-1.qiniup.com",
		},
		CdnUpHosts: []string{
			"upload-ap-northeast-1.qiniup.com",
		},
		RsHost:    "rs-ap-northeast-1.qiniuapi.com",
		RsfHost:   "rsf-ap-northeast-1.qiniuapi.com",
		ApiHost:   "api-ap-northeast-1.qiniuapi.com",
		IovipHost: "iovip-ap-northeast-1.qiniuio.com",
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
	RIDApNortheast1     = RegionID("ap-northeast-1")
)

// regionMap 是RegionID到具体的Region的映射
var regionMap = map[RegionID]Region{
	RIDHuadong:          regionHuadong,
	RIDHuadongZheJiang2: regionHuadongZhejiang2,
	RIDHuanan:           regionHuanan,
	RIDHuabei:           regionHuabei,
	RIDSingapore:        regionSingapore,
	RIDNorthAmerica:     regionNorthAmerica,
	RIDApNortheast1:     regionApNortheast1,
}

/// UcHost 为查询空间相关域名的API服务地址
/// 设置 UcHost 时，如果不指定 scheme 默认会使用 https
/// UcHost 已废弃，建议使用 SetUcHost
//Deprecated
var UcHost = "https://uc.qbox.me"

var ucHost = ""

func SetUcHost(host string, useHttps bool) {
	ucHost = endpoint(useHttps, host)
}

func getUcHostByDefaultProtocol() string {
	return getUcHost(true)
}

func getUcHost(useHttps bool) string {
	if ucHost != "" {
		return ucHost
	}

	if strings.Contains(UcHost, "://") {
		return UcHost
	} else {
		return endpoint(useHttps, UcHost)
	}
}

// GetRegion 用来根据ak和bucket来获取空间相关的机房信息
// 延用 v2, v2 结构和 v4 结构不同且暂不可替代
func GetRegion(ak, bucket string) (*Region, error) {
	return getRegionByV2(ak, bucket)
}

// 使用 v4
func getRegionGroup(ak, bucket string) (*RegionGroup, error) {
	return getRegionByV4(ak, bucket)
}

type RegionInfo struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

func SetRegionCachePath(newPath string) {
	setRegionV2CachePath(newPath)
	setRegionV4CachePath(newPath)
}

func GetRegionsInfo(mac *auth.Credentials) ([]RegionInfo, error) {
	var regions struct {
		Regions []RegionInfo `json:"regions"`
	}
	qErr := client.DefaultClient.CredentialedCallWithForm(context.Background(), mac, auth.TokenQiniu, &regions, "GET", getUcHostByDefaultProtocol()+"/regions", nil, nil)
	if qErr != nil {
		return nil, fmt.Errorf("query region error, %s", qErr.Error())
	} else {
		return regions.Regions, nil
	}
}
