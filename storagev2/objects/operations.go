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
		relatedEntries() []Entry
		// 处理返回结果
		handleResponse(*ObjectDetails, error)
	}

	bucketEntry struct {
		bucket     *Bucket
		objectName string
	}

	Entry struct {
		bucketName string
		objectName string
	}

	StatObjectOperation struct {
		entry      bucketEntry
		needParts  bool
		onResponse func(*ObjectDetails)
		onError    func(error)
	}

	MoveObjectOperation struct {
		fromEntry  bucketEntry
		toEntry    Entry
		force      bool
		onResponse func()
		onError    func(error)
	}

	CopyObjectOperation struct {
		fromEntry  bucketEntry
		toEntry    Entry
		force      bool
		onResponse func()
		onError    func(error)
	}

	DeleteObjectOperation struct {
		entry      bucketEntry
		onResponse func()
		onError    func(error)
	}

	RestoreObjectOperation struct {
		entry           bucketEntry
		freezeAfterDays int64
		onResponse      func()
		onError         func(error)
	}

	SetObjectStorageClassOperation struct {
		entry        bucketEntry
		storageClass StorageClass
		onResponse   func()
		onError      func(error)
	}

	SetObjectStatusOperation struct {
		entry      bucketEntry
		status     Status
		onResponse func()
		onError    func(error)
	}

	SetObjectMetadataOperation struct {
		entry      bucketEntry
		mimeType   string
		metadata   map[string]string
		conditions map[string]string
		onResponse func()
		onError    func(error)
	}

	SetObjectLifeCycleOperation struct {
		entry                  bucketEntry
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
	operation.needParts = needParts
	return operation
}

func (operation *StatObjectOperation) OnResponse(fn func(*ObjectDetails)) *StatObjectOperation {
	operation.onResponse = fn
	return operation
}

func (operation *StatObjectOperation) OnError(fn func(error)) *StatObjectOperation {
	operation.onError = fn
	return operation
}

func (operation *StatObjectOperation) relatedEntries() []Entry {
	return []Entry{{operation.entry.bucket.name, operation.entry.objectName}}
}

func (operation *StatObjectOperation) parseResponse(response *stat_object.Response, err error) (*ObjectDetails, error) {
	if err != nil {
		if operation.onError != nil {
			operation.onError(err)
		}
		return nil, err
	}

	object := ObjectDetails{
		Name:          operation.entry.objectName,
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
	s := "stat/" + operation.entry.encode()
	if operation.needParts {
		s += "/needparts/true"
	}
	return s
}

func (operation *StatObjectOperation) Call(ctx context.Context) (*ObjectDetails, error) {
	response, err := operation.entry.bucket.objectsManager.storage.StatObject(ctx, &apis.StatObjectRequest{
		Entry:     operation.entry.String(),
		NeedParts: operation.needParts,
	}, &apis.Options{
		OverwrittenBucketName: operation.entry.bucket.name,
	})
	return operation.parseResponse(response, err)
}

var _ Operation = (*StatObjectOperation)(nil)

func (operation *MoveObjectOperation) Force(force bool) *MoveObjectOperation {
	operation.force = force
	return operation
}

func (operation *MoveObjectOperation) OnResponse(fn func()) *MoveObjectOperation {
	operation.onResponse = fn
	return operation
}

func (operation *MoveObjectOperation) OnError(fn func(error)) *MoveObjectOperation {
	operation.onError = fn
	return operation
}

func (operation *MoveObjectOperation) relatedEntries() []Entry {
	return []Entry{{operation.fromEntry.bucket.name, operation.fromEntry.objectName}, operation.toEntry}
}

func (operation *MoveObjectOperation) handleResponse(_ *ObjectDetails, err error) {
	if err != nil && operation.onError != nil {
		operation.onError(err)
	} else if operation.onResponse != nil {
		operation.onResponse()
	}
}

func (operation *MoveObjectOperation) String() string {
	s := "move/" + operation.fromEntry.encode() + "/" + operation.toEntry.encode()
	if operation.force {
		s += "/force/true"
	}
	return s
}

func (operation *MoveObjectOperation) Call(ctx context.Context) error {
	_, err := operation.fromEntry.bucket.objectsManager.storage.MoveObject(ctx, &apis.MoveObjectRequest{
		SrcEntry:  operation.fromEntry.String(),
		DestEntry: operation.toEntry.String(),
		IsForce:   operation.force,
	}, &apis.Options{
		OverwrittenBucketName: operation.fromEntry.bucket.name,
	})
	operation.handleResponse(nil, err)
	return err
}

var _ Operation = (*MoveObjectOperation)(nil)

func (operation *CopyObjectOperation) Force(force bool) *CopyObjectOperation {
	operation.force = force
	return operation
}

func (operation *CopyObjectOperation) OnResponse(fn func()) *CopyObjectOperation {
	operation.onResponse = fn
	return operation
}

func (operation *CopyObjectOperation) OnError(fn func(error)) *CopyObjectOperation {
	operation.onError = fn
	return operation
}

func (operation *CopyObjectOperation) relatedEntries() []Entry {
	return []Entry{{operation.fromEntry.bucket.name, operation.fromEntry.objectName}, operation.toEntry}
}

func (operation *CopyObjectOperation) handleResponse(_ *ObjectDetails, err error) {
	if err != nil && operation.onError != nil {
		operation.onError(err)
	} else if operation.onResponse != nil {
		operation.onResponse()
	}
}

func (operation *CopyObjectOperation) String() string {
	s := "copy/" + operation.fromEntry.encode() + "/" + operation.toEntry.encode()
	if operation.force {
		s += "/force/true"
	}
	return s
}

func (operation *CopyObjectOperation) Call(ctx context.Context) error {
	_, err := operation.fromEntry.bucket.objectsManager.storage.CopyObject(ctx, &apis.CopyObjectRequest{
		SrcEntry:  operation.fromEntry.String(),
		DestEntry: operation.toEntry.String(),
		IsForce:   operation.force,
	}, &apis.Options{
		OverwrittenBucketName: operation.fromEntry.bucket.name,
	})
	operation.handleResponse(nil, err)
	return err
}

var _ Operation = (*CopyObjectOperation)(nil)

func (operation *DeleteObjectOperation) OnResponse(fn func()) *DeleteObjectOperation {
	operation.onResponse = fn
	return operation
}

func (operation *DeleteObjectOperation) OnError(fn func(error)) *DeleteObjectOperation {
	operation.onError = fn
	return operation
}

func (operation *DeleteObjectOperation) handleResponse(_ *ObjectDetails, err error) {
	if err != nil && operation.onError != nil {
		operation.onError(err)
	} else if operation.onResponse != nil {
		operation.onResponse()
	}
}

func (operation *DeleteObjectOperation) relatedEntries() []Entry {
	return []Entry{{operation.entry.bucket.name, operation.entry.objectName}}
}

func (operation *DeleteObjectOperation) String() string {
	return "delete/" + operation.entry.encode()
}

func (operation *DeleteObjectOperation) Call(ctx context.Context) error {
	_, err := operation.entry.bucket.objectsManager.storage.DeleteObject(ctx, &apis.DeleteObjectRequest{
		Entry: operation.entry.String(),
	}, &apis.Options{
		OverwrittenBucketName: operation.entry.bucket.name,
	})
	operation.handleResponse(nil, err)
	return err
}

var _ Operation = (*DeleteObjectOperation)(nil)

func (operation *RestoreObjectOperation) OnResponse(fn func()) *RestoreObjectOperation {
	operation.onResponse = fn
	return operation
}

func (operation *RestoreObjectOperation) OnError(fn func(error)) *RestoreObjectOperation {
	operation.onError = fn
	return operation
}

func (operation *RestoreObjectOperation) relatedEntries() []Entry {
	return []Entry{{operation.entry.bucket.name, operation.entry.objectName}}
}

func (operation *RestoreObjectOperation) handleResponse(_ *ObjectDetails, err error) {
	if err != nil && operation.onError != nil {
		operation.onError(err)
	} else if operation.onResponse != nil {
		operation.onResponse()
	}
}

func (operation *RestoreObjectOperation) String() string {
	return "restoreAr/" + operation.entry.encode() + "/freezeAfterDays/" + strconv.FormatInt(operation.freezeAfterDays, 10)
}

func (operation *RestoreObjectOperation) Call(ctx context.Context) error {
	_, err := operation.entry.bucket.objectsManager.storage.RestoreArchivedObject(ctx, &apis.RestoreArchivedObjectRequest{
		Entry:           operation.entry.String(),
		FreezeAfterDays: int64(operation.freezeAfterDays),
	}, &apis.Options{
		OverwrittenBucketName: operation.entry.bucket.name,
	})
	operation.handleResponse(nil, err)
	return err
}

var _ Operation = (*RestoreObjectOperation)(nil)

func (operation *SetObjectStorageClassOperation) OnResponse(fn func()) *SetObjectStorageClassOperation {
	operation.onResponse = fn
	return operation
}

func (operation *SetObjectStorageClassOperation) OnError(fn func(error)) *SetObjectStorageClassOperation {
	operation.onError = fn
	return operation
}

func (operation *SetObjectStorageClassOperation) relatedEntries() []Entry {
	return []Entry{{operation.entry.bucket.name, operation.entry.objectName}}
}

func (operation *SetObjectStorageClassOperation) handleResponse(_ *ObjectDetails, err error) {
	if err != nil && operation.onError != nil {
		operation.onError(err)
	} else if operation.onResponse != nil {
		operation.onResponse()
	}
}

func (operation *SetObjectStorageClassOperation) String() string {
	return "chtype/" + operation.entry.encode() + "/type/" + strconv.Itoa(int(operation.storageClass))
}

func (operation *SetObjectStorageClassOperation) Call(ctx context.Context) error {
	_, err := operation.entry.bucket.objectsManager.storage.SetObjectFileType(ctx, &apis.SetObjectFileTypeRequest{
		Entry: operation.entry.String(),
		Type:  int64(operation.storageClass),
	}, &apis.Options{
		OverwrittenBucketName: operation.entry.bucket.name,
	})
	operation.handleResponse(nil, err)
	return err
}

var _ Operation = (*SetObjectStorageClassOperation)(nil)

func (operation *SetObjectStatusOperation) OnResponse(fn func()) *SetObjectStatusOperation {
	operation.onResponse = fn
	return operation
}

func (operation *SetObjectStatusOperation) OnError(fn func(error)) *SetObjectStatusOperation {
	operation.onError = fn
	return operation
}

func (operation *SetObjectStatusOperation) relatedEntries() []Entry {
	return []Entry{{operation.entry.bucket.name, operation.entry.objectName}}
}

func (operation *SetObjectStatusOperation) handleResponse(_ *ObjectDetails, err error) {
	if err != nil && operation.onError != nil {
		operation.onError(err)
	} else if operation.onResponse != nil {
		operation.onResponse()
	}
}

func (operation *SetObjectStatusOperation) String() string {
	return "chstatus/" + operation.entry.encode() + "/status/" + strconv.Itoa(int(operation.status))
}

func (operation *SetObjectStatusOperation) Call(ctx context.Context) error {
	_, err := operation.entry.bucket.objectsManager.storage.ModifyObjectStatus(ctx, &apis.ModifyObjectStatusRequest{
		Entry:  operation.entry.String(),
		Status: int64(operation.status),
	}, &apis.Options{
		OverwrittenBucketName: operation.entry.bucket.name,
	})
	operation.handleResponse(nil, err)
	return err
}

var _ Operation = (*SetObjectStatusOperation)(nil)

func (operation *SetObjectMetadataOperation) Metadata(metadata map[string]string) *SetObjectMetadataOperation {
	operation.metadata = metadata
	return operation
}

func (operation *SetObjectMetadataOperation) Conditions(conds map[string]string) *SetObjectMetadataOperation {
	operation.conditions = conds
	return operation
}

func (operation *SetObjectMetadataOperation) OnResponse(fn func()) *SetObjectMetadataOperation {
	operation.onResponse = fn
	return operation
}

func (operation *SetObjectMetadataOperation) OnError(fn func(error)) *SetObjectMetadataOperation {
	operation.onError = fn
	return operation
}

func (operation *SetObjectMetadataOperation) relatedEntries() []Entry {
	return []Entry{{operation.entry.bucket.name, operation.entry.objectName}}
}

func (operation *SetObjectMetadataOperation) handleResponse(_ *ObjectDetails, err error) {
	if err != nil && operation.onError != nil {
		operation.onError(err)
	} else if operation.onResponse != nil {
		operation.onResponse()
	}
}

func (operation *SetObjectMetadataOperation) String() string {
	s := "chgm/" + operation.entry.encode() + "/mime/" + base64.URLEncoding.EncodeToString([]byte(operation.mimeType))
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
	_, err := operation.entry.bucket.objectsManager.storage.ModifyObjectMetadata(ctx, &apis.ModifyObjectMetadataRequest{
		Entry:     operation.entry.String(),
		MimeType:  operation.mimeType,
		Condition: strings.Join(conds, "&"),
		MetaData:  metadata,
	}, &apis.Options{
		OverwrittenBucketName: operation.entry.bucket.name,
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
	operation.toArchiveIRAfterDays = afterDays
	return operation
}

func (operation *SetObjectLifeCycleOperation) ToArchiveAfterDays(afterDays int64) *SetObjectLifeCycleOperation {
	operation.toArchiveAfterDays = afterDays
	return operation
}

func (operation *SetObjectLifeCycleOperation) ToDeepArchiveAfterDays(afterDays int64) *SetObjectLifeCycleOperation {
	operation.toDeepArchiveAfterDays = afterDays
	return operation
}

func (operation *SetObjectLifeCycleOperation) DeleteAfterDays(afterDays int64) *SetObjectLifeCycleOperation {
	operation.deleteAfterDays = afterDays
	return operation
}

func (operation *SetObjectLifeCycleOperation) OnResponse(fn func()) *SetObjectLifeCycleOperation {
	operation.onResponse = fn
	return operation
}

func (operation *SetObjectLifeCycleOperation) OnError(fn func(error)) *SetObjectLifeCycleOperation {
	operation.onError = fn
	return operation
}

func (operation *SetObjectLifeCycleOperation) relatedEntries() []Entry {
	return []Entry{{operation.entry.bucket.name, operation.entry.objectName}}
}

func (operation *SetObjectLifeCycleOperation) handleResponse(_ *ObjectDetails, err error) {
	if err != nil && operation.onError != nil {
		operation.onError(err)
	} else if operation.onResponse != nil {
		operation.onResponse()
	}
}

func (operation *SetObjectLifeCycleOperation) String() string {
	s := "lifecycle/" + operation.entry.encode()
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
	_, err := operation.entry.bucket.objectsManager.storage.ModifyObjectLifeCycle(ctx, &apis.ModifyObjectLifeCycleRequest{
		Entry:                  operation.entry.String(),
		ToIaAfterDays:          operation.toIAAfterDays,
		ToArchiveAfterDays:     operation.toArchiveAfterDays,
		ToDeepArchiveAfterDays: operation.toDeepArchiveAfterDays,
		ToArchiveIrAfterDays:   operation.toArchiveIRAfterDays,
		DeleteAfterDays:        operation.deleteAfterDays,
	}, &apis.Options{
		OverwrittenBucketName: operation.entry.bucket.name,
	})
	operation.handleResponse(nil, err)
	return err
}

var _ Operation = (*SetObjectLifeCycleOperation)(nil)

func (entry bucketEntry) String() string {
	return entry.bucket.name + ":" + entry.objectName
}

func (entry bucketEntry) encode() string {
	return base64.URLEncoding.EncodeToString([]byte(entry.String()))
}

func (entry Entry) String() string {
	return entry.bucketName + ":" + entry.objectName
}

func (entry Entry) encode() string {
	return base64.URLEncoding.EncodeToString([]byte(entry.String()))
}

func normalizeMetadataKey(k string) string {
	if !strings.HasPrefix(k, "x-qn-meta-") {
		k = "x-qn-meta-" + k
	}
	return k
}
