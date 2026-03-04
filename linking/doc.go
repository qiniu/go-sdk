// Package linking 提供七牛云 IoT 设备联网平台的 Go 客户端。
//
// 官方文档: https://developer.qiniu.com/linking
//
// # 创建客户端
//
//	manager := linking.NewManager(mac, nil)
//
// # 设备管理
//
//   - [Manager.AddDevice] / [Manager.QueryDevice] / [Manager.UpdateDevice] / [Manager.DeleteDevice]: 设备 CRUD
//   - [Manager.ListDevice]: 列举设备，支持按前缀、在线状态、类型过滤
//
// # 设备密钥
//
//   - [Manager.AddDeviceKey] / [Manager.QueryDeviceKey] / [Manager.DeleteDeviceKey]: 密钥管理
//   - [Manager.UpdateDeviceKeyState]: 启用/禁用密钥
//   - [Manager.CloneDeviceKey]: 克隆设备密钥到另一台设备
//
// # 录像与直播
//
//   - [Manager.Segments]: 查询录像片段
//   - [Manager.Saveas]: 录像存储（转存为指定格式）
//   - [Manager.StartLive]: 开始直播
//
// # 设备历史与统计
//
//   - [Manager.ListDeviceHistoryactivity]: 设备上下线历史
//   - [Manager.Stat]: 设备统计
//   - [Manager.RPC]: 向设备发送 RPC 命令
package linking
