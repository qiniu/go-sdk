// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 设置空间的跨域规则
package get_bucket_cors_rules

import (
	"encoding/json"
	credentials "github.com/qiniu/go-sdk/v7/storagev2/credentials"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
)

// 调用 API 所用的请求
type Request struct {
	Bucket      string                          // 指定空间名称
	Credentials credentials.CredentialsProvider // 鉴权参数，用于生成鉴权凭证，如果为空，则使用 HTTPClientOptions 中的 CredentialsProvider
}

// 获取 API 所用的响应
type Response struct {
	CorsRules CorsRules // 跨域规则列表
}

// 允许的域名列表
type AllowedOriginHosts = []string

// 允许的方法列表
type AllowedMethods = []string

// 允许的 Header 列表
type AllowedHeaders = []string

// 暴露的 Header 列表
type ExposedHeaders = []string

// 跨域规则
type CorsRule struct {
	AllowedOrigin AllowedOriginHosts // 允许的域名。必填；支持通配符 * ；*表示全部匹配；只有第一个 * 生效；需要设置 "Scheme"；大小写敏感
	AllowedMethod AllowedMethods     // 允许的方法。必填；不支持通配符；大小写不敏感；
	AllowedHeader AllowedHeaders
	ExposedHeader ExposedHeaders // 选填；不支持通配符；X-Log, X-Reqid 是默认会暴露的两个 header；其他的 header 如果没有设置，则不会暴露；大小写不敏感；
	MaxAge        int64          // 结果可以缓存的时间。选填；空则不缓存
}
type jsonCorsRule struct {
	AllowedOrigin AllowedOriginHosts `json:"allowed_origin"` // 允许的域名。必填；支持通配符 * ；*表示全部匹配；只有第一个 * 生效；需要设置 "Scheme"；大小写敏感
	AllowedMethod AllowedMethods     `json:"allowed_method"` // 允许的方法。必填；不支持通配符；大小写不敏感；
	AllowedHeader AllowedHeaders     `json:"allowed_header,omitempty"`
	ExposedHeader ExposedHeaders     `json:"exposed_header,omitempty"` // 选填；不支持通配符；X-Log, X-Reqid 是默认会暴露的两个 header；其他的 header 如果没有设置，则不会暴露；大小写不敏感；
	MaxAge        int64              `json:"max_age,omitempty"`        // 结果可以缓存的时间。选填；空则不缓存
}

func (j *CorsRule) MarshalJSON() ([]byte, error) {
	if err := j.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(&jsonCorsRule{AllowedOrigin: j.AllowedOrigin, AllowedMethod: j.AllowedMethod, AllowedHeader: j.AllowedHeader, ExposedHeader: j.ExposedHeader, MaxAge: j.MaxAge})
}
func (j *CorsRule) UnmarshalJSON(data []byte) error {
	var nj jsonCorsRule
	if err := json.Unmarshal(data, &nj); err != nil {
		return err
	}
	j.AllowedOrigin = nj.AllowedOrigin
	j.AllowedMethod = nj.AllowedMethod
	j.AllowedHeader = nj.AllowedHeader
	j.ExposedHeader = nj.ExposedHeader
	j.MaxAge = nj.MaxAge
	return nil
}
func (j *CorsRule) validate() error {
	if len(j.AllowedOrigin) == 0 {
		return errors.MissingRequiredFieldError{Name: "AllowedOrigin"}
	}
	if len(j.AllowedMethod) == 0 {
		return errors.MissingRequiredFieldError{Name: "AllowedMethod"}
	}
	return nil
}

// 跨域规则列表
type CorsRules []CorsRule

func (j *Response) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.CorsRules)
}
func (j *Response) UnmarshalJSON(data []byte) error {
	var array CorsRules
	if err := json.Unmarshal(data, &array); err != nil {
		return err
	}
	j.CorsRules = array
	return nil
}
