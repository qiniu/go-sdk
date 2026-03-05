---
name: gh-view
description: 分析 GitHub issues、PRs 与 discussions，提供见解或实施指导。在用户提供 GitHub URL、issue/PR 编号或询问仓库内容时使用。典型场景：看 issue/PR 状态、拉 diff 与 checks、要结论与建议下一步时。
argument-hint: "<GitHub URL | owner/repo | issue/PR编号> [repo=qiniu/go-sdk]"
allowed-tools:
  - Bash(gh repo view:*)
  - Bash(gh issue view:*)
  - Bash(gh pr view:*)
  - Bash(gh pr diff:*)
  - Bash(gh pr checks:*)
  - Bash(gh api *)
---

# GitHub 内容分析

开始时声明："我正在使用 gh-view skill 分析 GitHub 内容。"

## 快速开始

1. 识别输入类型（repo / issue / pr / discussion / release）。
2. 用 `gh` 拉取基础信息、评论、状态、代码差异。
3. 输出结论：背景、关键点、风险、建议下一步。
4. 默认只分析；仅在用户明确要求时执行写操作。

## 何时不要使用（Do NOT use）

- 用户请求直接修改代码、提交 commit。
- 用户主要目标是 CI 根因分析（优先 `ci-analyze`）。

## 触发样例与非样例

- 应触发：
- "看下这个 PR 的核心风险点：<GitHub URL>。"
- "帮我总结这个 issue 的讨论重点。"
- 不应触发：
- "帮我把这个 PR 的代码直接改完并提交。"
- "这个 pipeline fail 了，帮我查日志根因。"

## 使用示例

```bash
# 分析 PR
gh-view https://github.com/qiniu/go-sdk/pull/123

# 分析 issue（使用当前仓库上下文）
gh-view #456

# 查看仓库概览
gh-view qiniu/go-sdk
```

## 输入识别规则

- 仓库：`owner/repo` 或 `https://github.com/owner/repo`
- Issue：`owner/repo/issues/<n>` 或 `#<n>`（需 repo 上下文）
- PR：`owner/repo/pull/<n>` 或 `#<n>`（需 repo 上下文）
- 跨仓库编号：`owner/repo#<n>`

补充规则：
- 当输入是完整 URL 时，先提取 `owner/repo` 与对象编号，再执行 `gh` 命令。
- 用户未显式给 `repo` 时，默认使用 `qiniu/go-sdk`。
- 需要结构化信息时优先添加 `--json`。

## 命令矩阵（按对象）

```bash
# 仓库
gh repo view <owner/repo>
gh repo view <owner/repo> --json name,description,defaultBranchRef,stargazerCount,forkCount,openIssuesCount

# Issue
gh issue view <n> --repo <owner/repo>
gh issue view <n> --repo <owner/repo> --comments

# PR
gh pr view <n> --repo <owner/repo>
gh pr view <n> --repo <owner/repo> --comments
gh pr diff <n> --repo <owner/repo>
gh pr checks <n> --repo <owner/repo>
```

## 输出模板

```text
目标: <repo/issue/pr>
背景: <一句话>

关键信息:
- ...
- ...

风险/阻塞:
- ...

建议下一步:
1. ...
2. ...
3. ...
```

## 边界

- 仅使用 `gh repo view` / `gh issue view` / `gh pr view` / `gh pr diff` / `gh pr checks` / `gh api` 等只读命令。
- 不执行任何写操作（如 `gh pr comment`、`gh pr merge`、`gh issue comment`、打标签、关闭 issue）；未明确授权时仅分析不写入。
- 信息不足时先问澄清，不猜测。

## 失败回退

- URL 解析失败：提示用户检查 URL 格式，给出合法示例。
- `gh` 命令 403/404：检查仓库权限或对象是否存在；输出实际命令和报错。
- 信息不足以给出结论：明确标注"信息不足"，列出已获取内容和缺失项。

## 验收标准（统一）

- 输入前提：参数与上下文可解析；缺省参数按 skill 默认值执行，并在输出中注明。
- 产出要求：按 skill 约定的输出模板给出结果，并包含关键证据（命令、路径、链接或日志摘要）。
- 通过判定：主流程步骤已完成且无阻塞；若有未完成项，必须明确标注影响范围与下一步。
- 默认策略（非交互）：需要确认但用户未及时响应时，采用"推荐默认值/最小风险项"继续；需要交互选择时优先推荐项。
- 阻塞升级：遇到权限、凭证、外部依赖缺失时立即停止该步骤，输出"阻塞点 + 已尝试 + 需要用户提供的信息"。

## 系统规范（AGENTS.md）

执行本 skill 时同时遵守 `AGENTS.md` 中的系统规范；与本 skill 冲突时以本 skill 为准。
