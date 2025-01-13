// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 查询存储空间服务域名
package query_bucket_v4

import (
	"encoding/json"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
)

// 调用 API 所用的请求
type Request struct {
	Bucket    string // 存储空间名称
	AccessKey string // Access Key
}

// 获取 API 所用的响应
type Response struct {
	Hosts BucketQueryHosts // 存储空间服务域名
}

// 加速上传域名列表
type AcceleratedUpDomains = []string

// 主上传域名列表
type PreferedUpDomains = []string

// 备选上传域名列表
type AlternativeUpDomains = []string

// 上传域名
type UpDomains struct {
	AcceleratedUpDomains AcceleratedUpDomains // 加速上传域名列表
	PreferedUpDomains    PreferedUpDomains    // 主上传域名列表
	AlternativeUpDomains AlternativeUpDomains // 备选上传域名列表
}
type jsonUpDomains struct {
	AcceleratedUpDomains AcceleratedUpDomains `json:"acc_domains"` // 加速上传域名列表
	PreferedUpDomains    PreferedUpDomains    `json:"domains"`     // 主上传域名列表
	AlternativeUpDomains AlternativeUpDomains `json:"old"`         // 备选上传域名列表
}

func (j *UpDomains) MarshalJSON() ([]byte, error) {
	if err := j.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(&jsonUpDomains{AcceleratedUpDomains: j.AcceleratedUpDomains, PreferedUpDomains: j.PreferedUpDomains, AlternativeUpDomains: j.AlternativeUpDomains})
}
func (j *UpDomains) UnmarshalJSON(data []byte) error {
	var nj jsonUpDomains
	if err := json.Unmarshal(data, &nj); err != nil {
		return err
	}
	j.AcceleratedUpDomains = nj.AcceleratedUpDomains
	j.PreferedUpDomains = nj.PreferedUpDomains
	j.AlternativeUpDomains = nj.AlternativeUpDomains
	return nil
}
func (j *UpDomains) validate() error {
	if len(j.AcceleratedUpDomains) == 0 {
		return errors.MissingRequiredFieldError{Name: "AcceleratedUpDomains"}
	}
	if len(j.PreferedUpDomains) == 0 {
		return errors.MissingRequiredFieldError{Name: "PreferedUpDomains"}
	}
	if len(j.AlternativeUpDomains) == 0 {
		return errors.MissingRequiredFieldError{Name: "AlternativeUpDomains"}
	}
	return nil
}

// 主下载域名列表
type PreferedIoDomains = []string

// 下载域名
type IoDomains struct {
	PreferedIoDomains PreferedIoDomains // 主下载域名列表
}
type jsonIoDomains struct {
	PreferedIoDomains PreferedIoDomains `json:"domains"` // 主下载域名列表
}

func (j *IoDomains) MarshalJSON() ([]byte, error) {
	if err := j.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(&jsonIoDomains{PreferedIoDomains: j.PreferedIoDomains})
}
func (j *IoDomains) UnmarshalJSON(data []byte) error {
	var nj jsonIoDomains
	if err := json.Unmarshal(data, &nj); err != nil {
		return err
	}
	j.PreferedIoDomains = nj.PreferedIoDomains
	return nil
}
func (j *IoDomains) validate() error {
	if len(j.PreferedIoDomains) == 0 {
		return errors.MissingRequiredFieldError{Name: "PreferedIoDomains"}
	}
	return nil
}

// 主源站下载域名列表
type PreferedIoSrcDomains = []string

// 源站下载域名
type IoSrcDomains struct {
	PreferedIoSrcDomains PreferedIoSrcDomains // 主源站下载域名列表
}
type jsonIoSrcDomains struct {
	PreferedIoSrcDomains PreferedIoSrcDomains `json:"domains"` // 主源站下载域名列表
}

func (j *IoSrcDomains) MarshalJSON() ([]byte, error) {
	if err := j.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(&jsonIoSrcDomains{PreferedIoSrcDomains: j.PreferedIoSrcDomains})
}
func (j *IoSrcDomains) UnmarshalJSON(data []byte) error {
	var nj jsonIoSrcDomains
	if err := json.Unmarshal(data, &nj); err != nil {
		return err
	}
	j.PreferedIoSrcDomains = nj.PreferedIoSrcDomains
	return nil
}
func (j *IoSrcDomains) validate() error {
	if len(j.PreferedIoSrcDomains) == 0 {
		return errors.MissingRequiredFieldError{Name: "PreferedIoSrcDomains"}
	}
	return nil
}

// 主对象管理域名列表
type PreferedRsDomains = []string

// 对象管理域名
type RsDomains struct {
	PreferedRsDomains PreferedRsDomains // 主对象管理域名列表
}
type jsonRsDomains struct {
	PreferedRsDomains PreferedRsDomains `json:"domains"` // 主对象管理域名列表
}

func (j *RsDomains) MarshalJSON() ([]byte, error) {
	if err := j.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(&jsonRsDomains{PreferedRsDomains: j.PreferedRsDomains})
}
func (j *RsDomains) UnmarshalJSON(data []byte) error {
	var nj jsonRsDomains
	if err := json.Unmarshal(data, &nj); err != nil {
		return err
	}
	j.PreferedRsDomains = nj.PreferedRsDomains
	return nil
}
func (j *RsDomains) validate() error {
	if len(j.PreferedRsDomains) == 0 {
		return errors.MissingRequiredFieldError{Name: "PreferedRsDomains"}
	}
	return nil
}

// 主对象列举域名列表
type PreferedRsfDomains = []string

// 对象列举域名
type RsfDomains struct {
	PreferedRsfDomains PreferedRsfDomains // 主对象列举域名列表
}
type jsonRsfDomains struct {
	PreferedRsfDomains PreferedRsfDomains `json:"domains"` // 主对象列举域名列表
}

func (j *RsfDomains) MarshalJSON() ([]byte, error) {
	if err := j.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(&jsonRsfDomains{PreferedRsfDomains: j.PreferedRsfDomains})
}
func (j *RsfDomains) UnmarshalJSON(data []byte) error {
	var nj jsonRsfDomains
	if err := json.Unmarshal(data, &nj); err != nil {
		return err
	}
	j.PreferedRsfDomains = nj.PreferedRsfDomains
	return nil
}
func (j *RsfDomains) validate() error {
	if len(j.PreferedRsfDomains) == 0 {
		return errors.MissingRequiredFieldError{Name: "PreferedRsfDomains"}
	}
	return nil
}

// 主 API 域名列表
type PreferedApiDomains = []string

// API 域名
type ApiDomains struct {
	PreferedApiDomains PreferedApiDomains // 主 API 域名列表
}
type jsonApiDomains struct {
	PreferedApiDomains PreferedApiDomains `json:"domains"` // 主 API 域名列表
}

func (j *ApiDomains) MarshalJSON() ([]byte, error) {
	if err := j.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(&jsonApiDomains{PreferedApiDomains: j.PreferedApiDomains})
}
func (j *ApiDomains) UnmarshalJSON(data []byte) error {
	var nj jsonApiDomains
	if err := json.Unmarshal(data, &nj); err != nil {
		return err
	}
	j.PreferedApiDomains = nj.PreferedApiDomains
	return nil
}
func (j *ApiDomains) validate() error {
	if len(j.PreferedApiDomains) == 0 {
		return errors.MissingRequiredFieldError{Name: "PreferedApiDomains"}
	}
	return nil
}

// 主 API 域名列表
type PreferedS3Domains = []string

// S3 域名
type S3Domains struct {
	RegionAlias        string            // S3 Region
	PreferedApiDomains PreferedS3Domains // 主 S3 域名列表
}
type jsonS3Domains struct {
	RegionAlias        string            `json:"region_alias"` // S3 Region
	PreferedApiDomains PreferedS3Domains `json:"domains"`      // 主 S3 域名列表
}

func (j *S3Domains) MarshalJSON() ([]byte, error) {
	if err := j.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(&jsonS3Domains{RegionAlias: j.RegionAlias, PreferedApiDomains: j.PreferedApiDomains})
}
func (j *S3Domains) UnmarshalJSON(data []byte) error {
	var nj jsonS3Domains
	if err := json.Unmarshal(data, &nj); err != nil {
		return err
	}
	j.RegionAlias = nj.RegionAlias
	j.PreferedApiDomains = nj.PreferedApiDomains
	return nil
}
func (j *S3Domains) validate() error {
	if j.RegionAlias == "" {
		return errors.MissingRequiredFieldError{Name: "RegionAlias"}
	}
	if len(j.PreferedApiDomains) == 0 {
		return errors.MissingRequiredFieldError{Name: "PreferedApiDomains"}
	}
	return nil
}

// 存储空间服务域名
type BucketQueryHost struct {
	RegionId     string       // 区域 ID
	TimeToLive   int64        // 查询结果的 TTL
	UpDomains    UpDomains    // 上传域名
	IoDomains    IoDomains    // 下载域名
	IoSrcDomains IoSrcDomains // 源站下载域名
	RsDomains    RsDomains    // 对象管理域名
	RsfDomains   RsfDomains   // 对象列举域名
	ApiDomains   ApiDomains   // API 域名
	S3Domains    S3Domains    // S3 域名
}
type jsonBucketQueryHost struct {
	RegionId     string       `json:"region"`           // 区域 ID
	TimeToLive   int64        `json:"ttl"`              // 查询结果的 TTL
	UpDomains    UpDomains    `json:"up"`               // 上传域名
	IoDomains    IoDomains    `json:"io"`               // 下载域名
	IoSrcDomains IoSrcDomains `json:"io_src,omitempty"` // 源站下载域名
	RsDomains    RsDomains    `json:"rs"`               // 对象管理域名
	RsfDomains   RsfDomains   `json:"rsf"`              // 对象列举域名
	ApiDomains   ApiDomains   `json:"api"`              // API 域名
	S3Domains    S3Domains    `json:"s3"`               // S3 域名
}

func (j *BucketQueryHost) MarshalJSON() ([]byte, error) {
	if err := j.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(&jsonBucketQueryHost{RegionId: j.RegionId, TimeToLive: j.TimeToLive, UpDomains: j.UpDomains, IoDomains: j.IoDomains, IoSrcDomains: j.IoSrcDomains, RsDomains: j.RsDomains, RsfDomains: j.RsfDomains, ApiDomains: j.ApiDomains, S3Domains: j.S3Domains})
}
func (j *BucketQueryHost) UnmarshalJSON(data []byte) error {
	var nj jsonBucketQueryHost
	if err := json.Unmarshal(data, &nj); err != nil {
		return err
	}
	j.RegionId = nj.RegionId
	j.TimeToLive = nj.TimeToLive
	j.UpDomains = nj.UpDomains
	j.IoDomains = nj.IoDomains
	j.IoSrcDomains = nj.IoSrcDomains
	j.RsDomains = nj.RsDomains
	j.RsfDomains = nj.RsfDomains
	j.ApiDomains = nj.ApiDomains
	j.S3Domains = nj.S3Domains
	return nil
}
func (j *BucketQueryHost) validate() error {
	if j.RegionId == "" {
		return errors.MissingRequiredFieldError{Name: "RegionId"}
	}
	if j.TimeToLive == 0 {
		return errors.MissingRequiredFieldError{Name: "TimeToLive"}
	}
	if err := j.UpDomains.validate(); err != nil {
		return err
	}
	if err := j.IoDomains.validate(); err != nil {
		return err
	}
	if err := j.RsDomains.validate(); err != nil {
		return err
	}
	if err := j.RsfDomains.validate(); err != nil {
		return err
	}
	if err := j.ApiDomains.validate(); err != nil {
		return err
	}
	if err := j.S3Domains.validate(); err != nil {
		return err
	}
	return nil
}

// 存储空间服务域名
type BucketQueryHosts = []BucketQueryHost

// 存储空间服务域名查询结果
type BucketQueryResult = Response
type jsonResponse struct {
	Hosts BucketQueryHosts `json:"hosts"` // 存储空间服务域名
}

func (j *Response) MarshalJSON() ([]byte, error) {
	if err := j.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(&jsonResponse{Hosts: j.Hosts})
}
func (j *Response) UnmarshalJSON(data []byte) error {
	var nj jsonResponse
	if err := json.Unmarshal(data, &nj); err != nil {
		return err
	}
	j.Hosts = nj.Hosts
	return nil
}
func (j *Response) validate() error {
	if len(j.Hosts) == 0 {
		return errors.MissingRequiredFieldError{Name: "Hosts"}
	}
	for _, value := range j.Hosts {
		if err := value.validate(); err != nil {
			return err
		}
	}
	return nil
}
