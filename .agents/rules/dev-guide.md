# 七牛云 Go SDK — 开发维护指引

七牛云官方 Go SDK（`github.com/qiniu/go-sdk/v7`），Go 1.22+，MIT 许可。

完整开发维护指南见项目根目录 `AGENTS.md`。

## 项目结构

- `auth` — 认证凭证（AK/SK）、签名
- `storage` — 对象存储 v1（Manager 模式：BucketManager、FormUploader、ResumeUploader）
- `storagev2` — 对象存储 v2（Provider/Interface 模式，代码生成，推荐新功能使用）
- `cdn` / `pili` / `rtc` / `sms` / `linking` / `qvs` — 各业务服务
- `iam` / `media` / `audit` — 生成代码（`make generate` 维护）
- `sandbox` — 沙箱环境（`make generate-sandbox` 维护）
- `client` / `conf` / `reqid` — 基础设施包
- `internal/clientv2` — HTTP 拦截器链（Auth、Retry、AntiHijacking 等）
- `internal/api-generator` — YAML API 规范 → Go 代码生成器
- `api-specs/` — API 规范（git submodule）
- `examples/` — 使用示例（独立 main 包）

## 架构模式

- **Manager 模式**（storage v1）：`New*(mac, &cfg)` 构造，通过参数传递认证
- **Provider/Interface 模式**（storagev2）：通过 `http_client.Options` 注入凭证、区域、重试等组件
- **Interceptor 拦截器链**（internal/clientv2）：按优先级排序的请求处理链
- **双版本共存**：storage v1 和 storagev2 v2 并行维护，共享 auth/client/conf/reqid

## 编码规范

- `gofmt -s` 格式化（CI 检查）
- `staticcheck` 静态检查（CI 检查）
- 注释使用**中文**，导出标识符注释以名称开头（godoc 规范）
- 构造函数 `New*()` / `New*Ex()` 命名
- 错误使用 `fmt.Errorf("context: %w", err)` 包装
- 包级别注释写在 `doc.go` 文件中

## 测试要求

- 测试文件添加 `//go:build unit` 或 `//go:build integration` 标签
- `make unittest` — 单元测试（提交前必须通过）
- `make staticcheck` — 静态检查（提交前必须通过）
- 使用 `testify/assert` 断言，推荐表驱动测试

## CI 流程

gofmt 检查 → staticcheck → 编译 examples → 单元测试（Go 1.22 + stable，Linux → Windows → macOS）

## 关键约定

- 不要修改 `DO NOT EDIT DIRECTLY` 标记的生成代码
- 修改 API 规范后运行 `make generate` 重新生成
- 发布时同步 `conf/conf.go`、`CHANGELOG.md`、`README.md` 版本号
- `api-specs/` 是 git submodule，更新需在 submodule 中操作
