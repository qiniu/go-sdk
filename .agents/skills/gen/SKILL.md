---
name: gen
description: 运行代码生成并检查变更。在修改 API 规范（api-specs/ YAML）或 sandbox OpenAPI/protobuf 后使用。
argument-hint: "[sandbox]"
allowed-tools:
  - Bash(make generate)
  - Bash(make generate-sandbox)
  - Bash(go build *)
  - Bash(git diff *)
  - Bash(git status)
---

# 代码生成

开始时声明："我正在使用 gen skill 运行代码生成。"

## 执行步骤

### 步骤 1：记录当前状态

```bash
git diff --stat
```

### 步骤 2：运行代码生成

默认运行 API 代码生成（storagev2、iam、media、audit）：

```bash
make generate
```

如果用户提到 sandbox，则运行：

```bash
make generate-sandbox
```

### 步骤 3：检查变更

```bash
git diff --stat
```

对比生成前后的变更，列出生成产生的文件差异。

### 步骤 4：验证编译

```bash
go build ./...
```

确认编译通过。

## 输出格式

列出生成的文件变更清单和编译结果。如果没有变更，说明"代码生成未产生新变更"。

## 系统规范（AGENTS.md）

执行本 skill 时同时遵守 `AGENTS.md` 中的系统规范；与本 skill 冲突时以本 skill 为准。
