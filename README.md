# Qiniu Go SDK

[![LICENSE](https://img.shields.io/github/license/qiniu/go-sdk.svg)](https://github.com/qiniu/go-sdk/blob/master/LICENSE)
[![Build Status](https://github.com/qiniu/go-sdk/workflows/Run%20Test%20Cases/badge.svg)](https://github.com/qiniu/go-sdk/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/qiniu/go-sdk)](https://goreportcard.com/report/github.com/qiniu/go-sdk)
[![GitHub release](https://img.shields.io/github/v/tag/qiniu/go-sdk.svg?label=release)](https://github.com/qiniu/go-sdk/releases)
[![codecov](https://codecov.io/gh/qiniu/go-sdk/branch/master/graph/badge.svg)](https://codecov.io/gh/qiniu/go-sdk)
[![GoDoc](https://pkg.go.dev/badge/github.com/qiniu/go-sdk/v7)](https://pkg.go.dev/github.com/qiniu/go-sdk/v7)

七牛云官方 Go SDK，提供对象存储、CDN、直播、实时音视频、短信、视频监控、IoT 设备联网、IAM、多媒体处理、审计日志、沙箱等服务的 Go 客户端。

## 安装

```bash
go get github.com/qiniu/go-sdk/v7
```

在 `go.mod` 中：

```
require github.com/qiniu/go-sdk/v7 v7.26.7
```

要求 **Go 1.22** 或更高版本。

## 快速开始

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/qiniu/go-sdk/v7/storagev2/credentials"
    "github.com/qiniu/go-sdk/v7/storagev2/http_client"
    "github.com/qiniu/go-sdk/v7/storagev2/uploader"
    "github.com/qiniu/go-sdk/v7/storagev2/uptoken"
)

func main() {
    // 创建凭证（AK/SK 从 https://portal.qiniu.com/user/key 获取）
    cred := credentials.NewCredentials("AccessKey", "SecretKey")

    // 创建上传策略和凭证
    putPolicy, _ := uptoken.NewPutPolicy("my-bucket", time.Now().Add(time.Hour))

    // 创建上传管理器
    uploadManager := uploader.NewUploadManager(&uploader.UploadManagerOptions{
        Options: http_client.Options{Credentials: cred},
    })

    // 上传文件
    objectName := "my-file.txt"
    err := uploadManager.UploadFile(context.Background(), "/path/to/file", &uploader.ObjectOptions{
        BucketName: "my-bucket",
        ObjectName: &objectName,
        UpToken:    uptoken.NewSigner(putPolicy, cred),
    }, nil)
    if err != nil {
        fmt.Println("上传失败:", err)
        return
    }
    fmt.Println("上传成功")
}
```

## 功能模块

### 对象存储

| 包 | 说明 |
|---|---|
| [`storagev2`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/storagev2) | **v2 API（推荐）** — 类型化请求/响应、自动区域检测、连接池、重试 |
| [`storagev2/uploader`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/storagev2/uploader) | 上传管理，自动选择表单上传或分片上传 |
| [`storagev2/downloader`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/storagev2/downloader) | 下载管理，支持并发分片下载 |
| [`storagev2/objects`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/storagev2/objects) | 对象管理，链式 API（查询、复制、移动、删除、归档恢复等） |
| [`storagev2/credentials`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/storagev2/credentials) | 凭证管理，CredentialsProvider 接口 |
| [`storagev2/uptoken`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/storagev2/uptoken) | 上传凭证，PutPolicy 上传策略 |
| [`storagev2/region`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/storagev2/region) | 区域信息和服务端点管理 |
| [`storagev2/apis`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/storagev2/apis) | 低级类型化 API（自动生成） |
| [`storage`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/storage) | v1 API — BucketManager、FormUploader、ResumeUploaderV2 |

### 其他云服务

| 包 | 说明 |
|---|---|
| [`auth`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/auth) | 认证和签名（AK/SK 凭证、请求签名、回调验证） |
| [`cdn`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/cdn) | CDN 加速（刷新、预取、流量/带宽查询、日志下载、时间戳防盗链） |
| [`pili`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/pili) | 直播服务（Hub/Stream 管理、推流/播放 URL 生成、统计数据） |
| [`rtc`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/rtc) | 实时音视频（RoomToken 生成、房间管理） |
| [`sms`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/sms) | 短信服务（发送短信、签名/模板管理） |
| [`qvs`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/qvs) | 视频监控（GB/T 28181 设备管理、流管理、录制、截图） |
| [`linking`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/linking) | IoT 设备联网（设备管理、设备密钥、点播/直播） |
| [`iam`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/iam) | 身份与访问管理（子账号、用户组、权限策略） |
| [`media`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/media) | 多媒体数据处理（持久化处理触发和查询） |
| [`audit`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/audit) | 审计日志（账号操作审计日志查询） |
| [`sandbox`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/sandbox) | 沙箱服务（沙箱生命周期管理、文件操作、命令执行、模板） |

### 基础设施

| 包 | 说明 |
|---|---|
| [`client`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/client) | 底层 HTTP 客户端 |
| [`storagev2/http_client`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/storagev2/http_client) | storagev2 HTTP 客户端和通用选项 |
| [`reqid`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/reqid) | 请求 ID 传递 |
| [`conf`](https://pkg.go.dev/github.com/qiniu/go-sdk/v7/conf) | SDK 全局配置 |

## 示例

[examples/](https://github.com/qiniu/go-sdk/tree/master/examples) 目录包含各功能模块的示例代码：

- **对象存储**：表单上传、断点续传、文件管理（stat/copy/move/delete/fetch）、列举文件、异步抓取、下载
- **CDN**：刷新、预取、流量/带宽查询、日志下载、时间戳防盗链
- **多媒体处理**：持久化数据处理（pfop）
- **实时音视频**：RTC 房间管理
- **沙箱**：创建、生命周期管理、文件操作、命令执行、模板管理

## 文档

- [七牛 SDK 文档站](https://developer.qiniu.com/kodo/sdk/1238/go)
- [GoDoc API 文档](https://pkg.go.dev/github.com/qiniu/go-sdk/v7)
- [项目 Wiki](https://github.com/qiniu/go-sdk/wiki)
- [AGENTS.md](AGENTS.md) — SDK 开发维护指南（适用于 AI 编码工具）
- [llms.txt](llms.txt) — AI 编码工具文档索引（支持 Context7 MCP）

## 许可证

[MIT](https://github.com/qiniu/go-sdk/blob/master/LICENSE)
