package pili

// GetDomainsListRequest 查询域名列表请求参数
type GetDomainsListRequest struct {

	// Hub 直播空间
	Hub string `json:"-" validate:"required"`
}

// GetDomainsListResponse 查询域名列表返回值
type GetDomainsListResponse struct {

	// Domains 域名列表
	Domains []GetDomainsListItem `json:"domains"`
}

// GetDomainsListItem 查询域名列表项
type GetDomainsListItem struct {

	// Type 域名类型
	Type string `json:"type"`

	// Domain 域名
	Domain string `json:"domain"`

	// Cname CNAME
	Cname string `json:"cname"`

	// CertEnable 是否配置 SSL 证书
	CertEnable bool `json:"certEnable"`

	// CertName 证书名称
	CertName string `json:"certName"`
}

// GetDomainInfoRequest 查询域名信息请求参数
type GetDomainInfoRequest struct {

	// Hub 直播空间
	Hub string `json:"-" validate:"required"`

	// Domain 域名
	Domain string `json:"-" validate:"required"`
}

// GetDomainInfoResponse 查询域名信息返回值
type GetDomainInfoResponse struct {

	// Domain 域名
	Domain string `json:"domain"`

	// Type 域名类型
	Type string `json:"type"`

	// Cname CNAME
	Cname string `json:"cname"`

	// ConnectCallback 开播回调配置
	ConnectCallback DomainCallbackConfig `json:"connectCallback"`

	// DisconnectCallback 断播回调配置
	DisconnectCallback DomainCallbackConfig `json:"disconnectCallback"`

	// IPLimit IP 访问限制
	IPLimit DomainIPLimit `json:"ipLimit"`

	// PlaySecurity 时间戳防盗链配置
	PlaySecurity DomainPlaySecurity `json:"playSecurity"`

	// DisconnectDelay 断流延迟配置
	// 单位：秒，针对直播流短时间内闪断重连的情况，不触发断流回调，避免因为短时间内频繁断流造成大量回调
	DisconnectDelay int64 `json:"disconnectDelay"`

	// UrlRewrite URL 改写规则
	UrlRewrite DomainUrlRewrite `json:"urlRewrite"`

	// CertEnable 是否配置 SSL 证书
	CertEnable bool `json:"certEnable"`

	// CertName 证书名称
	CertName string `json:"certName"`

	// Disable 域名是否为禁用状态
	Disable bool `json:"disable"`
}

// DomainCallbackConfig 域名回调配置
type DomainCallbackConfig struct {

	// Type 回调类型
	// 可选回调类型为
	// - 留空: 不开启回调功能
	// - GET: 发送GET请求回调，请求参数携带在query中
	// - FORM: 发送POST请求回调，请求参数携带在body中，Content-Type 为 application/x-www-form-urlencoded
	// - JSON: 发送POST请求回调，请求参数携带在body中，Content-Type 为 application/json
	Type string `json:"type"`

	// URL 回调地址
	// 支持魔法变量
	URL string `json:"url"`

	// Timeout 超时时间
	// 与回调地址的 HTTP 连接超时时间，单位：秒
	// 默认值为 2 秒，配置范围为 0~10 秒
	Timeout int64 `json:"timeout"`

	// Vars 请求参数
	// 支持魔法变量，至少需要一组请求参数，规则解析出错误的会设置成空字段
	Vars map[string]string `json:"vars"`

	// RetryTimes 重试次数
	// 可选范围 0~5 次
	RetryTimes int `json:"retryTimes"`

	// RetryInterval 重试间隔
	// 可选范围 0~5 秒，单位：秒
	RetryInterval int `json:"retryInterval"`

	// SuccessCode 回调成功的 http code
	// 为 0 表示通配
	SuccessCode int `json:"successCode"`

	// FailCode 回调失败的 http code
	// 为 0 表示通配，当 SuccessCode 不为 0 的情况下生效
	FailCode int `json:"failCode"`
}

// DomainIPLimit IP 限制
type DomainIPLimit struct {

	// WhiteList 白名单
	// 允许推拉流的 IP 列表，CIDR 类型
	// 配置白名单后，黑名单列表将失效
	WhiteList []string `json:"whitelist"`

	// BlackList 黑名单
	// 限制推拉流的 IP 列表，CIDR 类型
	BlackList []string `json:"blacklist"`
}

// DomainUrlRewrite URL 改写规则配置
type DomainUrlRewrite struct {

	// Rules 规则列表
	Rules []DomainUrlRewriteItem `json:"rules"`
}

// DomainUrlRewriteItem URL 改写规则项
type DomainUrlRewriteItem struct {

	// Pattern 匹配规则
	Pattern string `json:"pattern"`

	// Replace 改写规则
	Replace string `json:"replace"`
}

// DomainPlaySecurity 时间戳防盗链配置
type DomainPlaySecurity struct {

	// Type 防盗链类型
	// 可选防盗链类型为
	// - 留空: 默认类型，表示继承直播空间级别配置
	// - none: 表示关闭鉴权
	// - tsStartMD5: 有效时间从 TsPart 表示的时间戳开始，到 Range 秒后截止
	// - tsExpireMD5: 有效时间从现在当前到 TsPart 表示的时间戳为止
	Type string `json:"type"`

	// Key1 主密钥
	Key1 string `json:"key1"`

	// Key2 副密钥
	// 两个密钥将同时生效，便于线上业务替换密钥
	Key2 string `json:"key2"`

	// Range 有效时间
	// 当 Type 为 "tsStartMD5" 时生效，单位：秒
	Range int `json:"range"`

	// Rule 签名规则
	// 支持魔法变量的规则，最终签算结果为所有变量的md5
	// - $(key): 密钥
	// - $(path): URL 中的 path 部分
	// - $(streamKey): URL 中的 hub/stream 部分
	// - $(streamTitle): URL 中的 stream 部分
	// - $(path_<number>): URL 中的 path 部分，<number> 表示 path 层级
	// - $(_<query>): URL 中的 query 字段，举例: key1=val，魔法变量中使用 $(_key1) 表示 val
	// 举例:
	// 配置Rule为: $(key)$(path)$(_t)
	// 魔法变量替换完成后: key/hub/streamTitle1634745600
	// 对结果进行md5计算，最终签算值为:3bc26fe6b35f5c7efab87778c5b27993
	Rule string `json:"rule"`

	// Rule2 签名规则 2
	// 两个签名规则将同时生效，便于线上业务更换签名规则
	Rule2 string `json:"rule2"`

	// TsPart 时间戳字段
	// URL中表示时间戳的字段名
	TsPart string `json:"tsPart"`

	// TsBase 时间戳进制
	// 可选进制格式为 2-36，即 2 进制到 36 进制，默认使用16进制
	TsBase int `json:"tsBase"`

	// SignPart 签名字段
	// URL中表示token的字段名
	SignPart string `json:"signPart"`

	// GapDuration 时间误差值
	// 针对 tsExpireMD5 生效，避免因签算方与服务器本地时间误差造成的鉴权失败
	GapDuration int `json:"gapDuration"`
}

// 域名类型
const (
	// DomainPublishRTMP 推流域名
	DomainPublishRTMP = "publishRtmp"

	// DomainLiveRTMP RTMP 播放域名
	DomainLiveRTMP = "liveRtmp"

	// DomainLiveHLS HLS 播放域名
	DomainLiveHLS = "liveHls"

	// DomainLiveHDL HDL 播放域名
	DomainLiveHDL = "liveHdl"
)

// BindDomainRequest 绑定域名请求参数
type BindDomainRequest struct {

	// Hub 直播空间
	Hub string `json:"-" validate:"required"`

	// Domain 域名
	Domain string `json:"domain" validate:"required"`

	// Type 域名类型
	// 可选域名类型为
	// - publishRtmp: RTMP 推流域名
	// - liveRtmp: RTMP 播放域名
	// - liveHls: HLS 播放域名
	// - liveHdl: HDL 播放域名
	Type string `validate:"required,oneof=publishRtmp liveRtmp liveHls liveHdl"`
}

// UnbindDomainRequest 解绑域名请求参数
type UnbindDomainRequest struct {

	// Hub 直播空间
	Hub string `json:"-" validate:"required"`

	// Domain 域名
	Domain string `json:"-" validate:"required"`
}

// BindVodDomainRequest 绑定点播域名请求参数
type BindVodDomainRequest struct {

	// Hub 直播空间
	Hub string `json:"-" validate:"required"`

	// VodDomain 点播域名
	VodDomain string `json:"vodDomain" validate:"required"`
}

// SetDomainCertRequest 修改域名证书配置请求参数
type SetDomainCertRequest struct {

	// Hub 直播空间
	Hub string `json:"-" validate:"required"`

	// Domain 域名
	Domain string `json:"-" validate:"required"`

	// CertName 证书名称
	// 需要将证书配置到七牛 CDN 服务中
	// 证书配置地址: https://portal.qiniu.com/certificate/ssl
	CertName string `json:"certName" validate:"required"`
}

// SetDomainURLRewriteRequest 修改域名改写规则配置请求参数
type SetDomainURLRewriteRequest struct {

	// Hub 直播空间
	Hub string `json:"-" validate:"required"`

	// Domain 域名
	Domain string `json:"-" validate:"required"`

	// Rules URL 改写规则列表
	Rules []DomainURLRewriteRule `json:"rules" validate:"-"`
}

// DomainURLRewriteRule URL 改写规则
type DomainURLRewriteRule struct {

	// Pattern 匹配规则
	// 针对完整URL的正则表达式，形式如：(.+)/live/(.+)/playlist.m3u8
	// 括号中的内容允许在 Replace 中使用${n}引用（n表示括号顺序）
	Pattern string `json:"pattern" validate:"required"`

	// Replace 改写规则
	// 希望得到的改写结果，形式如：${1}/hub/${2}.m3u8
	// 改写后的URL应符合七牛的直播URL规范: <scheme>://<domain>/<hub>/<stream>[<ext>]?<query>
	Replace string `json:"replace" validate:"required"`
}
