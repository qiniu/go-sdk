// Package objects 提供七牛云对象存储的对象管理功能。
//
// [ObjectsManager] 提供流式 API 来管理存储空间中的对象，支持查询、复制、
// 移动、删除、归档恢复、生命周期管理等操作，以及批量操作。
//
// # 创建管理器
//
//	objectsManager := objects.NewObjectsManager(&objects.ObjectsManagerOptions{
//	    Options: http_client.Options{Credentials: cred},
//	})
//
// # 链式 API
//
// 通过 Bucket → Object 链式调用构建操作：
//
//	bucket := objectsManager.Bucket("my-bucket")
//	obj := bucket.Object("my-file.txt")
//
//	// 查询对象信息
//	info, err := obj.Stat().Call(ctx)
//
//	// 复制对象
//	err := obj.CopyTo("target-bucket", "new-name.txt").Call(ctx)
//
//	// 移动对象
//	err := obj.MoveTo("target-bucket", "new-name.txt").Call(ctx)
//
//	// 删除对象
//	err := obj.Delete().Call(ctx)
//
//	// 修改存储类型
//	err := obj.SetStorageClass(objects.IAStorageClass).Call(ctx)
//
//	// 归档恢复
//	err := obj.Restore(7).Call(ctx)
//
// # 列举对象
//
//	lister := bucket.List(ctx, &objects.ListObjectsOptions{Prefix: "images/"})
//	defer lister.Close()
//	var details objects.ObjectDetails
//	for lister.Next(&details) {
//	    fmt.Println(details.Name, details.Size)
//	}
//	if err := lister.Error(); err != nil {
//	    // 处理错误
//	}
//
// # 批量操作
//
// 将多个 [Operation] 收集后批量执行：
//
//	ops := make([]objects.Operation, 0)
//	ops = append(ops, bucket.Object("a.txt").Delete())
//	ops = append(ops, bucket.Object("b.txt").Delete())
//	err := objectsManager.Batch(ctx, ops, &objects.BatchOptions{})
//
// # 目录操作
//
// 通过 [Directory] 批量操作同前缀的对象：
//
//	dir := bucket.Directory("logs/", "/")
//	err := dir.Delete(ctx)
//	err := dir.CopyTo(ctx, "backup-bucket", "logs-backup/")
//
// # 存储类型
//
//   - [StandardStorageClass]: 标准存储
//   - [IAStorageClass]: 低频存储
//   - [ArchiveStorageClass]: 归档存储
//   - [DeepArchiveStorageClass]: 深度归档存储
//   - [ArchiveIRStorageClass]: 归档直读存储
package objects
