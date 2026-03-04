// Package qvs 提供七牛云视频监控（QVS）服务的 Go 客户端。
//
// QVS 支持通过 GB/T 28181 协议和 RTMP 推流方式接入视频设备，
// 提供设备管理、流管理、录制和截图等功能。
//
// 官方文档: https://developer.qiniu.com/qvs
//
// # 创建客户端
//
//	mac := auth.New("AccessKey", "SecretKey")
//	manager := qvs.NewManager(mac, nil)
//
// # 空间管理
//
//   - [Manager.AddNamespace] / [Manager.QueryNamespace] / [Manager.DeleteNamespace]
//   - [Manager.ListNamespace] / [Manager.UpdateNamespace]
//   - [Manager.EnableNamespace] / [Manager.DisableNamespace]
//   - [Manager.AddDomain] / [Manager.DeleteDomain] / [Manager.ListDomain]
//
// # 设备管理
//
//   - [Manager.AddDevice] / [Manager.QueryDevice] / [Manager.DeleteDevice]
//   - [Manager.ListDevice] / [Manager.UpdateDevice]
//   - [Manager.StartDevice] / [Manager.StopDevice]
//   - [Manager.ListChannels] / [Manager.QueryChannel] / [Manager.FetchCatalog]
//
// # 流管理
//
//   - [Manager.AddStream] / [Manager.QueryStream] / [Manager.DeleteStream]
//   - [Manager.ListStream] / [Manager.UpdateStream]
//   - [Manager.EnableStream] / [Manager.DisableStream] / [Manager.StopStream]
//   - [Manager.DynamicPublishPlayURL] / [Manager.StaticPublishPlayURL]
//   - [Manager.OndemandSnap] / [Manager.StreamsSnapshots] / [Manager.DeleteSnapshots]
//
// # 录制管理
//
//   - [Manager.StartRecord] / [Manager.StopRecord]
//   - [Manager.RecordClipsSaveas]: 合并录制片段
//   - [Manager.RecordsPlayback]: 获取回放地址（M3U8）
//
// # 模板管理
//
//   - [Manager.AddTemplate] / [Manager.QueryTemplate] / [Manager.DeleteTemplate]
//
// # 统计查询
//
//   - [Manager.QueryFlow]: 流量统计
//   - [Manager.QueryBandwidth]: 带宽统计
package qvs
