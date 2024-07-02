package objects

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/apis"
	"github.com/qiniu/go-sdk/v7/storagev2/apis/stat_object"
)

type (
	// 操作接口，用于发送单次请求或批量请求
	Operation interface {
		// 批量请求命令
		fmt.Stringer
		// 被操作的对象，必须返回至少一个
		relatedEntries() []entry
		// 处理返回结果
		handleResponse(*ObjectDetails, error)
	}

	entry struct {
		bucketName string
		objectName string
	}

	// 获取对象元信息操作
	StatObjectOperation struct {
		object     Object
		needParts  bool
		onResponse func(*ObjectDetails)
		onError    func(error)
	}

	// 移动对象操作
	MoveObjectOperation struct {
		fromObject Object
		toObject   entry
		force      bool
		onResponse func()
		onError    func(error)
	}

	// 复制对象操作
	CopyObjectOperation struct {
		fromObject Object
		toObject   entry
		force      bool
		onResponse func()
		onError    func(error)
	}

	// 删除对象操作
	DeleteObjectOperation struct {
		object     Object
		onResponse func()
		onError    func(error)
	}

	// 解冻对象操作
	RestoreObjectOperation struct {
		object          Object
		freezeAfterDays int64
		onResponse      func()
		onError         func(error)
	}

	// 设置对象存储类型操作
	SetObjectStorageClassOperation struct {
		object       Object
		storageClass StorageClass
		onResponse   func()
		onError      func(error)
	}

	// 设置对象状态
	SetObjectStatusOperation struct {
		object     Object
		status     Status
		onResponse func()
		onError    func(error)
	}

	// 设置对象元信息操作
	SetObjectMetadataOperation struct {
		object     Object
		mimeType   string
		metadata   map[string]string
		conditions map[string]string
		onResponse func()
		onError    func(error)
	}

	// 设置对象生命周期操作
	SetObjectLifeCycleOperation struct {
		object                 Object
		toIAAfterDays          int64
		toArchiveIRAfterDays   int64
		toArchiveAfterDays     int64
		toDeepArchiveAfterDays int64
		deleteAfterDays        int64
		onResponse             func()
		onError                func(error)
	}
)

func (operation *StatObjectOperation) NeedParts(needParts bool) *StatObjectOperation {
	copy := *operation
	copy.needParts = needParts
	return &copy
}

func (operation *StatObjectOperation) OnResponse(fn func(*ObjectDetails)) *StatObjectOperation {
	copy := *operation
	copy.onResponse = fn
	return &copy
}

func (operation *StatObjectOperation) OnError(fn func(error)) *StatObjectOperation {
	copy := *operation
	copy.onError = fn
	return &copy
}

func (operation *StatObjectOperation) relatedEntries() []entry {
	return []entry{{operation.object.bucket.name, operation.object.name}}
}

func (operation *StatObjectOperation) parseResponse(response *stat_object.Response, err error) (*ObjectDetails, error) {
	if err != nil {
		if operation.onError != nil {
			operation.onError(err)
		}
		return nil, err
	}

	object := ObjectDetails{
		Name:          operation.object.name,
		UploadedAt:    time.Unix(response.PutTime/1e7, (response.PutTime%1e7)*1e2),
		ETag:          response.Hash,
		Size:          response.Size,
		MimeType:      response.MimeType,
		StorageClass:  StorageClass(response.Type),
		EndUser:       response.EndUser,
		Status:        Status(response.Status),
		RestoreStatus: RestoreStatus(response.RestoringStatus),
		Metadata:      response.Metadata,
	}
	var md5 []byte
	if response.Md5 != "" {
		md5, err = hex.DecodeString(response.Md5)
		if err != nil {
			if operation.onError != nil {
				operation.onError(err)
			}
			return nil, err
		}
	}
	if len(md5) > 0 {
		copy(object.MD5[:], md5)
	}
	if len(response.Parts) > 0 {
		object.Parts = append(make(stat_object.PartSizes, 0, len(response.Parts)), response.Parts...)
	}
	if response.TransitionToIaTime > 0 {
		transitionToIA := time.Unix(response.TransitionToIaTime, 0)
		object.TransitionToIA = &transitionToIA
	}
	if response.TransitionToArchiveIrTime > 0 {
		transitionToArchiveIR := time.Unix(response.TransitionToArchiveIrTime, 0)
		object.TransitionToArchiveIR = &transitionToArchiveIR
	}
	if response.TransitionToArchiveTime > 0 {
		transitionToArchive := time.Unix(response.TransitionToArchiveTime, 0)
		object.TransitionToArchive = &transitionToArchive
	}
	if response.TransitionToDeepArchiveTime > 0 {
		transitionToDeepArchive := time.Unix(response.TransitionToDeepArchiveTime, 0)
		object.TransitionToDeepArchive = &transitionToDeepArchive
	}
	if response.ExpirationTime > 0 {
		expireAt := time.Unix(response.ExpirationTime, 0)
		object.ExpireAt = &expireAt
	}
	if operation.onResponse != nil {
		operation.onResponse(&object)
	}
	return &object, nil
}

func (operation *StatObjectOperation) handleResponse(object *ObjectDetails, err error) {
	if err != nil && operation.onError != nil {
		operation.onError(err)
	} else if operation.onResponse != nil {
		operation.onResponse(object)
	}
}

func (operation *StatObjectOperation) String() string {
	s := "stat/" + operation.object.encode()
	if operation.needParts {
		s += "/needparts/true"
	}
	return s
}

func (operation *StatObjectOperation) Call(ctx context.Context) (*ObjectDetails, error) {
	response, err := operation.object.bucket.objectsManager.storage.StatObject(ctx, &apis.StatObjectRequest{
		Entry:     operation.object.String(),
		NeedParts: operation.needParts,
	}, &apis.Options{
		OverwrittenBucketName: operation.object.bucket.name,
	})
	return operation.parseResponse(response, err)
}

var _ Operation = (*StatObjectOperation)(nil)

func (operation *MoveObjectOperation) Force(force bool) *MoveObjectOperation {
	copy := *operation
	copy.force = force
	return &copy
}

func (operation *MoveObjectOperation) OnResponse(fn func()) *MoveObjectOperation {
	copy := *operation
	copy.onResponse = fn
	return &copy
}

func (operation *MoveObjectOperation) OnError(fn func(error)) *MoveObjectOperation {
	copy := *operation
	copy.onError = fn
	return &copy
}

func (operation *MoveObjectOperation) relatedEntries() []entry {
	return []entry{{operation.fromObject.bucket.name, operation.fromObject.name}, operation.toObject}
}

func (operation *MoveObjectOperation) handleResponse(_ *ObjectDetails, err error) {
	if err != nil && operation.onError != nil {
		operation.onError(err)
	} else if operation.onResponse != nil {
		operation.onResponse()
	}
}

func (operation *MoveObjectOperation) String() string {
	s := "move/" + operation.fromObject.encode() + "/" + operation.toObject.encode()
	if operation.force {
		s += "/force/true"
	}
	return s
}

func (operation *MoveObjectOperation) Call(ctx context.Context) error {
	_, err := operation.fromObject.bucket.objectsManager.storage.MoveObject(ctx, &apis.MoveObjectRequest{
		SrcEntry:  operation.fromObject.String(),
		DestEntry: operation.toObject.String(),
		IsForce:   operation.force,
	}, &apis.Options{
		OverwrittenBucketName: operation.fromObject.bucket.name,
	})
	operation.handleResponse(nil, err)
	return err
}

var _ Operation = (*MoveObjectOperation)(nil)

func (operation *CopyObjectOperation) Force(force bool) *CopyObjectOperation {
	copy := *operation
	copy.force = force
	return &copy
}

func (operation *CopyObjectOperation) OnResponse(fn func()) *CopyObjectOperation {
	copy := *operation
	copy.onResponse = fn
	return &copy
}

func (operation *CopyObjectOperation) OnError(fn func(error)) *CopyObjectOperation {
	copy := *operation
	copy.onError = fn
	return &copy
}

func (operation *CopyObjectOperation) relatedEntries() []entry {
	return []entry{{operation.fromObject.bucket.name, operation.fromObject.name}, operation.toObject}
}

func (operation *CopyObjectOperation) handleResponse(_ *ObjectDetails, err error) {
	if err != nil && operation.onError != nil {
		operation.onError(err)
	} else if operation.onResponse != nil {
		operation.onResponse()
	}
}

func (operation *CopyObjectOperation) String() string {
	s := "copy/" + operation.fromObject.encode() + "/" + operation.toObject.encode()
	if operation.force {
		s += "/force/true"
	}
	return s
}

func (operation *CopyObjectOperation) Call(ctx context.Context) error {
	_, err := operation.fromObject.bucket.objectsManager.storage.CopyObject(ctx, &apis.CopyObjectRequest{
		SrcEntry:  operation.fromObject.String(),
		DestEntry: operation.toObject.String(),
		IsForce:   operation.force,
	}, &apis.Options{
		OverwrittenBucketName: operation.fromObject.bucket.name,
	})
	operation.handleResponse(nil, err)
	return err
}

var _ Operation = (*CopyObjectOperation)(nil)

func (operation *DeleteObjectOperation) OnResponse(fn func()) *DeleteObjectOperation {
	copy := *operation
	copy.onResponse = fn
	return &copy
}

func (operation *DeleteObjectOperation) OnError(fn func(error)) *DeleteObjectOperation {
	copy := *operation
	copy.onError = fn
	return &copy
}

func (operation *DeleteObjectOperation) handleResponse(_ *ObjectDetails, err error) {
	if err != nil && operation.onError != nil {
		operation.onError(err)
	} else if operation.onResponse != nil {
		operation.onResponse()
	}
}

func (operation *DeleteObjectOperation) relatedEntries() []entry {
	return []entry{{operation.object.bucket.name, operation.object.name}}
}

func (operation *DeleteObjectOperation) String() string {
	return "delete/" + operation.object.encode()
}

func (operation *DeleteObjectOperation) Call(ctx context.Context) error {
	_, err := operation.object.bucket.objectsManager.storage.DeleteObject(ctx, &apis.DeleteObjectRequest{
		Entry: operation.object.String(),
	}, &apis.Options{
		OverwrittenBucketName: operation.object.bucket.name,
	})
	operation.handleResponse(nil, err)
	return err
}

var _ Operation = (*DeleteObjectOperation)(nil)

func (operation *RestoreObjectOperation) OnResponse(fn func()) *RestoreObjectOperation {
	copy := *operation
	copy.onResponse = fn
	return &copy
}

func (operation *RestoreObjectOperation) OnError(fn func(error)) *RestoreObjectOperation {
	copy := *operation
	copy.onError = fn
	return &copy
}

func (operation *RestoreObjectOperation) relatedEntries() []entry {
	return []entry{{operation.object.bucket.name, operation.object.name}}
}

func (operation *RestoreObjectOperation) handleResponse(_ *ObjectDetails, err error) {
	if err != nil && operation.onError != nil {
		operation.onError(err)
	} else if operation.onResponse != nil {
		operation.onResponse()
	}
}

func (operation *RestoreObjectOperation) String() string {
	return "restoreAr/" + operation.object.encode() + "/freezeAfterDays/" + strconv.FormatInt(operation.freezeAfterDays, 10)
}

func (operation *RestoreObjectOperation) Call(ctx context.Context) error {
	_, err := operation.object.bucket.objectsManager.storage.RestoreArchivedObject(ctx, &apis.RestoreArchivedObjectRequest{
		Entry:           operation.object.String(),
		FreezeAfterDays: int64(operation.freezeAfterDays),
	}, &apis.Options{
		OverwrittenBucketName: operation.object.bucket.name,
	})
	operation.handleResponse(nil, err)
	return err
}

var _ Operation = (*RestoreObjectOperation)(nil)

func (operation *SetObjectStorageClassOperation) OnResponse(fn func()) *SetObjectStorageClassOperation {
	copy := *operation
	copy.onResponse = fn
	return &copy
}

func (operation *SetObjectStorageClassOperation) OnError(fn func(error)) *SetObjectStorageClassOperation {
	copy := *operation
	copy.onError = fn
	return &copy
}

func (operation *SetObjectStorageClassOperation) relatedEntries() []entry {
	return []entry{{operation.object.bucket.name, operation.object.name}}
}

func (operation *SetObjectStorageClassOperation) handleResponse(_ *ObjectDetails, err error) {
	if err != nil && operation.onError != nil {
		operation.onError(err)
	} else if operation.onResponse != nil {
		operation.onResponse()
	}
}

func (operation *SetObjectStorageClassOperation) String() string {
	return "chtype/" + operation.object.encode() + "/type/" + strconv.Itoa(int(operation.storageClass))
}

func (operation *SetObjectStorageClassOperation) Call(ctx context.Context) error {
	_, err := operation.object.bucket.objectsManager.storage.SetObjectFileType(ctx, &apis.SetObjectFileTypeRequest{
		Entry: operation.object.String(),
		Type:  int64(operation.storageClass),
	}, &apis.Options{
		OverwrittenBucketName: operation.object.bucket.name,
	})
	operation.handleResponse(nil, err)
	return err
}

var _ Operation = (*SetObjectStorageClassOperation)(nil)

func (operation *SetObjectStatusOperation) OnResponse(fn func()) *SetObjectStatusOperation {
	copy := *operation
	copy.onResponse = fn
	return &copy
}

func (operation *SetObjectStatusOperation) OnError(fn func(error)) *SetObjectStatusOperation {
	copy := *operation
	copy.onError = fn
	return &copy
}

func (operation *SetObjectStatusOperation) relatedEntries() []entry {
	return []entry{{operation.object.bucket.name, operation.object.name}}
}

func (operation *SetObjectStatusOperation) handleResponse(_ *ObjectDetails, err error) {
	if err != nil && operation.onError != nil {
		operation.onError(err)
	} else if operation.onResponse != nil {
		operation.onResponse()
	}
}

func (operation *SetObjectStatusOperation) String() string {
	return "chstatus/" + operation.object.encode() + "/status/" + strconv.Itoa(int(operation.status))
}

func (operation *SetObjectStatusOperation) Call(ctx context.Context) error {
	_, err := operation.object.bucket.objectsManager.storage.ModifyObjectStatus(ctx, &apis.ModifyObjectStatusRequest{
		Entry:  operation.object.String(),
		Status: int64(operation.status),
	}, &apis.Options{
		OverwrittenBucketName: operation.object.bucket.name,
	})
	operation.handleResponse(nil, err)
	return err
}

var _ Operation = (*SetObjectStatusOperation)(nil)

func (operation *SetObjectMetadataOperation) Metadata(metadata map[string]string) *SetObjectMetadataOperation {
	copy := *operation
	copy.metadata = metadata
	return &copy
}

func (operation *SetObjectMetadataOperation) Conditions(conds map[string]string) *SetObjectMetadataOperation {
	copy := *operation
	copy.conditions = conds
	return &copy
}

func (operation *SetObjectMetadataOperation) OnResponse(fn func()) *SetObjectMetadataOperation {
	copy := *operation
	copy.onResponse = fn
	return &copy
}

func (operation *SetObjectMetadataOperation) OnError(fn func(error)) *SetObjectMetadataOperation {
	copy := *operation
	copy.onError = fn
	return &copy
}

func (operation *SetObjectMetadataOperation) relatedEntries() []entry {
	return []entry{{operation.object.bucket.name, operation.object.name}}
}

func (operation *SetObjectMetadataOperation) handleResponse(_ *ObjectDetails, err error) {
	if err != nil && operation.onError != nil {
		operation.onError(err)
	} else if operation.onResponse != nil {
		operation.onResponse()
	}
}

func (operation *SetObjectMetadataOperation) String() string {
	s := "chgm/" + operation.object.encode() + "/mime/" + base64.URLEncoding.EncodeToString([]byte(operation.mimeType))
	for k, v := range operation.metadata {
		s += "/" + normalizeMetadataKey(k) + "/" + base64.URLEncoding.EncodeToString([]byte(v))
	}
	conds := []string{}
	for k, v := range operation.conditions {
		conds = append(conds, k+"="+v)
	}
	if len(conds) > 0 {
		s += "/cond/" + base64.URLEncoding.EncodeToString([]byte(strings.Join(conds, "&")))
	}
	return s
}

func (operation *SetObjectMetadataOperation) Call(ctx context.Context) error {
	conds := make([]string, 0, len(operation.conditions))
	for k, v := range operation.conditions {
		conds = append(conds, k+"="+v)
	}
	metadata := make(map[string]string, len(operation.metadata))
	for k, v := range operation.metadata {
		metadata[normalizeMetadataKey(k)] = v
	}
	_, err := operation.object.bucket.objectsManager.storage.ModifyObjectMetadata(ctx, &apis.ModifyObjectMetadataRequest{
		Entry:     operation.object.String(),
		MimeType:  operation.mimeType,
		Condition: strings.Join(conds, "&"),
		MetaData:  metadata,
	}, &apis.Options{
		OverwrittenBucketName: operation.object.bucket.name,
	})
	operation.handleResponse(nil, err)
	return err
}

var _ Operation = (*SetObjectMetadataOperation)(nil)

func (operation *SetObjectLifeCycleOperation) ToIAAfterDays(afterDays int64) *SetObjectLifeCycleOperation {
	operation.toIAAfterDays = afterDays
	return operation
}

func (operation *SetObjectLifeCycleOperation) ToArchiveIRAfterDays(afterDays int64) *SetObjectLifeCycleOperation {
	copy := *operation
	copy.toArchiveIRAfterDays = afterDays
	return &copy
}

func (operation *SetObjectLifeCycleOperation) ToArchiveAfterDays(afterDays int64) *SetObjectLifeCycleOperation {
	copy := *operation
	copy.toArchiveAfterDays = afterDays
	return &copy
}

func (operation *SetObjectLifeCycleOperation) ToDeepArchiveAfterDays(afterDays int64) *SetObjectLifeCycleOperation {
	copy := *operation
	copy.toDeepArchiveAfterDays = afterDays
	return &copy
}

func (operation *SetObjectLifeCycleOperation) DeleteAfterDays(afterDays int64) *SetObjectLifeCycleOperation {
	copy := *operation
	copy.deleteAfterDays = afterDays
	return &copy
}

func (operation *SetObjectLifeCycleOperation) OnResponse(fn func()) *SetObjectLifeCycleOperation {
	copy := *operation
	copy.onResponse = fn
	return &copy
}

func (operation *SetObjectLifeCycleOperation) OnError(fn func(error)) *SetObjectLifeCycleOperation {
	copy := *operation
	copy.onError = fn
	return &copy
}

func (operation *SetObjectLifeCycleOperation) relatedEntries() []entry {
	return []entry{{operation.object.bucket.name, operation.object.name}}
}

func (operation *SetObjectLifeCycleOperation) handleResponse(_ *ObjectDetails, err error) {
	if err != nil && operation.onError != nil {
		operation.onError(err)
	} else if operation.onResponse != nil {
		operation.onResponse()
	}
}

func (operation *SetObjectLifeCycleOperation) String() string {
	s := "lifecycle/" + operation.object.encode()
	if operation.toIAAfterDays > 0 {
		s += "/toIAAfterDays/" + strconv.FormatInt(operation.toIAAfterDays, 10)
	}
	if operation.toArchiveIRAfterDays > 0 {
		s += "/toArchiveIRAfterDays/" + strconv.FormatInt(operation.toArchiveIRAfterDays, 10)
	}
	if operation.toArchiveAfterDays > 0 {
		s += "/toArchiveAfterDays/" + strconv.FormatInt(operation.toArchiveAfterDays, 10)
	}
	if operation.toDeepArchiveAfterDays > 0 {
		s += "/toDeepArchiveAfterDays/" + strconv.FormatInt(operation.toDeepArchiveAfterDays, 10)
	}
	if operation.deleteAfterDays > 0 {
		s += "/deleteAfterDays/" + strconv.FormatInt(operation.deleteAfterDays, 10)
	}
	return s
}

func (operation *SetObjectLifeCycleOperation) Call(ctx context.Context) error {
	_, err := operation.object.bucket.objectsManager.storage.ModifyObjectLifeCycle(ctx, &apis.ModifyObjectLifeCycleRequest{
		Entry:                  operation.object.String(),
		ToIaAfterDays:          operation.toIAAfterDays,
		ToArchiveAfterDays:     operation.toArchiveAfterDays,
		ToDeepArchiveAfterDays: operation.toDeepArchiveAfterDays,
		ToArchiveIrAfterDays:   operation.toArchiveIRAfterDays,
		DeleteAfterDays:        operation.deleteAfterDays,
	}, &apis.Options{
		OverwrittenBucketName: operation.object.bucket.name,
	})
	operation.handleResponse(nil, err)
	return err
}

var _ Operation = (*SetObjectLifeCycleOperation)(nil)

func (entry Object) String() string {
	return entry.bucket.name + ":" + entry.name
}

func (entry Object) encode() string {
	return base64.URLEncoding.EncodeToString([]byte(entry.String()))
}

func (entry entry) String() string {
	return entry.bucketName + ":" + entry.objectName
}

func (entry entry) encode() string {
	return base64.URLEncoding.EncodeToString([]byte(entry.String()))
}

func normalizeMetadataKey(k string) string {
	if !strings.HasPrefix(k, "x-qn-meta-") {
		k = "x-qn-meta-" + k
	}
	return k
}
