---
name: git-commit
description: 遵循 Angular 规范生成 git commit 消息。在用户请求创建 commit、写 commit message 或暂存并提交时使用。典型场景：改完代码要提交前、或要求按 Angular 规范写 message 时。
argument-hint: "<附加上下文(可选)> [push=true]"
allowed-tools:
  - Bash(git add:*)
  - Bash(git branch:*)
  - Bash(git commit:*)
  - Bash(git diff:*)
  - Bash(git log:*)
  - Bash(git status:*)
  - Bash(git push:*)
---

# 规范化提交

开始时声明："我正在使用 git-commit skill 生成并提交规范化 commit。"

## 快速开始

1. 查看 `status/diff/log`，确认提交范围与语言风格。
2. 暂存目标文件并展示"将提交列表"。
3. 生成符合 Angular 规范的 commit message。
4. 提交后如有上游分支则 push。

## 何时不要使用（Do NOT use）

- 工作区无待提交变更。
- 用户仅请求代码分析/评审，不需要提交。
- 用户明确要求仅生成建议，不执行 `git commit`。

## 触发样例与非样例

- 应触发：
- "帮我按 Angular 规范提交这些改动。"
- "写一个 commit message 并 push。"
- 不应触发：
- "review 这个 PR 有没有问题。"
- "分析下这个报错根因。"

## 使用示例

```bash
# 基本提交
git-commit 修复上传逻辑

# 提交并 push
git-commit push=true
```

## 预检查命令

```bash
git status --short
git diff HEAD
git branch --show-current
git log --oneline -10
```

## 消息规范

Header：`<type>(<scope>): <summary>`

Type：
- `feat` 新功能
- `fix` 缺陷修复
- `docs` 文档改动
- `refactor` 重构
- `perf` 性能优化
- `test` 测试相关
- `chore` 杂项维护
- `style` 代码格式调整

Scope 建议值（本项目常用）：
- `storage` / `storagev2` / `auth` / `cdn` / `pili` / `rtc` / `sms` / `linking` / `qvs`
- `iam` / `media` / `audit` / `sandbox`
- `client` / `conf` / `reqid`
- `internal` / `examples`

Summary 规则：祈使句、现在时、首字母小写、不加句号。

Body 规则：
- `docs` 可省略，其他类型建议必须有
- 至少说明"为什么改"与"影响范围"
- 非 `docs` 类型时，body 至少 20 个字符
- 单段 body 不超过 3 行，保持简洁

Footer（按需）：
- `BREAKING CHANGE: ...`
- `Fixes #123` / `Closes #456`

## 语言规则

- 参考最近 10 条提交语言。
- 近期中文为主则用中文；近期英文为主则用英文。

## 执行流程

1. 先确认提交文件列表。
2. 执行 `git add`（按文件名添加，避免 `git add -A`）。
3. 展示 commit message 草案并确认。
4. 执行 `git commit`。
5. 如有上游分支则 push：
   - `git push`
   - 或 `git push -u origin <current-branch>`

## 提交前检查提醒

提交代码前提醒用户确认已通过：
- `make unittest` — 单元测试
- `make staticcheck` — 静态检查
- `gofmt -s -w .` — 代码格式化

如果是生成代码相关修改，还需确认已运行 `make generate`。

## 关键约束

- 不要使用 emoji 在 commit message 中
- 不要在 commit message 中包含 AI 辅助工具相关信息
- 不要添加 "Generated with" 或 "Co-Authored-By: AI" 等内容
- 不要使用 `--no-verify` 跳过 hooks

## 输出模板

```text
已提交: <commit-hash>
标题: <type(scope): summary>
分支: <branch>
Push: 成功/失败/未执行
```

## 失败回退

- 工作区无变更：输出 `git status` 结果，提示无可提交内容。
- `git commit` 失败（hook 报错）：输出完整报错，建议修复后重试，不跳过 hook。
- `git push` 失败（远端冲突）：输出报错，建议先 pull/rebase 再重试。

## 验收标准（统一）

- 输入前提：参数与上下文可解析；缺省参数按 skill 默认值执行，并在输出中注明。
- 产出要求：按 skill 约定的输出模板给出结果，并包含关键证据（命令、路径、链接或日志摘要）。
- 通过判定：主流程步骤已完成且无阻塞；若有未完成项，必须明确标注影响范围与下一步。
- 默认策略（非交互）：需要确认但用户未及时响应时，采用"推荐默认值/最小风险项"继续；需要交互选择时优先推荐项。
- 阻塞升级：遇到权限、凭证、外部依赖缺失时立即停止该步骤，输出"阻塞点 + 已尝试 + 需要用户提供的信息"。

## 系统规范（AGENTS.md）

执行本 skill 时同时遵守 `AGENTS.md` 中的系统规范；与本 skill 冲突时以本 skill 为准。
