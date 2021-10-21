package pili

// GetHubListResponse 查询直播空间列表返回值
type GetHubListResponse struct {

	// Items 直播空间列表
	Items []GetHubListItem `json:"items"`
}

// GetHubListItem 查询直播空间列表项
type GetHubListItem struct {

	// Name 直播空间
	Name string `json:"name"`
}

// GetHubInfoRequest 查询直播空间信息请求参数
type GetHubInfoRequest struct {

	// Hub 直播空间
	Hub string `json:"-" validate:"required"`
}

// GetHubInfoResponse 查询直播空间信息返回值
type GetHubInfoResponse struct {

	// Name 直播空间名称
	Name string `json:"hubName"`

	// CreatedAt 创建时间
	// Unix 时间戳
	CreatedAt int64 `json:"createdAt"`

	// UpdatedAt 更新时间
	// Unix 时间戳
	UpdatedAt int64 `json:"updatedAt"`

	// Domains 直播空间下的域名列表
	Domains []HubDomain `json:"domains"`

	// DefaultDomains 直播空间默认域名
	DefaultDomains []DefaultDomain `json:"defaultDomains"`

	// StorageBucket 存储 bucket
	StorageBucket string `json:"storageBucket"`

	// LiveDataExpireDays 存储过期时间
	// 单位：天
	LiveDataExpireDays int64 `json:"liveDataExpireDays"`

	// PublishSecurity 推流鉴权方式
	PublishSecurity string `json:"publishSecurity"`

	// Nrop 鉴黄配置信息
	Nrop NropArgs `json:"nrop"`

	// PassiveCodecProfiles 被动转码配置
	// 形式如：720p
	PassiveCodecProfiles []string `json:"passiveCodecProfiles,omitempty"`

	// Converts 主动转码配置
	// 形式如：720p
	Converts []string `json:"converts"`

	// HlsPlus 是否开启 hls 低延迟
	HlsPlus bool `json:"hlsPlus"`

	// VodDomain 点播域名
	VodDomain string `json:"vodDomain"`

	// AccessLog 直播日志保存信息
	AccessLog AccessLogOptions `json:"accesslog"`

	// SnapshotInterval 直播封面的截图间隔
	// 单位：秒
	SnapshotInterval int `json:"snapshotInterval"`
}

// HubDomain 直播空间下的域名简要信息
type HubDomain struct {

	// Type 域名类型
	Type string `json:"type"`

	// Domain 域名
	Domain string `json:"domain"`

	// Cname CNAME
	Cname string `json:"cname"`
}

// DefaultDomain 默认域名
type DefaultDomain struct {

	// Type 域名类型
	Type string `json:"type"`

	// Domain 域名
	Domain string `json:"domain"`
}

// NropArgs 鉴黄配置
type NropArgs struct {

	// Enable 是否开启直播空间级别鉴黄功能
	Enable bool `json:"enable"`

	// Interval 截帧间隔
	// 每个流隔多久进行截帧并鉴黄，单位：秒
	Interval int `json:"interval"`

	// NotifyURL 回调 URL
	NotifyURL string `json:"notifyURL"`

	// NotifyRate 通知阈值
	// 鉴黄结果阈值表示AI模型判断当前直播画面有多大的概率涉黄
	// 当鉴黄结果阈值大于或等于通知阈值时，将发送回调信息到回调URL
	NotifyRate float64 `json:"notifyRate"`
}

// AccessLogOptions 配置
type AccessLogOptions struct {

	// SaveBucket 存储空间
	SaveBucket string `json:"saveBucket"`

	// ExpireDays 过期天数
	ExpireDays int `json:"expireDays"`
}

// HubSecurityRequest 修改直播空间推流鉴权配置请求参数
type HubSecurityRequest struct {

	// Hub 直播空间
	Hub string `json:"-" validate:"required"`

	// PublishSecurity 鉴权方式
	// 可选推流鉴权类型为
	// expiry: 限时鉴权
	// expiry_sk: 限时鉴权SK
	PublishSecurity string `json:"publishSecurity" validate:"required"`

	// PublishKey 密钥
	PublishKey string `json:"publishKey" validate:"required"`
}

// HubHlsplusRequest 修改直播空间 hls 低延迟配置请求参数
type HubHlsplusRequest struct {

	// Hub 直播空间
	Hub string `json:"-" validate:"required"`

	// HlsPlus 是否开启 hls 低延迟
	HlsPlus bool `json:"hlsPlus" validate:"-"`
}

// HubPersistenceRequest 修改直播空间存储配置请求参数
type HubPersistenceRequest struct {

	// Hub 直播空间
	Hub string `json:"-" validate:"required"`

	// StorageBucket 存储空间
	StorageBucket string `json:"storageBucket" validate:"required"`

	// LiveDataExpireDays 存储过期时间
	// 单位：天
	LiveDataExpireDays uint `json:"liveDataExpireDays" validate:"required"`
}

// HubSnapshotRequest 修改直播空间封面配置请求参数
type HubSnapshotRequest struct {

	// Hub 直播空间
	Hub string `json:"-" validate:"required"`

	// SnapshotInterval 间隔时间
	// 单位：秒
	SnapshotInterval int `json:"snapshotInterval" validate:"required,gte=0"`
}
