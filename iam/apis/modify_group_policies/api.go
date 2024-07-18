// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 添加授权策略到用户分组
package modify_group_policies

import (
	"encoding/json"
	credentials "github.com/qiniu/go-sdk/v7/storagev2/credentials"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
)

// 调用 API 所用的请求
type Request struct {
	Alias         string                          // 用户分组别名
	Credentials   credentials.CredentialsProvider // 鉴权参数，用于生成鉴权凭证，如果为空，则使用 HTTPClientOptions 中的 CredentialsProvider
	PolicyAliases PolicyAliases                   // 授权策略别名集合
}

// 添加给用户分组的授权策略别名集合
type PolicyAliases = []string

// 为用户分组添加授权策略参数
type AddedGroupIamPoliciesParam = Request
type jsonRequest struct {
	PolicyAliases PolicyAliases `json:"policy_aliases"` // 授权策略别名集合
}

func (j *Request) MarshalJSON() ([]byte, error) {
	if err := j.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(&jsonRequest{PolicyAliases: j.PolicyAliases})
}
func (j *Request) UnmarshalJSON(data []byte) error {
	var nj jsonRequest
	if err := json.Unmarshal(data, &nj); err != nil {
		return err
	}
	j.PolicyAliases = nj.PolicyAliases
	return nil
}
func (j *Request) validate() error {
	if len(j.PolicyAliases) == 0 {
		return errors.MissingRequiredFieldError{Name: "PolicyAliases"}
	}
	return nil
}

// 获取 API 所用的响应
type Response struct{}