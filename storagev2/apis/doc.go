// Package apis 提供七牛云对象存储的低级类型化 API 客户端。
//
// 本包代码由 api-generator 自动生成，请勿直接修改。
//
// [Storage] 提供了所有存储 API 的类型化方法，每个方法对应一个 API 接口，
// 使用独立的 Request/Response 结构体。推荐使用 storagev2/uploader、
// storagev2/downloader、storagev2/objects 等高级包，
// 仅在需要直接调用底层 API 时使用本包。
//
// # 创建客户端
//
//	storage := apis.NewStorage(&http_client.Options{Credentials: cred})
//
// # 调用 API
//
// 所有方法遵循统一签名：Method(ctx, request, options) (response, error)
//
//	// 查询对象信息
//	resp, err := storage.StatObject(ctx, &apis.StatObjectRequest{
//	    Entry: "my-bucket:my-file.txt",
//	}, nil)
//
//	// 获取存储空间列表
//	resp, err := storage.GetBuckets(ctx, &apis.GetBucketsRequest{}, nil)
//
// # 主要 API 分类
//
//   - 对象操作: StatObject, DeleteObject, CopyObject, MoveObject, BatchOps
//   - 存储空间: CreateBucket, DeleteBucket, GetBucketInfo, GetBuckets
//   - 上传: ResumableUploadV1*, ResumableUploadV2*
//   - 列举: GetObjects, GetObjectsV2
//   - 生命周期: ModifyObjectLifeCycle, DeleteObjectAfterDays, SetObjectFileType
//   - 归档: RestoreArchivedObject
//   - 异步抓取: AsyncFetchObject, GetAsyncFetchTask
//   - 区域查询: GetRegions, QueryBucketV2, QueryBucketV4
//   - 域名: GetBucketDomains, GetBucketDomainsV3
//   - 规则/事件: AddBucketRules, AddBucketEventRule 等
//   - CORS/标签/防盗链: SetBucketCorsRules, SetBucketTaggings, SetBucketReferAntiLeech
package apis
