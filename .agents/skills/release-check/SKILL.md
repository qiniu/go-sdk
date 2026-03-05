---
name: release-check
description: 检查发布版本号是否在三处保持一致（conf/conf.go、CHANGELOG.md、README.md）。在准备发布新版本时使用。
allowed-tools:
  - Read
  - Grep
  - Bash(grep *)
---

# 发布版本检查

开始时声明："我正在使用 release-check skill 检查版本号一致性。"

## 执行步骤

### 步骤 1：读取版本号

从 `conf/conf.go` 读取 `const Version = "x.y.z"`。

### 步骤 2：检查三处一致性

1. `conf/conf.go` — `const Version = "<version>"`
2. `CHANGELOG.md` — 包含 `## <version>` 条目
3. `README.md` — 包含 `require github.com/qiniu/go-sdk/v7 v<version>`

### 步骤 3：输出结果

输出版本号和三处的匹配状态表格：

```
| 位置 | 期望 | 状态 |
|------|------|------|
| conf/conf.go | vX.Y.Z | ✓ 匹配 / ✗ 不匹配 |
| CHANGELOG.md | ## X.Y.Z | ✓ 匹配 / ✗ 不匹配 |
| README.md | require ... vX.Y.Z | ✓ 匹配 / ✗ 不匹配 |
```

如果有不一致，给出具体的修复建议。

## 系统规范（AGENTS.md）

执行本 skill 时同时遵守 `AGENTS.md` 中的系统规范；与本 skill 冲突时以本 skill 为准。
