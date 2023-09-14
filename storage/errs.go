package storage

import (
	"errors"
)

var (
	// ErrBucketNotExist 用户存储空间不存在
	ErrBucketNotExist = errors.New("bucket not exist")

	// ErrNoSuchFile 文件已经存在
	//lint:ignore ST1005 历史问题，需要兼容
	ErrNoSuchFile = errors.New("No such file or directory")
)
