# 七牛云 Go SDK 使用指南

七牛云官方 Go SDK，模块路径 `github.com/qiniu/go-sdk/v7`，Go 1.22+。

安装：

```bash
go get github.com/qiniu/go-sdk/v7
```

## 认证（auth）

所有服务都需要先创建凭证：

```go
import "github.com/qiniu/go-sdk/v7/auth"

mac := auth.New("AccessKey", "SecretKey")

// 或从环境变量加载（QINIU_ACCESS_KEY、QINIU_SECRET_KEY）
mac := auth.Default()
```

`Credentials` 主要方法：
- `Sign(data) string` — HMAC-SHA1 签名
- `SignRequest(req) (string, error)` — 签名 HTTP 请求（Qbox 格式）
- `SignRequestV2(req) (string, error)` — 签名 HTTP 请求（Qiniu V2 格式）
- `SignWithData(b) string` — 签名并包含编码数据（用于上传凭证）
- `VerifyCallback(req) (bool, error)` — 验证回调请求签名

## 对象存储 v1（storage）

### 上传凭证

```go
import "github.com/qiniu/go-sdk/v7/storage"

putPolicy := storage.PutPolicy{
    Scope: "bucket",           // 或 "bucket:key" 覆盖上传
    // 可选配置：
    // Expires:       3600,    // 过期时间（秒），默认 3600
    // ReturnBody:    `{"key":"$(key)","hash":"$(etag)"}`,
    // CallbackURL:   "http://your-server.com/callback",
    // PersistentOps: "avthumb/mp4",
    // FsizeLimit:    10 * 1024 * 1024,  // 限制文件大小
    // FileType:      1,       // 0 标准 / 1 低频 / 2 归档 / 3 深度归档
}
upToken := putPolicy.UploadToken(mac)
```

### 表单上传（小文件）

```go
cfg := storage.Config{UseHTTPS: true, UseCdnDomains: false}
formUploader := storage.NewFormUploader(&cfg)

ret := storage.PutRet{}
err := formUploader.PutFile(ctx, &ret, upToken, "key", "/path/to/file", nil)
// 或从 io.Reader 上传：
err := formUploader.Put(ctx, &ret, upToken, "key", reader, size, nil)
```

### 分片上传（大文件）

```go
// V1 分片上传
resumeUploader := storage.NewResumeUploader(&cfg)
err := resumeUploader.PutFile(ctx, &ret, upToken, "key", "/path/to/large-file", nil)

// V2 分片上传
resumeUploaderV2 := storage.NewResumeUploaderV2(&cfg)
err := resumeUploaderV2.PutFile(ctx, &ret, upToken, "key", "/path/to/large-file", nil)
```

### UploadManager（自动选择上传方式）

```go
uploadManager := storage.NewUploadManager(&storage.UploadManagerConfig{Options: &cfg})
ret, err := uploadManager.Put(ctx, &storage.UploadManagerPutParam{...})
```

### Bucket 和对象管理

```go
bucketManager := storage.NewBucketManager(mac, &cfg)
```

**Bucket 操作：**
- `Buckets(shared) ([]string, error)` — 列举 Bucket
- `CreateBucket(name, regionID)` / `DropBucket(name)` — 创建/删除 Bucket
- `GetBucketInfo(name)` — 获取 Bucket 信息
- `MakeBucketPublic(name)` / `MakeBucketPrivate(name)` — 设置访问权限
- `SetTagging` / `GetTagging` / `ClearTagging` — 标签管理
- `ListBucketDomains(name)` — 列举绑定域名

**对象操作：**
- `Stat(bucket, key)` — 获取文件信息
- `Copy(srcBucket, srcKey, dstBucket, dstKey, force)` — 复制
- `Move(srcBucket, srcKey, dstBucket, dstKey, force)` — 移动/重命名
- `Delete(bucket, key)` — 删除
- `DeleteAfterDays(bucket, key, days)` — 设置过期删除
- `ChangeMime(bucket, key, mime)` — 修改 MIME
- `ChangeType(bucket, key, fileType)` — 修改存储类型
- `RestoreAr(bucket, key, freezeAfterDays)` — 解冻归档文件
- `Fetch(resURL, bucket, key)` — 抓取网络资源到空间
- `ListFiles(bucket, prefix, delimiter, marker, limit)` — 列举文件

**批量操作：**
- `Batch(operations)` — 批量执行操作

### 数据处理（pfop）

```go
operationManager := storage.NewOperationManager(mac, &cfg)
persistentID, err := operationManager.Pfop(bucket, key, fops, pipeline, notifyURL, force)
ret, err := operationManager.Prefop(persistentID) // 查询处理状态
```

## 对象存储 v2（storagev2）

v2 是新一代 API，提供类型化的请求/响应、自动区域检测、连接池、重试等特性。

### 凭证

```go
import "github.com/qiniu/go-sdk/v7/storagev2/credentials"

cred := credentials.NewCredentials("AccessKey", "SecretKey")
// 或从环境变量加载
cred := credentials.Default()
```

### 上传

```go
import (
    "time"
    "github.com/qiniu/go-sdk/v7/storagev2/http_client"
    "github.com/qiniu/go-sdk/v7/storagev2/uploader"
    "github.com/qiniu/go-sdk/v7/storagev2/uptoken"
)

putPolicy, _ := uptoken.NewPutPolicy("bucket", time.Now().Add(time.Hour))
uploadManager := uploader.NewUploadManager(&uploader.UploadManagerOptions{
    Options: http_client.Options{Credentials: cred},
})

objectName := "my-file.txt"
err := uploadManager.UploadFile(ctx, "/path/to/file", &uploader.ObjectOptions{
    BucketName: "bucket",
    ObjectName: &objectName,
    UpToken:    uptoken.NewSigner(putPolicy, cred),
}, nil)

// 也支持：
// uploadManager.UploadReader(ctx, reader, objectOptions, returnValue)
// uploadManager.UploadDirectory(ctx, dirPath, directoryOptions)
```

### 下载

```go
import "github.com/qiniu/go-sdk/v7/storagev2/downloader"

downloadManager := downloader.NewDownloadManager(&downloader.DownloadManagerOptions{...})
n, err := downloadManager.DownloadToFile(ctx, "objectName", "/local/path", options)
// 也支持：
// downloadManager.DownloadToWriter(ctx, objectName, writer, options)
// downloadManager.DownloadDirectory(ctx, dirPath, options)
```

### 对象管理

```go
import "github.com/qiniu/go-sdk/v7/storagev2/objects"

objectsManager := objects.NewObjectsManager(&objects.ObjectsManagerOptions{...})

// 流式 API
bucket := objectsManager.Bucket("my-bucket")
obj := bucket.Object("my-key")

// 单个操作
statOp := obj.Stat()
err := statOp.Call(ctx) // 执行
info := statOp.Result() // 获取结果

// 批量操作
copyOp := obj.CopyTo("dst-bucket", "dst-key")
deleteOp := bucket.Object("another-key").Delete()
objectsManager.Batch(ctx, []objects.Operation{copyOp, deleteOp}, nil)

// 列举文件
lister := bucket.List(ctx, &objects.ListObjectsOptions{Prefix: "photos/"})
```

### 低级 API 客户端

```go
import (
    "github.com/qiniu/go-sdk/v7/storagev2/apis"
    "github.com/qiniu/go-sdk/v7/storagev2/http_client"
)

storageClient := apis.NewStorage(&http_client.Options{Credentials: cred})
// 所有操作都是类型化的方法：
resp, err := storageClient.GetBucketInfo(ctx, &apis.GetBucketInfoRequest{...}, nil)
```

## CDN（cdn）

```go
import "github.com/qiniu/go-sdk/v7/cdn"

cdnManager := cdn.NewCdnManager(mac)
```

- `RefreshUrls(urls)` / `RefreshDirs(dirs)` — 刷新缓存
- `PrefetchUrls(urls)` — 预取
- `GetFluxData(start, end, granularity, domains)` — 流量查询
- `GetBandwidthData(start, end, granularity, domains)` — 带宽查询
- `GetCdnLogList(day, domains)` — 日志查询

**时间戳防盗链：**
```go
signedURL, _ := cdn.CreateTimestampAntileechURL(rawURL, encryptKey, durationInSeconds)
```

## 直播（pili）

```go
import "github.com/qiniu/go-sdk/v7/pili"

manager := pili.NewManager(pili.ManagerConfig{
    AccessKey: "AK",
    SecretKey: "SK",
})
```

**直播空间：** `GetHubList` / `GetHubInfo`

**流管理：** `GetStreamsList` / `GetStreamBaseInfo` / `GetStreamLiveStatus` / `StreamDisable` / `StreamSaveas` / `StreamSnapshot`

**域名管理：** `GetDomainsList` / `BindDomain` / `UnbindDomain`

**统计：** `GetStatUpflow` / `GetStatDownflow` / `GetStatCodec` 等

**推拉流地址生成（包级函数）：**
```go
rtmpPushURL := pili.RTMPPublishURL("hub", "domain", "streamTitle")
hlsPlayURL := pili.HLSPlayURL("hub", "domain", "streamTitle")
signedURL, _ := pili.SignPublishURL(rtmpPushURL, pili.SignPublishURLArgs{...})
```

## 实时音视频（rtc）

```go
import "github.com/qiniu/go-sdk/v7/rtc"

manager := rtc.NewManager(mac)
```

- `CreateApp(conf)` / `GetApp(appID)` / `UpdateApp` / `DeleteApp` — 应用管理
- `GetRoomToken(roomAccess)` — 生成房间 Token
- `ListUser(appID, roomName)` / `KickUser` — 用户管理
- `ListActiveRooms(appID, prefix, offset, limit)` — 房间查询

## 短信（sms）

```go
import "github.com/qiniu/go-sdk/v7/sms"

manager := sms.NewManager(mac)
```

- `SendMessage(MessagesRequest{SignatureID, TemplateID, Mobiles, Parameters})` — 发送短信
- `CreateSignature` / `QuerySignature` / `DeleteSignature` — 签名管理
- `CreateTemplate` / `QueryTemplate` / `DeleteTemplate` — 模板管理

## IoT 设备联网（linking）

```go
import "github.com/qiniu/go-sdk/v7/linking"

manager := linking.NewManager(mac, nil)
```

- `AddDevice` / `QueryDevice` / `UpdateDevice` / `DeleteDevice` / `ListDevice` — 设备管理
- `AddDeviceKey` / `QueryDeviceKey` / `DeleteDeviceKey` — 设备密钥
- `Segments(appid, device, start, end, ...)` — 查询录像片段
- `Saveas(appid, device, start, end, fname, format)` — 录像存储
- `StartLive(req)` — 开始直播

## IAM（iam）

```go
import (
    "github.com/qiniu/go-sdk/v7/iam/apis"
    "github.com/qiniu/go-sdk/v7/storagev2/http_client"
)

iamClient := apis.NewIam(&http_client.Options{Credentials: cred})
```

- **用户：** `CreateUser` / `GetUser` / `GetUsers` / `ModifyUser` / `DeleteUser`
- **用户密钥：** `CreateUserKeypairs` / `GetUserKeypairs` / `EnableUserKeypair` / `DisableUserKeypair`
- **组：** `CreateGroup` / `GetGroup` / `GetGroups` / `ModifyGroup` / `DeleteGroup`
- **策略：** `CreatePolicy` / `GetPolicy` / `GetPolicies` / `ModifyPolicy` / `DeletePolicy`
- **关联：** `GetUserPolicies` / `GetGroupPolicies` / `UpdatePolicyUsers` / `UpdatePolicyGroups`

所有方法签名：`Method(ctx, request, options) (response, error)`

## 多媒体处理（media）

```go
import (
    "github.com/qiniu/go-sdk/v7/media/apis"
    "github.com/qiniu/go-sdk/v7/storagev2/http_client"
)

mediaClient := apis.NewMedia(&http_client.Options{Credentials: cred})
```

- `Pfop(ctx, request, options)` — 触发持久化数据处理
- `Prefop(ctx, request, options)` — 查询处理状态

## 审计日志（audit）

```go
import (
    "github.com/qiniu/go-sdk/v7/audit/apis"
    "github.com/qiniu/go-sdk/v7/storagev2/http_client"
)

auditClient := apis.NewAudit(&http_client.Options{Credentials: cred})
```

- `QueryLog(ctx, request, options)` — 查询账号审计日志

## 沙箱（sandbox）

```go
import "github.com/qiniu/go-sdk/v7/sandbox"

client, _ := sandbox.NewClient(&sandbox.Config{
    APIKey: "your-api-key",
    // Endpoint: "https://cn-yangzhou-1-sandbox.qiniuapi.com", // 可选
})
```

### 沙箱生命周期

```go
// 创建并等待就绪
sb, info, _ := client.CreateAndWait(ctx, sandbox.CreateParams{
    TemplateID: "template-id",
    // Timeout, EnvVars, Metadata 等可选
})

// 或分步创建
sb, _ := client.Create(ctx, sandbox.CreateParams{TemplateID: "template-id"})
info, _ := sb.WaitForReady(ctx)

// 连接已有沙箱
sb, _ := client.Connect(ctx, "sandbox-id", sandbox.ConnectParams{})
```

- `sb.Kill(ctx)` / `sb.Pause(ctx)` / `sb.Refresh(ctx, params)` — 生命周期控制
- `sb.GetInfo(ctx)` / `sb.IsRunning(ctx)` / `sb.GetMetrics(ctx, params)` — 状态查询
- `sb.GetHost(port)` — 获取外部访问地址
- `client.List(ctx, params)` — 列举沙箱

### 文件操作

```go
fs := sb.Files()

data, _ := fs.Read(ctx, "/app/main.go")
text, _ := fs.ReadText(ctx, "/app/main.go")
fs.Write(ctx, "/app/main.go", []byte("package main"), sandbox.WithUser("user"))
fs.List(ctx, "/app/")
fs.MakeDir(ctx, "/app/src/")
fs.Remove(ctx, "/app/tmp/")
fs.Rename(ctx, "/app/old.go", "/app/new.go")
fs.Exists(ctx, "/app/main.go")
```

### 命令执行

```go
import "time"

cmds := sb.Commands()

// 同步执行
result, _ := cmds.Run(ctx, "go build ./...", sandbox.WithCwd("/app"), sandbox.WithTimeout(30*time.Second))

// 异步执行
handle, _ := cmds.Start(ctx, "go run main.go")
result, _ := handle.Wait()
handle.Kill(ctx)

// 进程管理
cmds.List(ctx)                       // 列出进程
cmds.SendStdin(ctx, pid, data)       // 发送输入
cmds.Kill(ctx, pid)                  // 终止进程
```

### PTY 终端

```go
pty := sb.Pty()
handle, _ := pty.Create(ctx, sandbox.PtySize{Rows: 24, Cols: 80})
pty.SendInput(ctx, handle.PID(), []byte("ls -la\n"))
pty.Resize(ctx, handle.PID(), sandbox.PtySize{Rows: 48, Cols: 120})
```

### 模板管理

```go
client.ListTemplates(ctx, params)
client.CreateTemplate(ctx, body)
client.GetTemplate(ctx, templateID, params)
client.WaitForBuild(ctx, templateID, buildID)
```

## 使用示例

`examples/` 目录包含各功能的完整示例：

- `form_upload_simple` / `resume_upload_simple` / `resume_upload_advanced` — 上传
- `rs_stat` / `rs_copy` / `rs_move` / `rs_delete` / `rs_fetch` — 对象操作
- `rs_list_files` / `rs_list_bucket` — 列举
- `rs_batch_*` — 批量操作
- `cdn_*` — CDN 操作
- `sandbox_*` — 沙箱操作
- `create_uptoken` — 上传凭证生成
- `prefop` / `video_pfop` — 数据处理
