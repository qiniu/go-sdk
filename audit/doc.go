// Package audit 提供七牛云账号审计日志查询的 Go 客户端。
//
// 通过 audit/apis 子包的 [apis.NewAudit] 创建客户端，查询账号操作审计日志。
//
//	auditClient := apis.NewAudit(&http_client.Options{Credentials: cred})
//
// 主要操作：
//
//   - QueryLog: 查询账号审计日志，支持按时间范围、操作类型等条件过滤
package audit

//go:generate go run ../internal/api-generator -- --api-specs=../api-specs/audit --output=apis/ --struct-name=Audit --api-package=github.com/qiniu/go-sdk/v7/audit/apis
//go:generate go build ./apis/...
