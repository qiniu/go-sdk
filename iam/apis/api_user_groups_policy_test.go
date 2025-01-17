//go:build integration
// +build integration

package apis

import (
	"context"
	"os"
	"testing"

	"github.com/qiniu/go-sdk/v7/auth"
	createpolicy "github.com/qiniu/go-sdk/v7/iam/apis/create_policy"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/stretchr/testify/require"
)

const (
	kGroupAlias   = "GoIamGroup"
	kPolicyAlias  = "GoIamPolicy"
	kPolicyAlias2 = "GoIamPolicy2"
)

var (
	testAK = os.Getenv("accessKey")
	testSK = os.Getenv("secretKey")
)

func TestUserGroupsPolicyApi(t *testing.T) {

	ctx := context.Background()
	iamClient := NewIam(&httpclient.Options{
		Credentials: auth.New(testAK, testSK),
	})

	// 清理环境
	_, _ = iamClient.DeleteGroup(ctx, &DeleteGroupRequest{
		Alias: kGroupAlias,
	}, nil)

	_, _ = iamClient.DeletePolicy(ctx, &DeletePolicyRequest{
		Alias: kPolicyAlias,
	}, nil)

	_, _ = iamClient.DeletePolicy(ctx, &DeletePolicyRequest{
		Alias: kPolicyAlias2,
	}, nil)

	// 创建组
	createGroupResponse, err := iamClient.CreateGroup(ctx, &CreateGroupRequest{
		Alias:       kGroupAlias,
		Description: kGroupAlias + "desc",
	}, nil)
	require.NoError(t, err, "1. create group failed")

	require.NotNil(t, createGroupResponse, "1.1 create group response is nil")
	require.True(t, len(createGroupResponse.Data.Id) > 0, "1.2 create group response Id is nil")
	require.True(t, createGroupResponse.Data.RootUid > 0, "1.3 create group response RootUid is nil")
	require.True(t, len(createGroupResponse.Data.Alias) > 0, "1.4 create group response Alias is nil")
	require.True(t, len(createGroupResponse.Data.Description) > 0, "1.5 create group response Description is nil")
	require.True(t, len(createGroupResponse.Data.CreatedAt) > 0, "1.6 create group response CreatedAt is nil")
	require.True(t, len(createGroupResponse.Data.UpdatedAt) > 0, "1.7 create group response UpdatedAt is nil")

	// 创建策略
	policyDesc := kPolicyAlias + "Desc"
	policyAction := "cdn/DownloadCDNLog"
	policyEffect := "Allow"
	policyResource := "qrn:product:::/a/b/c.txt"
	createPolicyResponse, err := iamClient.CreatePolicy(ctx, &CreatePolicyRequest{
		Alias:       kPolicyAlias,
		Description: policyDesc,
		EditType:    1,
		Statement: []createpolicy.CreateStatement{
			{
				Actions:   []string{policyAction},
				Resources: []string{policyResource},
				Effect:    policyEffect,
			},
		},
	}, nil)
	require.NoError(t, err, "2. create policy failed")
	require.NotNil(t, createPolicyResponse, "2.1 create policy response is nil")
	require.True(t, len(createPolicyResponse.Data.Id) > 0, "2.2 create policy response Id is nil")
	require.True(t, createPolicyResponse.Data.RootUid > 0, "2.3 create policy response RootUid is nil")
	require.True(t, len(createPolicyResponse.Data.Alias) > 0, "2.4 create policy response Alias is nil")
	require.True(t, len(createPolicyResponse.Data.Description) > 0, "2.5 create policy response Description is nil")
	require.True(t, len(createPolicyResponse.Data.CreatedAt) > 0, "2.6 create policy response CreatedAt is nil")
	require.True(t, len(createPolicyResponse.Data.UpdatedAt) > 0, "2.7 create policy response UpdatedAt is nil")
	require.True(t, len(createPolicyResponse.Data.Statement) > 0, "2.8 create policy response Statement is nil")
	require.True(t, len(createPolicyResponse.Data.Statement[0].Actions) > 0, "2.9 create policy response Statement Actions is nil")
	require.True(t, len(createPolicyResponse.Data.Statement[0].Resources) > 0, "2.10 create policy response Statement Resources is nil")
	require.True(t, len(createPolicyResponse.Data.Statement[0].Effect) > 0, "2.11 create policy response Statement Effect is nil")

	// 创建策略 2
	policyDesc = kPolicyAlias2 + "Desc"
	createPolicyResponse, err = iamClient.CreatePolicy(ctx, &CreatePolicyRequest{
		Alias:       kPolicyAlias2,
		Description: policyDesc,
		EditType:    1,
		Statement: []createpolicy.CreateStatement{
			{
				Actions:   []string{policyAction},
				Resources: []string{policyResource},
				Effect:    policyEffect,
			},
		},
	}, nil)
	require.NoError(t, err, "3. create policy failed")
	require.NotNil(t, createPolicyResponse, "3.1 create policy response is nil")
	require.True(t, len(createPolicyResponse.Data.Id) > 0, "3.2 create policy response Id is nil")
	require.True(t, createPolicyResponse.Data.RootUid > 0, "3.3 create policy response RootUid is nil")
	require.True(t, len(createPolicyResponse.Data.Alias) > 0, "3.4 create policy response Alias is nil")
	require.True(t, len(createPolicyResponse.Data.Description) > 0, "3.5 create policy response Description is nil")
	require.True(t, len(createPolicyResponse.Data.CreatedAt) > 0, "3.6 create policy response CreatedAt is nil")
	require.True(t, len(createPolicyResponse.Data.UpdatedAt) > 0, "3.7 create policy response UpdatedAt is nil")
	require.True(t, len(createPolicyResponse.Data.Statement) > 0, "3.8 create policy response Statement is nil")
	require.True(t, len(createPolicyResponse.Data.Statement[0].Actions) > 0, "3.9 create policy response Statement Actions is nil")
	require.True(t, len(createPolicyResponse.Data.Statement[0].Resources) > 0, "3.10 create policy response Statement Resources is nil")
	require.True(t, len(createPolicyResponse.Data.Statement[0].Effect) > 0, "3.11 create policy response Statement Effect is nil")

	// 分组添加策略
	_, err = iamClient.ModifyGroupPolicies(ctx, &ModifyGroupPoliciesRequest{
		Alias:         kGroupAlias,
		PolicyAliases: []string{kPolicyAlias},
	}, nil)
	if err != nil {
		t.Error("4. modify group policies failed", err)
		return
	}

	// 获取分组策略信息
	getGroupPoliciesResponse, err := iamClient.GetGroupPolicies(ctx, &GetGroupPoliciesRequest{
		Alias: kGroupAlias,
	}, nil)
	require.NoError(t, err, "5. get group policies failed")
	require.NotNil(t, getGroupPoliciesResponse, "5. get group policies response is nil")
	require.True(t, getGroupPoliciesResponse.Data.Count == 1, "5.1 get group policies response Data.Count is invalid")
	require.True(t, len(getGroupPoliciesResponse.Data.List) == 1, "5.2 get group policies response Data.List is invalid")

	addPolicy := getGroupPoliciesResponse.Data.List[0]
	require.True(t, len(addPolicy.Id) > 0, "5.3 get group policies response Data.List[0].Id is invalid")
	require.True(t, addPolicy.Alias == kPolicyAlias, "5.3 get group policies response Data.List[0].Alias is invalid")
	require.True(t, len(addPolicy.Description) > 0, "5.4 get group policies response Data.List[0].Description is invalid")
	require.True(t, len(addPolicy.CreatedAt) > 0, "5.5 get group policies response Data.List[0].CreatedAt is invalid")
	require.True(t, len(addPolicy.UpdatedAt) > 0, "5.6 get group policies response Data.List[0].UpdatedAt is invalid")
	require.True(t, len(addPolicy.Statement) == 1, "5.7 get group policies response Data.List[0].Statement is invalid")
	require.True(t, len(addPolicy.Statement[0].Actions) == 1, "5.8 get group policies response Data.List[0].Statement.Actions is invalid")
	require.True(t, addPolicy.Statement[0].Actions[0] == policyAction, "5.9 get group policies response Data.List[0].Statement.Actions[0] is invalid")
	require.True(t, len(addPolicy.Statement[0].Resources) == 1, "5.10 get group policies response Data.List[0].Statement.Resources is invalid")
	require.True(t, addPolicy.Statement[0].Resources[0] == policyResource, "5.9 get group policies response Data.List[0].Statement.Resources[0] is invalid")
	require.True(t, addPolicy.Statement[0].Effect == policyEffect, "5.12 get group policies response Data.List[0].Statement[0].Effect is invalid")

	// 更新分组策略
	_, err = iamClient.ModifyGroupPolicies(ctx, &ModifyGroupPoliciesRequest{
		Alias:         kGroupAlias,
		PolicyAliases: []string{kGroupAlias, kPolicyAlias2},
	}, nil)
	require.NoError(t, err, "6. modify group policies failed")

	// 获取分组策略信息
	getGroupPoliciesResponse, err = iamClient.GetGroupPolicies(ctx, &GetGroupPoliciesRequest{
		Alias: kGroupAlias,
	}, nil)
	require.NoError(t, err, "7. get group policies failed")
	require.NotNil(t, getGroupPoliciesResponse, "7. get group policies response is nil")
	require.True(t, getGroupPoliciesResponse.Data.Count == 2, "7.1 get group policies response Data.Count is invalid")
	require.True(t, len(getGroupPoliciesResponse.Data.List) == 2, "7.2 get group policies response Data.List is invalid")

	for _, policy := range getGroupPoliciesResponse.Data.List {
		require.True(t, policy.Alias == kPolicyAlias || policy.Alias == kPolicyAlias2, "7.3 get group policies response Data.List is invalid")
	}

	// 删除分组策略
	_, err = iamClient.DeleteGroupPolicies(ctx, &DeleteGroupPoliciesRequest{
		Alias:         kGroupAlias,
		PolicyAliases: []string{kPolicyAlias},
	}, nil)
	require.NoError(t, err, "8. modify group policies failed")

	// 获取分组策略信息
	getGroupPoliciesResponse, err = iamClient.GetGroupPolicies(ctx, &GetGroupPoliciesRequest{
		Alias: kGroupAlias,
	}, nil)
	require.NoError(t, err, "9. get group policies failed")
	require.NotNil(t, getGroupPoliciesResponse, "9.1 get group policies response is nil")
	require.True(t, getGroupPoliciesResponse.Data.Count == 1, "9.2 get group policies response Data.List is invalid")
	require.True(t, getGroupPoliciesResponse.Data.List[0].Alias == kPolicyAlias2, "9.3 get group policies response Data.List is invalid")
}
