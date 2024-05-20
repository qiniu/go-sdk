package objects

type Object struct {
	bucket *Bucket
	name   string
}

func (object *Object) Stat() *StatObjectOperation {
	return &StatObjectOperation{
		entry: object.toBucketEntry(),
	}
}

func (object *Object) MoveTo(toBucketName, toObjectName string) *MoveObjectOperation {
	return &MoveObjectOperation{
		fromEntry: object.toBucketEntry(),
		toEntry:   Entry{toBucketName, toObjectName},
	}
}

func (object *Object) CopyTo(toBucketName, toObjectName string) *CopyObjectOperation {
	return &CopyObjectOperation{
		fromEntry: object.toBucketEntry(),
		toEntry:   Entry{toBucketName, toObjectName},
	}
}

func (object *Object) Delete() *DeleteObjectOperation {
	return &DeleteObjectOperation{
		entry: object.toBucketEntry(),
	}
}

func (object *Object) Restore(freezeAfterDays int64) *RestoreObjectOperation {
	return &RestoreObjectOperation{
		entry:           object.toBucketEntry(),
		freezeAfterDays: freezeAfterDays,
	}
}

func (object *Object) SetStorageClass(storageClass StorageClass) *SetObjectStorageClassOperation {
	return &SetObjectStorageClassOperation{
		entry:        object.toBucketEntry(),
		storageClass: storageClass,
	}
}

func (object *Object) SetStatus(status Status) *SetObjectStatusOperation {
	return &SetObjectStatusOperation{
		entry:  object.toBucketEntry(),
		status: status,
	}
}

func (object *Object) SetMetadata(mimeType string) *SetObjectMetadataOperation {
	return &SetObjectMetadataOperation{
		entry:    object.toBucketEntry(),
		mimeType: mimeType,
	}
}

func (object *Object) SetLifeCycle() *SetObjectLifeCycleOperation {
	return &SetObjectLifeCycleOperation{
		entry: object.toBucketEntry(),
	}
}

func (object *Object) toBucketEntry() bucketEntry {
	return bucketEntry{object.bucket, object.name}
}
