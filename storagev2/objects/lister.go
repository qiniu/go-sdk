package objects

import (
	"encoding/hex"
	"io"
	"time"

	"github.com/qiniu/go-sdk/v7/internal/context"
	"github.com/qiniu/go-sdk/v7/storagev2/apis"
	"github.com/qiniu/go-sdk/v7/storagev2/apis/get_objects"
	"github.com/qiniu/go-sdk/v7/storagev2/apis/stat_object"
)

type (
	// 对象列举接口
	Lister interface {
		io.Closer

		// 读取下一条记录
		Next(*ObjectDetails) bool

		// 获取错误信息
		Error() error

		// 获取位置标记
		Marker() string
	}

	listerV1 struct {
		ctx       context.Context
		bucket    *Bucket
		rest      *uint64
		marker    string
		entries   get_objects.ListedObjects
		options   *ListObjectsOptions
		firstCall bool
		err       error
	}
)

const listerV1DefaultLimit = 1000

func newListerV1(ctx context.Context, bucket *Bucket, options *ListObjectsOptions) Lister {
	if options == nil {
		options = &ListObjectsOptions{}
	}
	return &listerV1{ctx: ctx, bucket: bucket, rest: options.Limit, marker: options.Marker, options: options, firstCall: true}
}

func (v1 *listerV1) Next(object *ObjectDetails) bool {
	if len(v1.entries) == 0 {
		if err := v1.callListApi(); err != nil {
			v1.err = err
			return false
		}
	}
	if len(v1.entries) == 0 {
		return false
	}
	entry := v1.entries[0]
	v1.entries = v1.entries[1:]
	if err := object.fromListedObjectEntry(&entry); err != nil {
		v1.err = err
		return false
	}
	return true
}

func (v1 *listerV1) Marker() string {
	return v1.marker
}

func (v1 *listerV1) callListApi() error {
	if v1.marker == "" && !v1.firstCall {
		return nil
	}
	v1.firstCall = false

	request := apis.GetObjectsRequest{
		Bucket:    v1.bucket.name,
		Marker:    v1.marker,
		Prefix:    v1.options.Prefix,
		NeedParts: v1.options.NeedParts,
	}
	if v1.rest != nil && *v1.rest < listerV1DefaultLimit {
		if *v1.rest == 0 {
			return nil
		}
		request.Limit = int64(*v1.rest)
	}
	response, err := v1.bucket.objectsManager.storage.GetObjects(v1.ctx, &request, nil)
	if err != nil {
		return err
	}
	v1.entries = response.Items
	v1.marker = response.Marker
	request.Marker = response.Marker
	if v1.rest != nil {
		*v1.rest -= uint64(len(response.Items))
	}
	return nil
}

func (v1 *listerV1) Error() error {
	return v1.err
}

func (v1 *listerV1) Close() error {
	return v1.err
}

func (object *ObjectDetails) fromListedObjectEntry(entry *get_objects.ListedObjectEntry) error {
	var (
		md5 []byte
		err error
	)
	object.Name = entry.Key
	object.UploadedAt = time.Unix(entry.PutTime/1e7, (entry.PutTime%1e7)*1e2)
	object.ETag = entry.Hash
	object.Size = entry.Size
	object.MimeType = entry.MimeType
	object.StorageClass = StorageClass(entry.Type)
	object.EndUser = entry.EndUser
	object.Status = Status(entry.Status)
	object.RestoreStatus = RestoreStatus(entry.RestoringStatus)
	object.Metadata = entry.Metadata
	if entry.Md5 != "" {
		md5, err = hex.DecodeString(entry.Md5)
		if err != nil {
			return err
		}
	}
	if len(md5) > 0 {
		copy(object.MD5[:], md5)
	}
	if len(entry.Parts) > 0 {
		object.Parts = append(make(stat_object.PartSizes, 0, len(entry.Parts)), entry.Parts...)
	}
	return nil
}
