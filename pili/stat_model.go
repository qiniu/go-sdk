package pili

import (
	"reflect"
	"strings"
	"time"
)

const (
	// FlowDefaultSelect 上下行流量默认查询字段
	FlowDefaultSelect = "flow"

	// CodecDefaultSelect 转码使用量默认查询字段
	CodecDefaultSelect = "duration"

	// NropDefaultSelect 鉴黄使用量默认查询字段
	NropDefaultSelect = "count"
)

// GetStatCommonRequest 统计接口通用请求参数
type GetStatCommonRequest struct {

	// Begin 开始时间
	// 支持格式：20060102、20060102150405
	Begin string `validate:"required,len=8|len=14"`

	// End 结束时间
	// 支持格式：20060102、20060102150405
	// 超过当前时间，则以当前时间为准
	// 时间范围为左闭右开区间
	End string `validate:"len=8|len=14"`

	// G 时间粒度
	// 可选项 5min hour day month，部分接口仅支持部分粒度
	G string `validate:"required,oneof=5min hour day month"`
}

// GetStatUpflowRequest 获取上行流量请求参数
type GetStatUpflowRequest struct {

	// 通用请求参数
	GetStatCommonRequest

	// Where 查询条件
	// hub 直播空间
	// domain 域名
	// area 区域 中国大陆(cn)、香港(hk)、台湾(tw)、亚太(apac)、美洲(am)、欧洲/中东/非洲(emea)
	Where map[string][]string `validate:"dive,keys,oneof=hub domain area"`

	// Select 查询值
	// flow 流量，单位：byte，带宽可以从流量转换，公式为 带宽=流量*8/时间粒度，单位：bps
	Select []string `validate:"dive,eq=flow"`
}

// GroupStatUpflowRequest 分组获取上行流量请求参数
type GroupStatUpflowRequest struct {

	// 通用请求参数
	GetStatCommonRequest

	// Group 分组
	Group string `validate:"required,oneof=hub domain area"`

	// Where 查询条件
	// hub 直播空间
	// domain 域名
	// area 区域 中国大陆(cn)、香港(hk)、台湾(tw)、亚太(apac)、美洲(am)、欧洲/中东/非洲(emea)
	Where map[string][]string `validate:"dive,keys,oneof=hub domain area"`

	// Select 查询值
	// flow 流量，单位：byte，带宽可以从流量转换，公式为 带宽=流量*8/时间粒度，单位：bps
	Select []string `validate:"dive,eq=flow"`
}

// GetStatDownflowRequest 获取下行流量请求参数
type GetStatDownflowRequest struct {

	// 通用请求参数
	GetStatCommonRequest

	// Where 查询条件
	// hub 直播空间
	// domain 域名
	// area 区域 中国大陆(cn)、香港(hk)、台湾(tw)、亚太(apac)、美洲(am)、欧洲/中东/非洲(emea)
	Where map[string][]string `validate:"dive,keys,oneof=hub domain area"`

	// Select 查询值
	// flow 流量，单位：byte，带宽可以从流量转换，公式为 带宽=流量*8/时间粒度，单位：bps
	Select []string `validate:"dive,eq=flow"`
}

// GroupStatDownflowRequest 分组获取下行流量请求参数
type GroupStatDownflowRequest struct {

	// 通用请求参数
	GetStatCommonRequest

	// Group 分组
	Group string `validate:"required,oneof=hub domain area"`

	// Where 查询条件
	// hub 直播空间
	// domain 域名
	// area 区域 中国大陆(cn)、香港(hk)、台湾(tw)、亚太(apac)、美洲(am)、欧洲/中东/非洲(emea)
	Where map[string][]string `validate:"dive,keys,oneof=hub domain area"`

	// Select 查询值
	// flow 流量，单位：byte，带宽可以从流量转换，公式为 带宽=流量*8/时间粒度，单位：bps
	Select []string `validate:"dive,eq=flow"`
}

// GetStatCodecRequest 获取直播转码使用量请求参数
type GetStatCodecRequest struct {

	// 通用请求参数
	GetStatCommonRequest

	// Where 查询条件
	// hub 直播空间
	// profile 转码规格
	Where map[string][]string `validate:"dive,keys,oneof=hub profile"`

	// Select 查询值
	// duration 时长，单位：毫秒
	Select []string `validate:"dive,eq=duration"`
}

// GroupStatCodecRequest 分组获取直播转码使用量请求参数
type GroupStatCodecRequest struct {

	// 通用请求参数
	GetStatCommonRequest

	// Group 分组
	Group string `validate:"required,oneof=hub profile"`

	// Where 查询条件
	// hub 直播空间
	// profile 转码规格
	Where map[string][]string `validate:"dive,keys,oneof=hub profile"`

	// Select 查询值
	// duration 时长，单位：毫秒
	Select []string `validate:"dive,eq=duration"`
}

// GetStatNropRequest 获取直播鉴黄使用量请求参数
type GetStatNropRequest struct {

	// 通用请求参数
	GetStatCommonRequest

	// Where 查询条件
	// hub 直播空间
	// assured 鉴黄结果是否确定，true或false
	Where map[string][]string `validate:"dive,keys,oneof=hub assured"`

	// Select 查询值
	// count 鉴黄次数
	Select []string `validate:"dive,eq=count"`
}

// GroupStatNropRequest 分组获取直播鉴黄使用量请求参数
type GroupStatNropRequest struct {

	// 通用请求参数
	GetStatCommonRequest

	// Group 分组
	Group string `validate:"required,oneof=hub assured"`

	// Where 查询条件
	// hub 直播空间
	// assured 鉴黄结果是否确定，true或false
	Where map[string][]string `validate:"dive,keys,oneof=hub assured"`

	// Select 查询值
	// count 鉴黄次数
	Select []string `validate:"dive,eq=count"`
}

// GetStatCasterRequest 获取导播台使用量请求参数
type GetStatCasterRequest struct {

	// 通用请求参数
	GetStatCommonRequest

	// Where 查询条件
	// container 容器
	// resolution 分辨率
	Where map[string][]string `validate:"dive,keys,oneof=container resolution"`

	// Select 查询值
	// upflow 上行流量，单位：byte
	// downflow 下行流量，单位：byte
	// duration 使用时长，单位：秒
	Select []string `validate:"required,dive,oneof=upflow downflow duration"`
}

// GroupStatCasterRequest 分组获取导播台使用量请求参数
type GroupStatCasterRequest struct {

	// 通用请求参数
	GetStatCommonRequest

	// Group 分组
	Group string `validate:"required,oneof=container resolution"`

	// Where 查询条件
	// container 容器
	// resolution 分辨率
	Where map[string][]string `validate:"dive,keys,oneof=container resolution"`

	// Select 查询值
	// upflow 上行流量，单位：byte
	// downflow 下行流量，单位：byte
	// duration 使用时长，单位：秒
	Select []string `validate:"required,dive,oneof=upflow downflow duration"`
}

// GetStatPubRequest 获取Pub服务使用量请求参数
type GetStatPubRequest struct {

	// 通用请求参数
	GetStatCommonRequest

	// Where 查询条件
	// tp 状态
	Where map[string][]string `validate:"dive,keys,eq=tp"`

	// Select 查询值
	// count 转推次数
	// duration 转推时长，单位：毫秒
	Select []string `validate:"required,dive,oneof=count duration"`
}

// GroupStatPubRequest 分组获取Pub服务使用量请求参数
type GroupStatPubRequest struct {

	// 通用请求参数
	GetStatCommonRequest

	// Group 分组
	Group string `validate:"required,eq=tp"`

	// Where 查询条件
	// tp 状态
	Where map[string][]string `validate:"dive,keys,eq=tp"`

	// Select 查询值
	// count 转推次数
	// duration 转推时长，单位：毫秒
	Select []string `validate:"required,dive,oneof=count duration"`
}

// StatResponse 统计返回值
type StatResponse struct {

	// Time 时间
	Time time.Time `json:"time"`

	// Values 数据
	// 形式如："values":{"flow":4527162977}，其中"flow"对应select项
	Values map[string]int `json:"values"`
}

// StatGroupResponse 分组统计返回值
type StatGroupResponse struct {

	// Time 时间
	Time time.Time `json:"time"`

	// Values 数据
	// 形式如："values":{"SD":{"duration":4527162977}}，其中"SD"对应Group项，"duration"对应Select项
	Values map[string]map[string]int `json:"values"`
}

// Form 请求参数格式化
func Form(obj interface{}) map[string][]string {
	value := reflect.ValueOf(obj)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil
	}

	ret := make(map[string][]string)

	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		switch value.Field(i).Kind() {
		case reflect.Struct:
			div := Form(value.Field(i).Interface())
			for k, v := range div {
				ret[k] = v
			}
		case reflect.String:
			val := value.Field(i).Interface().(string)
			if len(val) > 0 {
				ret[strings.ToLower(typ.Field(i).Name)] = []string{val}
			}
		case reflect.Slice:
			ret[strings.ToLower(typ.Field(i).Name)] = value.Field(i).Interface().([]string)
		case reflect.Map:
			for k, v := range value.Field(i).Interface().(map[string][]string) {
				ret["$"+k] = v
			}
		}
	}

	return ret
}
