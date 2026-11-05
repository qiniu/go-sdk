// storage 包提供了资源的上传，管理，数据处理等功能。其中资源的上传又提供了表单上传的方式以及分片上传的方式，其中分片上传的方式还支持断点续传。
//
// 该包中提供了 BucketManager 用来进行资源管理，比如获取文件信息，文件复制，删除，重命名等，以及很多高级功能如修改文件类型，
// 修改文件的生命周期，修改文件的存储类型等。
//
// 该包中还提供了 FormUploader 和 ResumeUploader 来分别支持表单上传和分片上传，断点续传等功能，对于较大的文件，比如100MB以上的文件，一般
// 建议采用分片上传的方式来保证上传的效率和可靠性。
//
// 对于数据处理，则提供了 OperationManager，可以使用它来发送持久化数据处理请求，及查询数据处理的状态。
//
// # 生命周期查询的空值
//
// 生命周期时间以 Unix 秒表示；PutTime 的单位仍为 100 纳秒。
// 单文件查询的 FileInfo 生命周期字段为 int64，接口未返回有效时间时为 0。
// 批量查询的 BatchOpRet.Data 中，Expiration、TransitionToIA、TransitionToIntelligentTiering、
// TransitionToArchive 和 TransitionToDeepArchive 为 *int64，接口缺失或返回 0 时为 nil。
// TransitionToArchiveIR 保留 int64 类型，相同情况下返回 0。
// 0 和 nil 均表示接口未提供有效时间，不能据此判断对象没有生命周期规则或永不过期。
// 部分历史对象即使会执行生命周期操作，接口也可能不返回对应时间。
package storage
