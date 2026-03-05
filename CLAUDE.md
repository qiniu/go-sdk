# Qiniu Go SDK — Claude Code Instructions

## 项目概要

七牛云 Go SDK，模块路径 `github.com/qiniu/go-sdk/v7`，Go 1.22+，MIT 许可。

## 核心开发规范

@AGENTS.md

## Claude Code 特定指引

### 常用命令

| 命令 | 说明 | 时机 |
|------|------|------|
| `make unittest` | 运行单元测试 | 修改前确认状态 + 提交前必须通过 |
| `make staticcheck` | 运行静态检查 | 提交前必须通过 |
| `make generate` | API 代码生成 | 修改 api-specs/ 中的 YAML 规范后 |
| `make generate-sandbox` | Sandbox 代码生成 | 修改 sandbox OpenAPI/protobuf 规范后 |
| `gofmt -s -w .` | 格式化代码 | 提交前 |
| `git submodule update --init --recursive` | 初始化 submodule | 首次 clone 或 submodule 更新后 |

### 开发注意事项

- 修改代码前先运行 `make unittest` 确认当前状态
- **绝不修改** `DO NOT EDIT DIRECTLY` 标记的生成代码，需修改请更新 API 规范后重新生成
- 注释默认使用**中文**，导出标识符注释以标识符名称开头
- 测试文件必须添加 `//go:build unit` 或 `//go:build integration` 标签
- `api-specs/` 是 git submodule，使用 `git submodule update --init --recursive` 初始化
- 发布时确保 `conf/conf.go`（`const Version`）、`CHANGELOG.md`、`README.md` 版本号一致
- `examples/` 下每个示例是独立的 `main` 包，不要破坏其编译
