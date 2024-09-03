package objects

import (
	"context"
	"strings"
)

type (
	// 存储空间
	Bucket struct {
		name           string
		objectsManager *ObjectsManager
	}

	// 列举对象选项
	ListObjectsOptions struct {
		Limit     *uint64 // 最大列举数量
		Prefix    string  // 前缀
		Marker    string  // 标记
		NeedParts bool    // 是否需要分片信息
	}
)

// 存储空间名称
func (bucket *Bucket) Name() string {
	return bucket.name
}

// 获取存储空间对象
func (bucket *Bucket) Object(name string) *Object {
	return &Object{bucket, name}
}

// 获取存储空间目录
func (bucket *Bucket) Directory(prefix, pathSeparator string) *Directory {
	if pathSeparator == "" {
		pathSeparator = "/"
	}
	if prefix != "" && !strings.HasSuffix(prefix, pathSeparator) {
		prefix += pathSeparator
	}
	return &Directory{bucket, prefix, pathSeparator}
}

// 列举对象
func (bucket *Bucket) List(ctx context.Context, options *ListObjectsOptions) Lister {
	if options == nil {
		options = &ListObjectsOptions{}
	}

	switch bucket.objectsManager.listerVersion {
	case ListerVersionV1:
		fallthrough
	default:
		return newListerV1(ctx, bucket, options)
	}
}
