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
)

const (
	kGroupAlias   = "GoIamGroup"
	kPolicyAlias  = "GoIamPolicy"
	kPolicyAlias2 = "GoIamPolicy2"
)

var (
	testAK     = os.Getenv("accessKey")
	testSK     = os.Getenv("secretKey")
	testBucket = os.Getenv("QINIU_TEST_BUCKET")
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
	if err != nil {
		t.Error("1. create group failed", err)
		return
	}
	if createGroupResponse == nil {
		t.Error("1.1 create group response is nil")
		return
	}
	if len(createGroupResponse.Data.Id) == 0 {
		t.Error("1.2 create group response Id is nil")
	}
	if createGroupResponse.Data.RootUid == 0 {
		t.Error("1.3 create group response RootUid is nil")
	}
	if len(createGroupResponse.Data.Alias) == 0 {
		t.Error("1.4 create group response Alias is nil")
	}
	if len(createGroupResponse.Data.Description) == 0 {
		t.Error("1.5 create group response Description is nil")
	}
	if len(createGroupResponse.Data.CreatedAt) == 0 {
		t.Error("1.6 create group response CreatedAt is nil")
	}
	if len(createGroupResponse.Data.UpdatedAt) == 0 {
		t.Error("1.7 create group response UpdatedAt is nil")
	}

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
	if err != nil {
		t.Error("2. create policy failed", err)
		return
	}
	if createPolicyResponse == nil {
		t.Error("2.1 create policy response is nil")
		return
	}
	if len(createPolicyResponse.Data.Id) == 0 {
		t.Error("2.2 create policy response Id is nil")
	}
	if createPolicyResponse.Data.RootUid == 0 {
		t.Error("2.3 create policy response RootUid is nil")
	}
	if len(createPolicyResponse.Data.Alias) == 0 {
		t.Error("2.4 create policy response Alias is nil")
	}
	if len(createPolicyResponse.Data.Description) == 0 {
		t.Error("2.5 create policy response Description is nil")
	}
	if len(createPolicyResponse.Data.CreatedAt) == 0 {
		t.Error("2.6 create policy response CreatedAt is nil")
	}
	if len(createPolicyResponse.Data.UpdatedAt) == 0 {
		t.Error("2.7 create policy response UpdatedAt is nil")
	}
	if len(createPolicyResponse.Data.Statement) == 0 {
		t.Error("2.8 create policy response Statement is nil")
	}
	if len(createPolicyResponse.Data.Statement[0].Actions) == 0 {
		t.Error("2.9 create policy response Statement Actions is nil")
	}
	if len(createPolicyResponse.Data.Statement[0].Resources) == 0 {
		t.Error("2.10 create policy response Statement Resources is nil")
	}
	if len(createPolicyResponse.Data.Statement[0].Effect) == 0 {
		t.Error("2.11 create policy response Statement Effect is nil")
	}

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
	if err != nil {
		t.Error("3. create policy failed")
		return
	}
	if createPolicyResponse == nil {
		t.Error("3.1 create policy response is nil")
		return
	}
	if len(createPolicyResponse.Data.Id) == 0 {
		t.Error("3.2 create policy response Id is nil")
	}
	if createPolicyResponse.Data.RootUid == 0 {
		t.Error("3.3 create policy response RootUid is nil")
	}
	if len(createPolicyResponse.Data.Alias) == 0 {
		t.Error("3.4 create policy response Alias is nil")
	}
	if len(createPolicyResponse.Data.Description) == 0 {
		t.Error("3.5 create policy response Description is nil")
	}
	if len(createPolicyResponse.Data.CreatedAt) == 0 {
		t.Error("3.6 create policy response CreatedAt is nil")
	}
	if len(createPolicyResponse.Data.UpdatedAt) == 0 {
		t.Error("3.7 create policy response UpdatedAt is nil")
	}
	if len(createPolicyResponse.Data.Statement) == 0 {
		t.Error("3.8 create policy response Statement is nil")
	}
	if len(createPolicyResponse.Data.Statement[0].Actions) == 0 {
		t.Error("3.9 create policy response Statement Actions is nil")
	}
	if len(createPolicyResponse.Data.Statement[0].Resources) == 0 {
		t.Error("3.10 create policy response Statement Resources is nil")
	}
	if len(createPolicyResponse.Data.Statement[0].Effect) == 0 {
		t.Error("3.11 create policy response Statement Effect is nil")
	}

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
	if err != nil {
		t.Error("5. get group policies failed")
		return
	}
	if getGroupPoliciesResponse == nil {
		t.Error("5.1 get group policies response is nil")
		return
	}
	if getGroupPoliciesResponse.Data.Count != 1 {
		t.Error("5.2 get group policies response Data.Count is invalid")
	}
	if len(getGroupPoliciesResponse.Data.List) != 1 {
		t.Error("5.2 get group policies response Data.List is invalid")
	}
	addPolicy := getGroupPoliciesResponse.Data.List[0]
	if len(addPolicy.Id) == 0 {
		t.Error("5.3 get group policies response Data.List[0].Id is invalid")
	}
	if addPolicy.Alias != kPolicyAlias {
		t.Error("5.4 get group policies response Data.List[0].Alias is invalid")
	}
	if len(addPolicy.Description) == 0 {
		t.Error("5.5 get group policies response Data.List[0].Description is invalid")
	}
	if len(addPolicy.CreatedAt) == 0 {
		t.Error("5.6 get group policies response Data.List[0].CreatedAt is invalid")
	}
	if len(addPolicy.UpdatedAt) == 0 {
		t.Error("5.7 get group policies response Data.List[0].UpdatedAt is invalid")
	}
	if len(addPolicy.Statement) != 1 {
		t.Error("5.8 get group policies response Data.List[0].Statement is invalid")
	}
	if len(addPolicy.Statement[0].Actions) != 1 {
		t.Error("5.9 get group policies response Data.List[0].Statement[0].Actions is invalid")
	}
	if addPolicy.Statement[0].Actions[0] != policyAction {
		t.Error("5.10 get group policies response Data.List[0].Statement[0].Actions is invalid")
	}
	if len(addPolicy.Statement[0].Resources) != 1 {
		t.Error("5.11 get group policies response Data.List[0].Statement[0].Resources is invalid")
	}
	if addPolicy.Statement[0].Resources[0] != policyResource {
		t.Error("5.12 get group policies response Data.List[0].Statement[0].Resources is invalid")
	}
	if addPolicy.Statement[0].Effect != policyEffect {
		t.Error("5.13 get group policies response Data.List[0].Statement[0].Effect is invalid")
	}

	// 更新分组策略
	_, err = iamClient.ModifyGroupPolicies(ctx, &ModifyGroupPoliciesRequest{
		Alias:         kGroupAlias,
		PolicyAliases: []string{kGroupAlias, kPolicyAlias2},
	}, nil)
	if err != nil {
		t.Error("6. modify group policies failed")
		return
	}

	// 获取分组策略信息
	getGroupPoliciesResponse, err = iamClient.GetGroupPolicies(ctx, &GetGroupPoliciesRequest{
		Alias: kGroupAlias,
	}, nil)
	if err != nil {
		t.Error("7. get group policies failed")
		return
	}
	if getGroupPoliciesResponse == nil {
		t.Error("7.1 get group policies response is nil")
		return
	}
	if getGroupPoliciesResponse.Data.Count != 2 {
		t.Error("7.2 get group policies response Data.Count is invalid")
	}
	if len(getGroupPoliciesResponse.Data.List) != 2 {
		t.Error("7.3 get group policies response Data.List is invalid")
	}
	for _, policy := range getGroupPoliciesResponse.Data.List {
		if policy.Alias == kPolicyAlias {
			continue
		}
		if policy.Alias == kPolicyAlias2 {
			continue
		}
		t.Error("7.4 get group policies response Data.List is invalid")
	}

	// 删除分组策略
	_, err = iamClient.DeleteGroupPolicies(ctx, &DeleteGroupPoliciesRequest{
		Alias:         kGroupAlias,
		PolicyAliases: []string{kPolicyAlias},
	}, nil)
	if err != nil {
		t.Error("8. delete group policies failed")
		return
	}

	// 获取分组策略信息
	getGroupPoliciesResponse, err = iamClient.GetGroupPolicies(ctx, &GetGroupPoliciesRequest{
		Alias: kGroupAlias,
	}, nil)
	if err != nil {
		t.Error("9. get group policies failed")
		return
	}
	if getGroupPoliciesResponse == nil {
		t.Error("9.1 get group policies response is nil")
		return
	}
	if len(getGroupPoliciesResponse.Data.List) != 1 {
		t.Error("9.2 get group policies response Data.List is invalid")
	}
	if getGroupPoliciesResponse.Data.List[0].Alias != kPolicyAlias2 {
		t.Error("9.3 get group policies response Data.List is invalid")
	}
}
