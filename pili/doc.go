// Package pili 提供七牛云直播服务（Pili）的 Go 客户端。
//
// 官方文档: https://developer.qiniu.com/pili
//
// # 创建客户端
//
//	manager := pili.NewManager(pili.ManagerConfig{
//	    AccessKey: "AK",
//	    SecretKey: "SK",
//	})
//
// # 直播空间管理
//
//   - [Manager.GetHubList] / [Manager.GetHubInfo]: 查询直播空间
//   - [Manager.HubSecurity] / [Manager.HubHlsplus] / [Manager.HubPersistence] / [Manager.HubSnapshot]: 空间配置
//
// # 流管理
//
//   - [Manager.GetStreamsList] / [Manager.GetStreamBaseInfo]: 查询流信息
//   - [Manager.GetStreamLiveStatus] / [Manager.BatchGetStreamLiveStatus]: 查询直播状态
//   - [Manager.StreamDisable]: 禁用/启用流
//   - [Manager.StreamSaveas]: 录制保存
//   - [Manager.StreamSnapshot]: 截图
//   - [Manager.GetStreamHistory]: 推流历史
//
// # 域名管理
//
//   - [Manager.GetDomainsList] / [Manager.GetDomainInfo]: 查询域名
//   - [Manager.BindDomain] / [Manager.UnbindDomain]: 绑定/解绑域名
//   - [Manager.SetDomainCert] / [Manager.SetDomainURLRewrite]: 域名配置
//
// # 推拉流地址生成
//
// 提供包级函数生成推拉流地址：
//
//	rtmpPushURL := pili.RTMPPublishURL("hub", "domain", "streamTitle")
//	hlsPlayURL := pili.HLSPlayURL("hub", "domain", "streamTitle")
//	signedURL, _ := pili.SignPublishURL(rtmpPushURL, pili.SignPublishURLArgs{...})
//
// 支持的协议: [RTMPPublishURL]、[SRTPublishURL]、[RTMPPlayURL]、[HLSPlayURL]、[HDLPlayURL]。
//
// # 数据统计
//
//   - [Manager.GetStatUpflow] / [Manager.GetStatDownflow]: 上下行流量统计
//   - [Manager.GetStatCodec] / [Manager.GetStatNrop] / [Manager.GetStatPub]: 编码、断流、推流统计
package pili
