# 七牛云 Go SDK — GitHub Copilot 指引

七牛云官方 Go SDK（`github.com/qiniu/go-sdk/v7`），Go 1.22+，MIT 许可。

提供七牛云各服务的 Go 客户端：对象存储、CDN、直播、RTC、短信、IoT、IAM、多媒体处理、审计、沙箱等。

## 模块概览

- `auth` — 认证（AK/SK、签名、Token），`auth.New(ak, sk)` 创建凭证，`auth.Default()` 从环境变量加载
- `storage` — 对象存储 v1，Manager 模式：`BucketManager`（Bucket/对象管理）、`FormUploader`/`ResumeUploader`（上传）、`OperationManager`（数据处理 pfop）
- `storagev2` — 对象存储 v2，Provider/Interface 模式：`uploader.UploadManager`（上传）、`downloader.DownloadManager`（下载）、`objects.ObjectsManager`（对象管理），支持自动区域检测和重试
- `cdn` — CDN 加速：`CdnManager`（刷新、预取、流量/带宽查询），`CreateTimestampAntileechURL()`（时间戳防盗链）
- `pili` — 直播云：流管理、域名管理、推拉流地址生成（`RTMPPublishURL`、`HLSPlayURL` 等）
- `rtc` — 实时音视频：应用管理、房间 Token 生成、用户管理
- `sms` — 短信：发送短信、签名/模板管理
- `linking` — IoT 设备联网：设备管理、设备密钥、录像片段
- `iam` — 身份与访问管理：用户、组、策略的 CRUD
- `media` — 多媒体处理：`Pfop`（触发处理）、`Prefop`（查询状态）
- `audit` — 审计日志：`QueryLog` 查询
- `sandbox` — 沙箱：创建/连接沙箱、文件操作（`Files()`）、命令执行（`Commands()`）、PTY 终端（`Pty()`）、模板管理

## 关键模式

- **上传凭证**：`storage.PutPolicy{Scope: "bucket"}.UploadToken(mac)`，或 v2 的 `uptoken.NewSigner(putPolicy, cred)`
- **构造函数**：`New*()` 或 `New*Ex()`
- **v1 Manager 构造**：`storage.NewBucketManager(mac, &cfg)`、`cdn.NewCdnManager(mac)`
- **v2 客户端构造**：`apis.NewStorage(&httpclient.Options{Credentials: cred})`
- **错误类型**：根包 `QError`（Code + Message）、`storagev2/errors` 类型化错误
- **注释默认中文**，导出标识符注释以标识符名称开头

## 详细 API 参考

完整的模块 API 列表、使用示例见项目根目录 `AGENTS.md`。
