package objects

import (
	"context"

	"github.com/qiniu/go-sdk/v7/storagev2/apis"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
)

type (
	// 对象管理器
	ObjectsManager struct {
		storage          *apis.Storage
		options          httpclient.Options
		listerVersion    ListerVersion
		batchOpsExecutor BatchOpsExecutor
	}

	// 对象管理器选项
	ObjectsManagerOptions struct {
		// HTTP 客户端选项
		httpclient.Options

		// 分片列举版本，如果不填写，默认为 V1
		ListerVersion ListerVersion

		// 批处理执行器，如果不填写，默认为串型批处理执行器
		BatchOpsExecutor BatchOpsExecutor
	}

	// 批处理选项
	BatchOptions struct {
		// 批处理执行器，如果不填写，默认使用 ObjectsManager 的批处理执行器
		BatchOpsExecutor BatchOpsExecutor
	}
)

// 创建对象管理器
func NewObjectsManager(options *ObjectsManagerOptions) *ObjectsManager {
	if options == nil {
		options = &ObjectsManagerOptions{}
	}
	batchOpsExecutor := options.BatchOpsExecutor
	if batchOpsExecutor == nil {
		batchOpsExecutor = NewConcurrentBatchOpsExecutor(nil)
	}
	return &ObjectsManager{
		storage:          apis.NewStorage(&options.Options),
		options:          options.Options,
		listerVersion:    options.ListerVersion,
		batchOpsExecutor: batchOpsExecutor,
	}
}

// 获取存储空间
func (objectsManager *ObjectsManager) Bucket(name string) *Bucket {
	return &Bucket{name: name, objectsManager: objectsManager}
}

// 执行批处理操作
func (objectsManager *ObjectsManager) Batch(ctx context.Context, operations []Operation, options *BatchOptions) error {
	if len(operations) == 0 {
		return nil
	}
	if options == nil {
		options = &BatchOptions{}
	}
	batchOpsExecutor := options.BatchOpsExecutor
	if batchOpsExecutor == nil {
		batchOpsExecutor = objectsManager.batchOpsExecutor
	}
	return batchOpsExecutor.ExecuteBatchOps(ctx, operations, objectsManager.storage)
}
