// Package region 提供七牛云对象存储的区域信息和服务端点管理。
//
// 区域（[Region]）包含各服务的访问端点，SDK 通过 [RegionsProvider] 接口
// 自动获取区域信息进行请求路由。
//
// # 获取区域
//
//	// 通过区域 ID 获取（z0=华东, z1=华北, z2=华南, na0=北美, as0=东南亚）
//	r := region.GetRegionByID("z0", true)
//
// # RegionsProvider 接口
//
// [RegionsProvider] 是 storagev2 体系中传递区域信息的标准接口：
//
//	type RegionsProvider interface {
//	    GetRegions(context.Context) ([]*Region, error)
//	}
//
// [*Region] 本身实现了该接口，可直接作为 Provider 使用。
//
// # 自动查询区域
//
// 根据 Bucket 自动查询所属区域（推荐），结果会自动缓存：
//
//	bucketQuery, err := region.NewBucketRegionsQuery(
//	    http_client.DefaultBucketHosts(),
//	    &region.BucketRegionsQueryOptions{},
//	)
//	provider := bucketQuery.Query("accessKey", "my-bucket")
//
// # 服务端点
//
// [Endpoints] 结构包含 Preferred（首选）、Alternative（备选）和
// Accelerated（加速）三组地址，SDK 按优先级依次尝试。
//
// 服务类型常量：
//
//   - [ServiceUp]: 上传服务
//   - [ServiceIo]: IO 下载服务
//   - [ServiceRs]: 资源管理服务
//   - [ServiceRsf]: 资源列举服务
//   - [ServiceApi]: API 服务
//   - [ServiceBucket]: Bucket 服务
package region
