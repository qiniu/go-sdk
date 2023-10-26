package uptoken

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// PutPolicy 存储上传策略
type PutPolicy map[string]interface{}

const (
	// PutPolicyKeyScope 指定上传的目标资源空间 Bucket 和资源键 Key
	PutPolicyKeyScope = "scope"
	// PutPolicyKeyDeadline 上传策略有效截止时间
	PutPolicyKeyDeadline = "deadline"
	// PutPolicyKeyIsPrefixalScope 是否允许用户上传以 scope 的 keyPrefix 为前缀的文件
	PutPolicyKeyIsPrefixalScope = "isPrefixalScope"
	// PutPolicyKeyInsertOnly 限定为新增语意
	PutPolicyKeyInsertOnly = "insertOnly"
	// PutPolicyKeyEndUser 唯一属主标识
	PutPolicyKeyEndUser = "endUser"
	// PutPolicyKeyReturnUrl Web 端文件上传成功后，浏览器执行 303 跳转的 URL
	PutPolicyKeyReturnUrl = "returnUrl"
	// PutPolicyKeyReturnBody 上传成功后，自定义七牛云最终返回给上传端的数据
	PutPolicyKeyReturnBody = "returnBody"
	// PutPolicyKeyCallbackUrl 上传成功后，七牛云向业务服务器发送 POST 请求的 URL
	PutPolicyKeyCallbackUrl = "callbackUrl"
	// PutPolicyKeyCallbackHost 上传成功后，七牛云向业务服务器发送回调通知时的 Host 值
	PutPolicyKeyCallbackHost = "callbackHost"
	// PutPolicyKeyCallbackBody 上传成功后，七牛云向业务服务器发送 Content-Type: application/x-www-form-urlencoded 的 POST 请求
	PutPolicyKeyCallbackBody = "callbackBody"
	// PutPolicyKeyCallbackBodyType 上传成功后，七牛云向业务服务器发送回调通知 callbackBody 的 Content-Type
	PutPolicyKeyCallbackBodyType = "callbackBodyType"
	// PutPolicyKeyPersistentOps 资源上传成功后触发执行的预转持久化处理指令列表
	PutPolicyKeyPersistentOps = "persistentOps"
	// PutPolicyKeyPersistentNotifyUrl 接收持久化处理结果通知的 URL
	PutPolicyKeyPersistentNotifyUrl = "persistentNotifyUrl"
	// PutPolicyKeyPersistentPipeline 转码队列名
	PutPolicyKeyPersistentPipeline = "persistentPipeline"
	// PutPolicyKeyForceSaveKey saveKey 的优先级设置
	PutPolicyKeyForceSaveKey = "forceSaveKey"
	// PutPolicyKeySaveKey 自定义资源名
	PutPolicyKeySaveKey = "saveKey"
	// PutPolicyKeyFsizeMin 限定上传文件大小最小值
	PutPolicyKeyFsizeMin = "fsizeMin"
	// PutPolicyKeyFsizeLimit 限定上传文件大小最大值
	PutPolicyKeyFsizeLimit = "fsizeLimit"
	// PutPolicyKeyDetectMime 开启 MimeType 侦测功能
	PutPolicyKeyDetectMime = "detectMime"
	// PutPolicyKeyMimeLimit 限定用户上传的文件类型
	PutPolicyKeyMimeLimit = "mimeLimit"
	// PutPolicyKeyFileType 文件存储类型
	PutPolicyKeyFileType = "fileType"
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
	putPolicy := make(PutPolicy)
	putPolicy.Set(PutPolicyKeyScope, scope)
	putPolicy.Set(PutPolicyKeyDeadline, expiry.Unix())
	return putPolicy, nil
}

// NewPutPolicyWithKeyPrefix 为指定的空间和对象前缀生成上传策略
func NewPutPolicyWithKeyPrefix(bucket, keyPrefix string, expiry time.Time) (PutPolicy, error) {
	putPolicy, err := NewPutPolicyWithKey(bucket, keyPrefix, expiry)
	if err != nil {
		return nil, err
	}
	putPolicy.Set(PutPolicyKeyIsPrefixalScope, 1)
	return putPolicy, nil
}

// Get 获取上传策略的值
func (putPolicy PutPolicy) Get(key string) (value interface{}, ok bool) {
	value, ok = putPolicy[key]
	return
}

// GetString 获取上传策略的值作为字符串
func (putPolicy PutPolicy) GetString(key string) (string, bool) {
	v, ok := putPolicy[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// GetBool 获取上传策略的值作为布尔值
func (putPolicy PutPolicy) GetBool(key string) (bool, bool) {
	v, ok := putPolicy[key]
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

// GetInt64 获取上传策略的值作为整型
func (putPolicy PutPolicy) GetInt64(key string) (int64, bool) {
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
		}
	}
	return 0, false
}

// Set 设置上传策略的值
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
func (putPolicy PutPolicy) GetBucketName() (string, error) {
	if scope, ok := putPolicy.GetString(PutPolicyKeyScope); !ok {
		return "", ErrInvalidPolicyValue
	} else {
		fields := strings.SplitN(scope, ":", 2)
		if len(fields) == 0 {
			return "", ErrEmptyBucketName
		}
		return fields[0], nil
	}
}
