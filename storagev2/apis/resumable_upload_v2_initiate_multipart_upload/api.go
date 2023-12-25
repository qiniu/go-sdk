// THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!

// 使用 Multipart Upload 方式上传数据前，必须先调用 API 来获取一个全局唯一的 UploadId，后续的块数据通过 uploadPart API 上传，整个文件完成 completeMultipartUpload API，已经上传块的删除 abortMultipartUpload API 都依赖该 UploadId
package resumable_upload_v2_initiate_multipart_upload

import (
	"encoding/json"
	errors "github.com/qiniu/go-sdk/v7/storagev2/errors"
	uptoken "github.com/qiniu/go-sdk/v7/storagev2/uptoken"
)

// 调用 API 所用的请求
type Request struct {
	BucketName string           // 存储空间名称
	ObjectName *string          // 对象名称
	UpToken    uptoken.Provider // 上传凭证，如果为空，则使用 HTTPClientOptions 中的 UpToken
}

// 获取 API 所用的响应
type Response struct {
	UploadId  string // 初始化文件生成的 id
	ExpiredAt int64  // UploadId 的过期时间 UNIX 时间戳，过期之后 UploadId 不可用
}

// 返回本次 MultipartUpload 相关信息
type NewMultipartUpload = Response
type jsonResponse struct {
	UploadId  string `json:"uploadId"` // 初始化文件生成的 id
	ExpiredAt int64  `json:"expireAt"` // UploadId 的过期时间 UNIX 时间戳，过期之后 UploadId 不可用
}

func (j *Response) MarshalJSON() ([]byte, error) {
	if err := j.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(&jsonResponse{UploadId: j.UploadId, ExpiredAt: j.ExpiredAt})
}
func (j *Response) UnmarshalJSON(data []byte) error {
	var nj jsonResponse
	if err := json.Unmarshal(data, &nj); err != nil {
		return err
	}
	j.UploadId = nj.UploadId
	j.ExpiredAt = nj.ExpiredAt
	return nil
}
func (j *Response) validate() error {
	if j.UploadId == "" {
		return errors.MissingRequiredFieldError{Name: "UploadId"}
	}
	if j.ExpiredAt == 0 {
		return errors.MissingRequiredFieldError{Name: "ExpiredAt"}
	}
	return nil
}
