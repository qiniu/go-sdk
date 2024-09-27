// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 触发持久化数据处理命令
package pfop

import (
	"encoding/json"
	credentials "github.com/qiniu/go-sdk/v7/storagev2/credentials"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
)

// 调用 API 所用的请求
type Request struct {
	Credentials        credentials.CredentialsProvider // 鉴权参数，用于生成鉴权凭证，如果为空，则使用 HTTPClientOptions 中的 CredentialsProvider
	BucketName         string                          // 空间名称
	ObjectName         string                          // 对象名称
	Fops               string                          // 数据处理命令列表，以 `;` 分隔，可以指定多个数据处理命令
	NotifyUrl          string                          // 处理结果通知接收 URL
	Force              int64                           // 强制执行数据处理，设为 `1`，则可强制执行数据处理并覆盖原结果
	Type               int64                           // 任务类型，支持 `0` 表示普通任务，`1` 表示闲时任务
	Pipeline           string                          // 对列名，仅适用于普通任务
	WorkflowTemplateId string                          // 工作流模板 ID
}

// 获取 API 所用的响应
type Response struct {
	PersistentId string // 持久化数据处理任务 ID
}

// 返回的持久化数据处理任务 ID
type PfopId = Response
type jsonResponse struct {
	PersistentId string `json:"persistentId"` // 持久化数据处理任务 ID
}

func (j *Response) MarshalJSON() ([]byte, error) {
	if err := j.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(&jsonResponse{PersistentId: j.PersistentId})
}
func (j *Response) UnmarshalJSON(data []byte) error {
	var nj jsonResponse
	if err := json.Unmarshal(data, &nj); err != nil {
		return err
	}
	j.PersistentId = nj.PersistentId
	return nil
}
func (j *Response) validate() error {
	if j.PersistentId == "" {
		return errors.MissingRequiredFieldError{Name: "PersistentId"}
	}
	return nil
}
