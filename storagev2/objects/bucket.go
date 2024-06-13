package objects

import (
	"context"
	"strings"
)

type (
	Bucket struct {
		name           string
		objectsManager *ObjectsManager
	}

	ListObjectsOptions struct {
		Limit     *uint64
		Prefix    string
		Marker    string
		NeedParts bool
	}
)

func (bucket *Bucket) Name() string {
	return bucket.name
}

func (bucket *Bucket) Object(name string) *Object {
	return &Object{bucket, name}
}

func (bucket *Bucket) Directory(prefix, delimiter string) *Directory {
	if delimiter == "" {
		delimiter = "/"
	}
	if prefix != "" && !strings.HasSuffix(prefix, delimiter) {
		prefix += delimiter
	}
	return &Directory{bucket, prefix, delimiter}
}

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
