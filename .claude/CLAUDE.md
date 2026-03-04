# Qiniu Go SDK — Claude Code Instructions

## 项目概要

七牛云 Go SDK，模块路径 `github.com/qiniu/go-sdk/v7`，Go 1.22+，MIT 许可。

## 核心开发规范

@AGENTS.md

## Claude Code 特定指引

### 常用命令

- `make unittest` — 运行单元测试（提交前必须通过）
- `make staticcheck` — 运行静态检查（提交前必须通过）
- `make generate` — 运行代码生成（修改 API 规范后执行）
- `make generate-sandbox` — 运行 sandbox 代码生成
- `gofmt -s -w .` — 格式化代码

### 开发注意事项

- 修改代码前先运行 `make unittest` 确认当前状态
- 不要修改 `DO NOT EDIT DIRECTLY` 标记的生成代码
- 注释默认使用中文
- 测试文件需添加 `//go:build unit` 或 `//go:build integration` 标签
- `api-specs/` 是 git submodule，使用 `git submodule update --init --recursive` 初始化
