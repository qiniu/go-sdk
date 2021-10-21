# Pili Streaming Cloud Server-Side Library for Go

## Features

- 推拉流URL && 签算方法
  - [x] RTMP 推流地址
  - [x] SRT 推流地址
  - [x] RTMP 播放地址
  - [x] HLS 播放地址
  - [x] HDL(FLV) 播放地址
  - [x] 推流 URL 签算（直播鉴权）
  - [x] 播放 URL 签算（时间戳防盗链）

- 直播空间管理（Hub）
  - [x] 查询直播空间列表 GET /v2/hubs
  - [x] 查询直播空间信息 GET /v2/hubs/\<hub>
  - [x] 修改直播空间推流鉴权配置 POST /v2/hubs/\<hub\>/security
  - [x] 修改直播空间hls低延迟配置 POST /v2/hubs/\<hub\>/hlsplus
  - [x] 修改直播空间存储配置 POST /v2/hubs/\<hub\>/persistence
  - [x] 修改直播空间封面配置 POST /v2/hubs/\<hub\>/snapshot

- 域名管理（Domain）
  - [x] 查询域名列表 GET /v2/hubs/\<hub\>/domains
  - [x] 查询域名信息 GET /v2/hubs/\<hub\>/domains/\<domain>
  - [x] 绑定直播域名 POST /v2/hubs/\<hub\>/newdomains
  - [x] 解绑直播域名 DELETE /v2/hubs/\<hub\>/domains/\<domain>
  - [x] 配置点播域名 POST /v2/hubs/\<hub\>/voddomain
  - [x] 修改域名证书配置 POST /v2/hubs/\<hub\>/domains/\<domain\>/cert
  - [x] 更新域名改写规则 POST /v2/hubs/\<hub\>/domains/\<domain\>/urlrewrite

- 直播流管理（Stream）
  - [x] 查询直播流列表 GET v2/hubs/\<hub\>/streams
  - [x] 查询直播流信息 GET v2/hubs/\<hub\>/streams/\<EncodedStreamTitle>
  - [x] 禁用直播流 POST /v2/hubs/\<hub\>/streams/\<EncodedStreamTitle\>/disabled
  - [x] 查询在线流信息 GET v2/hubs/\<hub\>/streams/\<EncodedStreamTitle\>/live
  - [x] 批量查询在线流信息 POST /v2/hubs/\<hub\>/livestreams
  - [x] 查询直播流推流记录 GET /v2/hubs/\<hub\>/streams/\<EncodedStreamTitle\>/historyactivity
  - [x] 录制直播回放 POST /v2/hubs/\<hub\>/streams/\<EncodedStreamTitle\>/saveas
  - [x] 保存直播截图 POST /v2/hubs/\<hub\>/streams/\<EncodedStreamTitle\>/snapshot
  - [x] 修改直播流转码配置 POST /v2/hubs/\<hub\>/streams/\<EncodedStreamTitle\>/converts

- 数据统计（Stat）
  - [x] 上行流量 GET /statd/upflow
  - [x] 下行流量 GET /statd/downflow
  - [x] 直播转码使用量 GET /statd/codec
  - [x] 直播鉴黄使用量 GET /statd/nrop
  - [x] 导播台使用量 GET /statd/caster
  - [x] Pub服务使用量 GET /statd/pub

## Usage

```go
package main

import (
	// ...
	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/pili"
)

var (
	AccessKey = "<QINIU ACCESS KEY>" // 替换成自己 Qiniu 账号的 AccessKey
	SecretKey = "<QINIU SECRET KEY>" // 替换成自己 Qiniu 账号的 SecretKey
)

func main() {
	// ...
	mac := auth.New(AccessKey, SecretKey)
	manager := pili.NewManager(mac)
	// ...
}
```