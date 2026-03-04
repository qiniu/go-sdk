// Package iam 提供七牛云身份与访问管理（IAM）的 Go 客户端。
//
// 通过 iam/apis 子包的 [apis.NewIam] 创建客户端，管理子账号用户、用户组和权限策略。
//
//	iamClient := apis.NewIam(&http_client.Options{Credentials: cred})
//
// # 用户管理
//
//   - CreateUser / GetUser / GetUsers / ModifyUser / DeleteUser
//   - CreateUserKeypairs / GetUserKeypairs / EnableUserKeypair / DisableUserKeypair
//
// # 用户组管理
//
//   - CreateGroup / GetGroup / GetGroups / ModifyGroup / DeleteGroup
//   - GetGroupUsers / ModifyGroupUsers
//
// # 策略管理
//
//   - CreatePolicy / GetPolicy / GetPolicies / ModifyPolicy / DeletePolicy
//   - GetUserPolicies / GetGroupPolicies / UpdatePolicyUsers / UpdatePolicyGroups
//
// 所有方法签名: Method(ctx, request, options) (response, error)。
package iam

//go:generate go run ../internal/api-generator -- --api-specs=../api-specs/iam --output=apis/ --struct-name=IAM --api-package=github.com/qiniu/go-sdk/v7/iam/apis
//go:generate go build ./apis/...
