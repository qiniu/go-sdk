package pili

import (
	"encoding/base64"
)

// GetStreamListRequest 查询直播流列表请求参数
type GetStreamListRequest struct {

	// Hub 直播空间
	Hub string `validate:"required"`

	// LiveOnly 只返回当前在线的流
	LiveOnly bool `validate:"-"`

	// Prefix 流名称前缀匹配
	Prefix string `validate:"-"`

	// Limit 返回值数量限制
	Limit int `validate:"gte=0,lte=5000"`

	// Marker 游标
	// 从该位置开始返回
	Marker string `validate:"-"`
}

// GetStreamsListResponse 查询直播流列表返回值
type GetStreamsListResponse struct {

	// Items 流列表
	Items []GetStreamsListResponseItem `json:"items"`

	// Marker 游标
	// 表示当前位置，若marker为空，表示遍历完成
	Marker string `json:"marker"`
}

type GetStreamsListResponseItem struct {

	// Key 流名
	Key string `json:"key"`
}

// GetStreamBaseInfoRequest 查询直播流信息请求参数
type GetStreamBaseInfoRequest struct {

	// Hub 直播空间
	Hub string `validate:"required"`

	// Stream 流名
	Stream string `validate:"required"`
}

// GetStreamBaseInfoResponse 查询直播流信息返回值
type GetStreamBaseInfoResponse struct {

	// CreatedAt 直播流创建时间
	// Unix 时间戳，单位：秒
	CreatedAt int64 `json:"createdAt"`

	// UpdatedAt 直播流更新时间
	// Unix 时间戳，单位：秒
	UpdatedAt int64 `json:"updatedAt"`

	// ExpireAt 直播流过期时间
	// 默认流过期时间15天，Unix 时间戳，单位：秒
	ExpireAt int64 `json:"expireAt"`

	// DisabledTill 禁用结束时间
	DisabledTill int `json:"disabledTill"`

	// Converts 转码配置
	Converts []string `json:"converts"`

	// Watermark 是否开启水印
	Watermark bool `json:"watermark"`

	// PublishSecurity 推流鉴权类型
	PublishSecurity string `json:"publishSecurity"`

	// PublishKey 推流密钥
	PublishKey string `json:"publishKey"`

	// NropEnable 是否开启鉴黄
	NropEnable bool `json:"nropEnable"`
}

// StreamDisabledRequest 禁用直播流请求参数
type StreamDisabledRequest struct {

	// Hub 直播空间
	Hub string `json:"-" validate:"required"`

	// Stream 流名
	Stream string `json:"-" validate:"required"`

	// DisabledTill 禁用结束时间
	// Unix 时间戳，单位：秒
	// 特殊值 -1 表示永久禁用；0 解除禁用
	DisabledTill int `json:"disabledTill" validate:"-"`

	// DisablePeriodSecond 禁用时长
	// 单位：秒
	// 当 DisabledTill 为0时，DisablePeriodSecond 参数生效
	DisablePeriodSecond int `json:"disablePeriodSecond" validate:"gte=0"`
}

// GetStreamLiveStatusRequest 查询直播流实时信息请求参数
type GetStreamLiveStatusRequest struct {

	// Hub 直播空间
	Hub string `validate:"required"`

	// Stream 流名
	Stream string `validate:"required"`
}

// StreamLiveStatus 直播流实时状态
type StreamLiveStatus struct {

	// StartAt 推流开始时间
	// Unix 时间戳，单位：秒
	StartAt int64 `json:"startAt"`

	// ClientIP 推流端IP
	ClientIP string `json:"clientIP"`

	// ServerIP 服务端IP
	ServerIP string `json:"serverIP"`

	// BPS 推流码率
	BPS int `json:"bps"`

	// FPS 帧率
	FPS StreamLiveStatusFPS `json:"fps"`

	// Key 流名
	Key string `json:"key,omitempty"`
}

// StreamLiveStatusFPS 流实时帧率
type StreamLiveStatusFPS struct {

	// Audio 音频帧率
	Audio int `json:"audio"`

	// Video 视频帧率
	Video int `json:"video"`

	// Data metadata帧率
	Data int `json:"data"`
}

// GetStreamLiveStatusResponse 查询直播流实时信息返回值
type GetStreamLiveStatusResponse struct {

	// StreamLiveStatus 直播流实时状态
	StreamLiveStatus

	// 最近30秒内GOP最大值，单位：毫秒
	MaxGop int `json:"maxGop"`

	// 最近30秒内GOP平均值，单位：毫秒
	AvgGop int `json:"avgGop"`

	// VideoBitRate 视频码率，单位：bps
	VideoBitRate int `json:"videoBitRate"`

	// AudioBitRate 音频码率，单位：bps
	AudioBitRate int `json:"audioBitRate"`

	// 播放人数
	PlayCount int64 `json:"playCount"`

	// 播放带宽
	Bandwidth int64 `json:"bandwidth"`

	// 推流端所在地区
	Location string `json:"location"`

	// 推流端所在运营商
	Isp string `json:"isp"`
}

// BatchGetStreamLiveStatusRequest 批量查询直播实时状态请求参数
type BatchGetStreamLiveStatusRequest struct {

	// Hub 直播空间
	Hub string `json:"-" validate:"required"`

	// Items 流列表
	// 查询流列表数量不超过100
	Items []string `json:"items" validate:"-"`
}

// BatchGetStreamLiveStatusResponse 批量查询直播实时状态返回值
type BatchGetStreamLiveStatusResponse struct {

	// Items 流列表
	Items []StreamLiveStatus `json:"items"`
}

// GetStreamHistoryRequest 查询直播流推流记录请求参数
type GetStreamHistoryRequest struct {

	// Hub 直播空间
	Hub string `validate:"required"`

	// Stream 流名
	Stream string `validate:"required"`

	// Start 开始时间
	// 单位：秒，默认为0
	Start int64 `validate:"gte=0"`

	// End 结束时间
	// 单位：秒，默认为当前时间
	End int64 `validate:"gte=0"`
}

// GetStreamHistoryResponse 查询直播流推流记录返回值
type GetStreamHistoryResponse struct {

	// Items 直播流推流记录列表
	Items []GetStreamHistoryItem `json:"items"`
}

// GetStreamHistoryItem 查询直播流推流记录项
type GetStreamHistoryItem struct {

	// Start 推流开始时间
	Start int64 `json:"start"`

	// End 推流结束时间
	End int64 `json:"end"`

	// ClientIP 推流端IP
	ClientIP string `json:"clientIP"`

	// ServerIP 服务端IP
	ServerIP string `json:"serverIP"`
}

// StreamSaveasRequest 录制直播回放请求参数
type StreamSaveasRequest struct {

	// Hub 直播空间
	Hub string `json:"-" validate:"required"`

	// Stream 流名
	Stream string `json:"-" validate:"required"`

	// Start 开始时间
	// Unix 时间戳，单位：秒
	Start int64 `json:"start" validate:"gte=0"`

	// End 结束时间
	// Unix 时间戳，单位：秒，默认为当前时间
	End int64 `json:"end" validate:"gte=0"`

	// Fname 文件名
	// 为空时会随机生成一个文件名
	Fname string `json:"fname" validate:"-"`

	// Format 文件格式
	// 可选文件格式为
	// - m3u8: HLS格式，默认值
	// - flv: FLV格式，将回放切片转封装为单个flv文件，异步模式
	// - mp4: MP4格式，将回放切片转封装为单个mp4文件，异步模式
	// 异步模式下，生成回放文件需要一定时间
	Format string `json:"format" validate:"-"`

	// Pipeline 异步模式时，指定数据处理的队列
	// 可以将优先级较高的任务配置到独立的队列中进行执行
	// 参考文档：https://developer.qiniu.com/dora/kb/2500/streaming-media-queue-about-seven-cows
	Pipeline string `json:"pipeline" validate:"-"`

	// Notify 回调地址
	// 异步模式完成任务后的回调通知地址，不指定表示不做回调
	Notify string `json:"notify" validate:"-"`

	// ExpireDays 切片文件的生命周期
	// - 0: 默认值，表示修改ts文件生命周期为永久保存
	// - >0: 表示修改ts文件的的生命周期为 ExpireDays 参数值
	// - -1: 表示不修改ts文件的expire属性，可显著提升接口响应速度
	ExpireDays int `json:"expireDays" validate:"-"`

	// PersistentDeleteAfterDays 生成文件的生命周期
	// - 0: 默认值，表示生成文件（m3u8/flv/mp4）永久保存
	// - >0: 表示生成文件（m3u8/flv/mp4）的生命周期为 PersistentDeleteAfterDays 参数值
	// m3u8文件只有当ExpireDays为-1时生效
	PersistentDeleteAfterDays int `json:"persistentDeleteAfterDays" validate:"-"`

	// FirstTsType 过滤ts切片文件类型
	// 部分非标准的直播流，在推流初期缺少视频帧或音频帧，过滤功能可以剔除这部分切片
	// - 0: 默认值，不做过滤
	// - 1: 第一个ts切片需要是纯视频类型，不符合预期的ts切片将被跳过
	// - 2: 第一个ts切片需要是纯音频类型，不符合预期的ts切片将被跳过
	// - 3: 第一个ts切片需要是音视频类型，不符合预期的ts切片将被跳过
	FirstTsType byte `json:"firstTsType" validate:"-"`
}

// StreamSaveasResponse 录制直播回放返回值
type StreamSaveasResponse struct {

	// Fname 文件名
	Fname string `json:"fname"`

	// Start 开始时间
	// Unix 时间戳，单位：秒
	Start int64 `json:"start"`

	// End 结束时间
	// Unix 时间戳，单位：秒
	End int64 `json:"end"`

	// PersistentID 异步任务ID
	PersistentID string `json:"persistentID"`
}

// StreamSnapshotRequest 保存直播截图请求参数
type StreamSnapshotRequest struct {

	// Hub 直播空间
	Hub string `json:"-" validate:"required"`

	// Stream 流名
	Stream string `json:"-" validate:"required"`

	// Time 截图时间
	// Unix 时间戳，单位：秒，不指定则为当前时间
	Time int `json:"time" validate:"-"`

	// Fname 文件名
	// 不指定系统会随机生成
	Fname string `json:"fname" validate:"-"`

	// Format 文件格式
	// 默认为jpg，支持选择jpg/png
	Format string `json:"format" validate:"-"`

	// Pipeline 异步模式时，指定数据处理的队列
	// 可以将优先级较高的任务配置到独立的队列中进行执行
	// 参考文档：https://developer.qiniu.com/dora/kb/2500/streaming-media-queue-about-seven-cows
	Pipeline string `json:"pipeline" validate:"-"`

	// Notify 回调地址
	// 若指定回调地址，则截图动作为异步模式
	Notify string `json:"notify" validate:"-"`

	// DeleteAfterDays 生命周期
	// - 0: 默认值，表示截图文件永久保存，单位：天
	DeleteAfterDays int `json:"deleteAfterDays" validate:"gte=0"`
}

// StreamSnapshotResponse 保存直播截图返回值
type StreamSnapshotResponse struct {

	// Fname 文件名
	Fname string `json:"fname"`

	// PersistentID 异步任务ID
	PersistentID string `json:"persistentID"`
}

// StreamConvertsRequest 修改直播流转码配置请求参数
type StreamConvertsRequest struct {

	// Hub 直播空间
	Hub string `json:"-" validate:"required"`

	// Stream 流名
	Stream string `json:"-" validate:"required"`

	// Converts 转码配置
	Converts []string `json:"converts" validate:"-"`
}

// encodeStream 请求流名base64编码
func encodeStream(str string) string {
	return base64.URLEncoding.EncodeToString([]byte(str))
}
