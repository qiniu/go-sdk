#!/bin/bash
# pre-commit-check.sh — 在 git commit 前检查 gofmt 和 staticcheck
# 作为 Claude Code PreToolUse hook 使用

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')

# 仅拦截 git commit 命令
if ! echo "$COMMAND" | grep -qE '^git\s+commit\b'; then
  exit 0
fi

# 跳过 --amend 和 merge commit 场景
if echo "$COMMAND" | grep -qE '\-\-amend|\-\-allow-empty'; then
  exit 0
fi

ERRORS=""

# 检查 gofmt
UNFORMATTED=$(gofmt -s -l . 2>/dev/null | grep -v vendor/ | head -20)
if [ -n "$UNFORMATTED" ]; then
  ERRORS="${ERRORS}gofmt: 以下文件未格式化（运行 gofmt -s -w . 修复）:\n${UNFORMATTED}\n\n"
fi

# 检查 staticcheck
STATIC_OUTPUT=$(make staticcheck 2>&1)
STATIC_EXIT=$?
if [ $STATIC_EXIT -ne 0 ]; then
  ERRORS="${ERRORS}staticcheck: 静态检查未通过:\n$(echo "$STATIC_OUTPUT" | tail -20)\n\n"
fi

if [ -n "$ERRORS" ]; then
  REASON=$(printf "提交前检查未通过:\n\n%b请修复后再提交。" "$ERRORS")
  jq -n --arg reason "$REASON" '{
    hookSpecificOutput: {
      hookEventName: "PreToolUse",
      permissionDecision: "deny",
      permissionDecisionReason: $reason
    }
  }'
else
  exit 0
fi
