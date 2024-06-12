package objects

import (
	"context"

	"github.com/qiniu/go-sdk/v7/storagev2/apis"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
)

type (
	// 对象管理器
	ObjectsManager struct {
		storage *apis.Storage
		options *ObjectsManagerOptions
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

func NewObjectsManager(options *ObjectsManagerOptions) *ObjectsManager {
	if options == nil {
		options = &ObjectsManagerOptions{}
	}
	if options.BatchOpsExecutor == nil {
		options.BatchOpsExecutor = NewSerialBatchOpsExecutor(nil)
	}
	return &ObjectsManager{apis.NewStorage(&options.Options), options}
}

func (objectsManager *ObjectsManager) Bucket(name string) *Bucket {
	return &Bucket{name: name, objectsManager: objectsManager}
}

func (objectsManager *ObjectsManager) Batch(ctx context.Context, operations []Operation, options *BatchOptions) error {
	if len(operations) == 0 {
		return nil
	}
	if options == nil {
		options = &BatchOptions{}
	}
	batchOpsExecutor := options.BatchOpsExecutor
	if batchOpsExecutor == nil {
		batchOpsExecutor = objectsManager.options.BatchOpsExecutor
	}
	return batchOpsExecutor.ExecuteBatchOps(ctx, operations, objectsManager.storage)
}
