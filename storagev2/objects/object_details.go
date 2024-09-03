package objects

import (
	"crypto/md5"
	"time"
)

type (
	// 存储类型
	StorageClass int64

	// 解冻状态
	RestoreStatus int64

	// 禁用状态
	Status int64

	// 列举 API 版本
	ListerVersion int64

	// 对象详情
	ObjectDetails struct {
		Name                    string         // 对象名称
		UploadedAt              time.Time      // 上传时间
		ETag                    string         // 哈希值
		Size                    int64          // 对象大小，单位为字节
		MimeType                string         // 对象 MIME 类型
		StorageClass            StorageClass   // 存储类型
		EndUser                 string         // 唯一属主标识
		Status                  Status         // 存储状态
		RestoreStatus           RestoreStatus  // 冻结状态，仅对归档存储或深度归档存储的对象生效
		TransitionToIA          *time.Time     // 文件生命周期中转为低频存储的日期
		TransitionToArchiveIR   *time.Time     // 文件生命周期中转为归档直读存储的日期
		TransitionToArchive     *time.Time     // 文件生命周期中转为归档存储的日期
		TransitionToDeepArchive *time.Time     // 文件生命周期中转为深度归档存储的日期
		ExpireAt                *time.Time     // 文件过期删除日期
		MD5                     [md5.Size]byte // 对象 MD5 值
		Metadata                map[string]string
		Parts                   []int64 // 分片的大小
	}
)

const (
	// 标准存储类型
	StandardStorageClass StorageClass = iota

	// 低频访问存储类型
	IAStorageClass

	// 归档存储类型
	ArchiveStorageClass

	// 深度归档存储类型
	DeepArchiveStorageClass

	// 归档直读存储类型
	ArchiveIRStorageClass
)

const (
	// 启用状态
	EnabledStatus Status = iota

	// 禁用状态
	DisabledStatus
)

const (
	// 冻结中
	FrozenStatus RestoreStatus = iota

	// 解冻中
	RestoringStatus

	// 已解冻
	RestoredStatus
)

const (
	// 列举 V1
	ListerVersionV1 ListerVersion = iota
	// ListerVersionV2
)
