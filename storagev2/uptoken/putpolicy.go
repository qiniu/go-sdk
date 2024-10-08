package uptoken

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// PutPolicy 存储上传策略
type PutPolicy map[string]interface{}

const (
	putPolicyKeyScope                        = "scope"
	putPolicyKeyDeadline                     = "deadline"
	putPolicyKeyIsPrefixalScope              = "isPrefixalScope"
	putPolicyKeyInsertOnly                   = "insertOnly"
	putPolicyKeyEndUser                      = "endUser"
	putPolicyKeyReturnUrl                    = "returnUrl"
	putPolicyKeyReturnBody                   = "returnBody"
	putPolicyKeyCallbackUrl                  = "callbackUrl"
	putPolicyKeyCallbackHost                 = "callbackHost"
	putPolicyKeyCallbackBody                 = "callbackBody"
	putPolicyKeyCallbackBodyType             = "callbackBodyType"
	putPolicyKeyPersistentOps                = "persistentOps"
	putPolicyKeyPersistentNotifyUrl          = "persistentNotifyUrl"
	putPolicyKeyPersistentPipeline           = "persistentPipeline"
	putPolicyKeyPersistentType               = "persistentType"
	putPolicyKeyPersistentWorkflowTemplateID = "persistentWorkflowTemplateID"
	putPolicyKeyForceSaveKey                 = "forceSaveKey"
	putPolicyKeySaveKey                      = "saveKey"
	putPolicyKeyFsizeMin                     = "fsizeMin"
	putPolicyKeyFsizeLimit                   = "fsizeLimit"
	putPolicyKeyDetectMime                   = "detectMime"
	putPolicyKeyMimeLimit                    = "mimeLimit"
	putPolicyKeyFileType                     = "fileType"
)

var (
	// ErrEmptyBucketName 空的 Bucket 名称
	ErrEmptyBucketName = errors.New("empty bucket name")

	// ErrInvalidPolicyValue 非法凭证值
	ErrInvalidPolicyValue = errors.New("invalid put policy value")
)

// FieldError indicates an error condition occurred while setting put policy
type FieldError struct {
	Err error
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("failed to set put policy: %v", e.Err)
}

// Unwrap returns the underlying error cause
func (e *FieldError) Unwrap() error {
	return e.Err
}

// NewPutPolicy 为指定的空间生成上传策略
func NewPutPolicy(bucket string, expiry time.Time) (PutPolicy, error) {
	return NewPutPolicyWithKey(bucket, "", expiry)
}

// NewPutPolicyWithKey 为指定的空间和对象名称生成上传策略
func NewPutPolicyWithKey(bucket, key string, expiry time.Time) (PutPolicy, error) {
	if bucket == "" {
		return nil, &FieldError{Err: ErrEmptyBucketName}
	}
	scope := bucket
	if key != "" {
		scope += ":" + key
	}
	return make(PutPolicy).SetScope(scope).SetDeadline(expiry.Unix()), nil
}

// NewPutPolicyWithKeyPrefix 为指定的空间和对象前缀生成上传策略
func NewPutPolicyWithKeyPrefix(bucket, keyPrefix string, expiry time.Time) (PutPolicy, error) {
	putPolicy, err := NewPutPolicyWithKey(bucket, keyPrefix, expiry)
	if err != nil {
		return nil, err
	}
	return putPolicy.SetIsPrefixalScope(1), nil
}

// GetScope 获取指定的上传的目标资源空间 Bucket 和资源键 Key
func (putPolicy PutPolicy) GetScope() (string, bool) {
	return putPolicy.getString(putPolicyKeyScope)
}

// SetScope 指定上传的目标资源空间 Bucket 和资源键 Key
func (putPolicy PutPolicy) SetScope(value string) PutPolicy {
	_ = putPolicy.Set(putPolicyKeyScope, value)
	return putPolicy
}

// GetDeadline 获取上传策略有效截止时间
func (putPolicy PutPolicy) GetDeadline() (int64, bool) {
	return putPolicy.getInt64(putPolicyKeyDeadline)
}

// SetDeadline 指定上传策略有效截止时间
func (putPolicy PutPolicy) SetDeadline(value int64) PutPolicy {
	_ = putPolicy.Set(putPolicyKeyDeadline, value)
	return putPolicy
}

// GetIsPrefixalScope 获取是否允许用户上传以 scope 的 keyPrefix 为前缀的文件
func (putPolicy PutPolicy) GetIsPrefixalScope() (int64, bool) {
	return putPolicy.getInt64(putPolicyKeyIsPrefixalScope)
}

// SetIsPrefixalScope 指定是否允许用户上传以 scope 的 keyPrefix 为前缀的文件
func (putPolicy PutPolicy) SetIsPrefixalScope(value int64) PutPolicy {
	_ = putPolicy.Set(putPolicyKeyIsPrefixalScope, value)
	return putPolicy
}

// GetInsertOnly 获取是否限定为新增语意
func (putPolicy PutPolicy) GetInsertOnly() (int64, bool) {
	return putPolicy.getInt64(putPolicyKeyInsertOnly)
}

// SetInsertOnly 指定是否限定为新增语意
func (putPolicy PutPolicy) SetInsertOnly(value int64) PutPolicy {
	_ = putPolicy.Set(putPolicyKeyInsertOnly, value)
	return putPolicy
}

// GetEndUser 获取唯一属主标识
func (putPolicy PutPolicy) GetEndUser() (string, bool) {
	return putPolicy.getString(putPolicyKeyEndUser)
}

// SetEndUser 指定唯一属主标识
func (putPolicy PutPolicy) SetEndUser(value string) PutPolicy {
	_ = putPolicy.Set(putPolicyKeyEndUser, value)
	return putPolicy
}

// GetReturnUrl 获取 Web 端文件上传成功后，浏览器执行 303 跳转的 URL
func (putPolicy PutPolicy) GetReturnUrl() (string, bool) {
	return putPolicy.getString(putPolicyKeyReturnUrl)
}

// SetReturnUrl 指定 Web 端文件上传成功后，浏览器执行 303 跳转的 URL
func (putPolicy PutPolicy) SetReturnUrl(value string) PutPolicy {
	_ = putPolicy.Set(putPolicyKeyReturnUrl, value)
	return putPolicy
}

// GetReturnBody 获取上传成功后，自定义七牛云最终返回给上传端的数据
func (putPolicy PutPolicy) GetReturnBody() (string, bool) {
	return putPolicy.getString(putPolicyKeyReturnBody)
}

// SetReturnBody 指定上传成功后，自定义七牛云最终返回给上传端的数据
func (putPolicy PutPolicy) SetReturnBody(value string) PutPolicy {
	_ = putPolicy.Set(putPolicyKeyReturnBody, value)
	return putPolicy
}

// GetCallbackUrl 获取上传成功后，七牛云向业务服务器发送 POST 请求的 URL
func (putPolicy PutPolicy) GetCallbackUrl() (string, bool) {
	return putPolicy.getString(putPolicyKeyCallbackUrl)
}

// SetCallbackUrl 指定上传成功后，七牛云向业务服务器发送 POST 请求的 URL
func (putPolicy PutPolicy) SetCallbackUrl(value string) PutPolicy {
	_ = putPolicy.Set(putPolicyKeyCallbackUrl, value)
	return putPolicy
}

// GetCallbackHost 获取上传成功后，七牛云向业务服务器发送回调通知时的 Host 值
func (putPolicy PutPolicy) GetCallbackHost() (string, bool) {
	return putPolicy.getString(putPolicyKeyCallbackHost)
}

// SetCallbackHost 指定上传成功后，七牛云向业务服务器发送回调通知时的 Host 值
func (putPolicy PutPolicy) SetCallbackHost(value string) PutPolicy {
	_ = putPolicy.Set(putPolicyKeyCallbackHost, value)
	return putPolicy
}

// GetCallbackBody 获取上传成功后，七牛云向业务服务器发送 Content-Type: application/x-www-form-urlencoded 的 POST 请求
func (putPolicy PutPolicy) GetCallbackBody() (string, bool) {
	return putPolicy.getString(putPolicyKeyCallbackBody)
}

// SetCallbackBody 指定上传成功后，七牛云向业务服务器发送 Content-Type: application/x-www-form-urlencoded 的 POST 请求
func (putPolicy PutPolicy) SetCallbackBody(value string) PutPolicy {
	_ = putPolicy.Set(putPolicyKeyCallbackBody, value)
	return putPolicy
}

// GetCallbackBodyType 获取上传成功后，七牛云向业务服务器发送回调通知 callbackBody 的 Content-Type
func (putPolicy PutPolicy) GetCallbackBodyType() (string, bool) {
	return putPolicy.getString(putPolicyKeyCallbackBodyType)
}

// SetCallbackBodyType 指定上传成功后，七牛云向业务服务器发送回调通知 callbackBody 的 Content-Type
func (putPolicy PutPolicy) SetCallbackBodyType(value string) PutPolicy {
	_ = putPolicy.Set(putPolicyKeyCallbackBodyType, value)
	return putPolicy
}

// GetPersistentOps 获取资源上传成功后触发执行的预转持久化处理指令列表
func (putPolicy PutPolicy) GetPersistentOps() (string, bool) {
	return putPolicy.getString(putPolicyKeyPersistentOps)
}

// SetPersistentOps 指定资源上传成功后触发执行的预转持久化处理指令列表
func (putPolicy PutPolicy) SetPersistentOps(value string) PutPolicy {
	_ = putPolicy.Set(putPolicyKeyPersistentOps, value)
	return putPolicy
}

// GetPersistentNotifyUrl 获取接收持久化处理结果通知的 URL
func (putPolicy PutPolicy) GetPersistentNotifyUrl() (string, bool) {
	return putPolicy.getString(putPolicyKeyPersistentNotifyUrl)
}

// SetPersistentNotifyUrl 指定接收持久化处理结果通知的 URL
func (putPolicy PutPolicy) SetPersistentNotifyUrl(value string) PutPolicy {
	_ = putPolicy.Set(putPolicyKeyPersistentNotifyUrl, value)
	return putPolicy
}

// GetPersistentPipeline 获取转码队列名
func (putPolicy PutPolicy) GetPersistentPipeline() (string, bool) {
	return putPolicy.getString(putPolicyKeyPersistentPipeline)
}

// SetPersistentPipeline 指定转码队列名
func (putPolicy PutPolicy) SetPersistentPipeline(value string) PutPolicy {
	_ = putPolicy.Set(putPolicyKeyPersistentPipeline, value)
	return putPolicy
}

// GetPersistentType 获取任务类型
func (putPolicy PutPolicy) GetPersistentType() (int64, bool) {
	return putPolicy.getInt64(putPolicyKeyPersistentType)
}

// SetPersistentType 指定任务类型
func (putPolicy PutPolicy) SetPersistentType(value int64) PutPolicy {
	_ = putPolicy.Set(putPolicyKeyPersistentType, value)
	return putPolicy
}

// GetPersistentWorkflowTemplateID 获取工作流模版 ID
func (putPolicy PutPolicy) GetPersistentWorkflowTemplateID() (string, bool) {
	return putPolicy.getString(putPolicyKeyPersistentWorkflowTemplateID)
}

// SetPersistentWorkflowTemplateID 指定工作流模版 ID
func (putPolicy PutPolicy) SetPersistentWorkflowTemplateID(value string) PutPolicy {
	_ = putPolicy.Set(putPolicyKeyPersistentWorkflowTemplateID, value)
	return putPolicy
}

// GetForceSaveKey 获取 saveKey 的优先级设置
func (putPolicy PutPolicy) GetForceSaveKey() (bool, bool) {
	return putPolicy.getBool(putPolicyKeyForceSaveKey)
}

// SetForceSaveKey 指定 saveKey 的优先级设置
func (putPolicy PutPolicy) SetForceSaveKey(value bool) PutPolicy {
	_ = putPolicy.Set(putPolicyKeyForceSaveKey, value)
	return putPolicy
}

// GetSaveKey 获取自定义资源名
func (putPolicy PutPolicy) GetSaveKey() (string, bool) {
	return putPolicy.getString(putPolicyKeySaveKey)
}

// SetSaveKey 指定自定义资源名
func (putPolicy PutPolicy) SetSaveKey(value string) PutPolicy {
	_ = putPolicy.Set(putPolicyKeySaveKey, value)
	return putPolicy
}

// GetFsizeMin 获取限定上传文件大小最小值
func (putPolicy PutPolicy) GetFsizeMin() (int64, bool) {
	return putPolicy.getInt64(putPolicyKeyFsizeMin)
}

// SetFsizeMin 指定限定上传文件大小最小值
func (putPolicy PutPolicy) SetFsizeMin(value int64) PutPolicy {
	_ = putPolicy.Set(putPolicyKeyFsizeMin, value)
	return putPolicy
}

// GetFsizeLimit 获取限定上传文件大小最大值
func (putPolicy PutPolicy) GetFsizeLimit() (int64, bool) {
	return putPolicy.getInt64(putPolicyKeyFsizeLimit)
}

// SetFsizeLimit 指定限定上传文件大小最大值
func (putPolicy PutPolicy) SetFsizeLimit(value int64) PutPolicy {
	_ = putPolicy.Set(putPolicyKeyFsizeLimit, value)
	return putPolicy
}

// GetDetectMime 获取开启 MimeType 侦测功能
func (putPolicy PutPolicy) GetDetectMime() (int64, bool) {
	return putPolicy.getInt64(putPolicyKeyDetectMime)
}

// SetDetectMime 指定开启 MimeType 侦测功能
func (putPolicy PutPolicy) SetDetectMime(value int64) PutPolicy {
	_ = putPolicy.Set(putPolicyKeyDetectMime, value)
	return putPolicy
}

// GetMimeLimit 获取限定用户上传的文件类型
func (putPolicy PutPolicy) GetMimeLimit() (string, bool) {
	return putPolicy.getString(putPolicyKeyMimeLimit)
}

// SetMimeLimit 指定限定用户上传的文件类型
func (putPolicy PutPolicy) SetMimeLimit(value string) PutPolicy {
	_ = putPolicy.Set(putPolicyKeyMimeLimit, value)
	return putPolicy
}

// GetFileType 获取文件存储类型
func (putPolicy PutPolicy) GetFileType() (int64, bool) {
	return putPolicy.getInt64(putPolicyKeyFileType)
}

// SetFileType 指定文件存储类型
func (putPolicy PutPolicy) SetFileType(value int64) PutPolicy {
	_ = putPolicy.Set(putPolicyKeyFileType, value)
	return putPolicy
}

// Get 获取上传策略的值
func (putPolicy PutPolicy) Get(key string) (value interface{}, ok bool) {
	value, ok = putPolicy[key]
	return
}

func (putPolicy PutPolicy) getString(key string) (string, bool) {
	v, ok := putPolicy[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func (putPolicy PutPolicy) getBool(key string) (bool, bool) {
	v, ok := putPolicy[key]
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

func (putPolicy PutPolicy) getInt64(key string) (int64, bool) {
	if v, ok := putPolicy[key]; ok {
		switch i := v.(type) {
		case int64:
			return i, true
		case float64:
			return int64(i), true
		case float32:
			return int64(i), true
		case int32:
			return int64(i), true
		case int16:
			return int64(i), true
		case int8:
			return int64(i), true
		case int:
			return int64(i), true
		case uint64:
			return int64(i), true
		case uint32:
			return int64(i), true
		case uint16:
			return int64(i), true
		case uint8:
			return int64(i), true
		case uint:
			return int64(i), true
		case json.Number:
			i64, err := i.Int64()
			return i64, err == nil
		}
	}
	return 0, false
}

// Set 设置上传策略的值
//
// 如果 value 为 nil，则返回 ErrInvalidPolicyValue 错误。
func (putPolicy PutPolicy) Set(key string, value interface{}) error {
	if value == nil {
		return &FieldError{Err: ErrInvalidPolicyValue}
	}
	putPolicy[key] = value
	return nil
}

// Delete 删除上传策略的值
func (putPolicy PutPolicy) Delete(key string) (value interface{}, ok bool) {
	value, ok = putPolicy[key]
	delete(putPolicy, key)
	return
}

// GetBucketName 获取上传策略内的空间名称
//
// 该方法会从上传策略中解析出空间名称，如果上传策略中没有 scope 字段，则返回 ErrEmptyBucketName 错误，如果上传策略的 scope 字段格式有误，则返回 ErrInvalidPolicyValue 错误。
func (putPolicy PutPolicy) GetBucketName() (string, error) {
	if scope, ok := putPolicy.GetScope(); !ok {
		return "", ErrInvalidPolicyValue
	} else {
		fields := strings.SplitN(scope, ":", 2)
		if len(fields) == 0 {
			return "", ErrEmptyBucketName
		}
		return fields[0], nil
	}
}
