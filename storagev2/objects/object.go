package objects

// 对象
type Object struct {
	bucket *Bucket
	name   string
}

// 获取对象元信息
func (object *Object) Stat() *StatObjectOperation {
	return &StatObjectOperation{
		object: *object,
	}
}

// 移动对象
func (object *Object) MoveTo(toBucketName, toObjectName string) *MoveObjectOperation {
	return &MoveObjectOperation{
		fromObject: *object,
		toObject:   entry{toBucketName, toObjectName},
	}
}

// 复制对象
func (object *Object) CopyTo(toBucketName, toObjectName string) *CopyObjectOperation {
	return &CopyObjectOperation{
		fromObject: *object,
		toObject:   entry{toBucketName, toObjectName},
	}
}

// 删除对象
func (object *Object) Delete() *DeleteObjectOperation {
	return &DeleteObjectOperation{
		object: *object,
	}
}

// 解冻对象
func (object *Object) Restore(freezeAfterDays int64) *RestoreObjectOperation {
	return &RestoreObjectOperation{
		object:          *object,
		freezeAfterDays: freezeAfterDays,
	}
}

// 设置对象存储类型
func (object *Object) SetStorageClass(storageClass StorageClass) *SetObjectStorageClassOperation {
	return &SetObjectStorageClassOperation{
		object:       *object,
		storageClass: storageClass,
	}
}

// 设置对象状态
func (object *Object) SetStatus(status Status) *SetObjectStatusOperation {
	return &SetObjectStatusOperation{
		object: *object,
		status: status,
	}
}

// 设置对象元信息
func (object *Object) SetMetadata(mimeType string) *SetObjectMetadataOperation {
	return &SetObjectMetadataOperation{
		object:   *object,
		mimeType: mimeType,
	}
}

// 设置对象生命周期
func (object *Object) SetLifeCycle() *SetObjectLifeCycleOperation {
	return &SetObjectLifeCycleOperation{
		object: *object,
	}
}
