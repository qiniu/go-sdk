---
name: check
description: 运行提交前检查：代码格式化验证、静态检查和单元测试。在准备提交代码、检查代码质量或验证修改是否通过时使用。
allowed-tools:
  - Bash(gofmt *)
  - Bash(make staticcheck)
  - Bash(make unittest)
---

# 提交前检查

开始时声明："我正在使用 check skill 运行提交前检查。"

## 执行步骤

### 步骤 1：代码格式化检查

```bash
gofmt -s -l .
```

如果有未格式化的文件，列出文件清单并提示运行 `gofmt -s -w .` 修复。

### 步骤 2：静态检查

```bash
make staticcheck
```

如果失败，列出 staticcheck 报告的问题并给出修复建议。

### 步骤 3：单元测试

```bash
make unittest
```

如果任何步骤失败，停止并报告错误，给出修复建议。

## 输出格式

全部通过时：

```
✓ gofmt — 通过
✓ staticcheck — 通过
✓ unittest — 通过
```

有失败时列出具体错误和修复建议。

## 系统规范（AGENTS.md）

执行本 skill 时同时遵守 `AGENTS.md` 中的系统规范；与本 skill 冲突时以本 skill 为准。
