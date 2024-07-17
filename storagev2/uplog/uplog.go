package uplog

import (
	"context"
	"time"

	"github.com/qiniu/go-sdk/v7/internal/uplog"
)

// UpApiVersion 表示上传接口的版本
type UpApiVersion = uplog.UpApiVersion

// UpType 表示上传类型
type UpType = uplog.UpType

const (
	// UpApiVersionV1 表示上传接口的版本 1
	UpApiVersionV1 = uplog.UpApiVersionV1
	// UpApiVersionV1 表示上传接口的版本 2
	UpApiVersionV2 = uplog.UpApiVersionV2
	// UpTypeForm 表示表单上传
	UpTypeForm = uplog.UpTypeForm
	// UpTypeResumableV1 表示分片上传 V1
	UpTypeResumableV1 = uplog.UpTypeResumableV1
	// UpTypeResumableV2 表示分片上传 V2
	UpTypeResumableV2 = uplog.UpTypeResumableV2
)

// WithBlock 用于记录一轮分片上传的日志
func WithBlock(ctx context.Context, upApiVersion UpApiVersion, fileSize, recoveredFrom uint64, targetBucket, targetKey, upToken string, handle func(context.Context) error) error {
	return uplog.WithBlock(ctx, upApiVersion, fileSize, recoveredFrom, targetBucket, targetKey, upToken, handle)
}

// WithQuality 用于记录上传质量日志
func WithQuality(ctx context.Context, upType UpType, fileSize uint64, targetBucket, targetKey, upToken string, handle func(ctx context.Context, switchRegion func()) error) error {
	return uplog.WithQuality(ctx, upType, fileSize, targetBucket, targetKey, upToken, handle)
}

// DisableUplog 禁止日志功能
func DisableUplog() {
	uplog.DisableUplog()
}

// EnableUplog 启用日志功能
func EnableUplog() {
	uplog.EnableUplog()
}

// IsUplogEnabled 判断日志功能是否启用
func IsUplogEnabled() bool {
	return uplog.IsUplogEnabled()
}

// GetUplogMaxStorageBytes 获取日志最大存储容量
func GetUplogMaxStorageBytes() uint64 {
	return uplog.GetUplogMaxStorageBytes()
}

// SetUplogMaxStorageBytes 设置日志最大存储容量
func SetUplogMaxStorageBytes(max uint64) {
	uplog.SetUplogMaxStorageBytes(max)
}

// SetUplogFileBufferDirPath 设置日志文件缓存目录
func SetUplogFileBufferDirPath(path string) {
	uplog.SetUplogFileBufferDirPath(path)
}

// SetFlushFileBufferInterval 设置日志文件缓存刷新间隔
func SetFlushFileBufferInterval(d time.Duration) {
	uplog.SetWriteFileBufferInterval(d)
}

// GetWriteFileBufferInterval() 获取日志文件缓存刷新间隔
func GetWriteFileBufferInterval() time.Duration {
	return uplog.GetWriteFileBufferInterval()
}

// FlushBuffer 刷新日志缓存
func FlushBuffer() error {
	return uplog.FlushBuffer()
}

// GetUplogUrl 获取日志上传 URL
func GetUplogUrl() string {
	return uplog.GetUplogUrl()
}

// SetUplogUrl 设置日志上传 URL
func SetUplogUrl(url string) {
	uplog.SetUplogUrl(url)
}
