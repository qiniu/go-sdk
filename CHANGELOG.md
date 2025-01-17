# Changelog

## 7.25.2

* 优化
  * 简化 Degbug 模式下日志输出
* 修复
  * storage 包上传接口配置 UC 域名未生效
  * 处理 Windows 平台和 gomobile 因 go-sysinfo 依赖导致的兼容性问题
  * 修复删除用户组策略问题
  * 修复 bucket/v4 获取 buckets 不全问题


## 7.25.1

* 修复
  * QVS 录制合并接口

## 7.25.0

* 新增
  * CDN API 支持计量数据类型

## 7.24.0
* 新增
  * 支持工作流模版
  * 在 [storage](github.com/qiniu/go-sdk/v7/storage) 包中支持上传加速
* 优化
  * 超时连接重试机制
  * 域名解析请求机制
  * 域名解析缓存过期机制

## 7.23.0
* 新增
  * 支持上传加速
  * 多媒体处理库 [media](github.com/qiniu/go-sdk/v7/media/apis) 包，提供多媒体处理接口
  * IAM [iam](github.com/qiniu/go-sdk/v7/iam/apis) 包，提供权限管理系统接口
  * 获取下载 URL 的实用方法
  * 提供文件夹分享接口
* 修复
  * `downloader.DownloadDirectory` 下载根目录出错的 Bug

## 7.22.0
* 新增
  * 新版存储客户端库 storagev2 包，包含
    * 新的 [uploader](github.com/qiniu/go-sdk/v7/storagev2/uploader) 包设计，提供更好的对象上传接口
    * 新的 [downloader](github.com/qiniu/go-sdk/v7/storagev2/downloader) 包设计，提供更好的对象下载接口
    * 新的 [objects](github.com/qiniu/go-sdk/v7/storagev2/objects) 包设计，提供更好的对象管理接口
  * 为 HTTP 客户端新增防劫持能力
* 调整
  * 查询区域主备域名
* 修复
  * Go SDK v7.21.1 可能出现崩溃的问题

## 7.21.1
* 修复
  * 修复通过 ResumeUploaderV2 上传的文件无法正确设置 Content-Type 的问题

## 7.21.0
* 新增
  * 统一环境变量和配置文件
  * 新版存储客户端库 storagev2 包，包含
    * 新的 [resolver](github.com/qiniu/go-sdk/v7/storagev2/resolver) 包设计，可以为 HTTP 客户端指定域名解析器
    * 新的 [chooser](github.com/qiniu/go-sdk/v7/storagev2/chooser) 包设计，可以为 HTTP 客户端选择成功率较高的 IP 地址
    * 新的 [backoff](github.com/qiniu/go-sdk/v7/storagev2/backoff) 包设计，可以为 HTTP 客户端指定重试退避时长
    * 新的 [retrier](github.com/qiniu/go-sdk/v7/storagev2/retrier) 包设计，可以为 HTTP 客户端指定重试策略
    * 为 HTTP 客户端新增 uplog 机制

## 7.20.2
* 修复
  * 修复通过 FormUploader 上传的文件无法正确设置 Content-Type 的问题

## 7.20.1
* 新增
  * 获取区域 API 支持单独配置 UC 域名
  * BucketManager List Bucket 接口支持返回文件的 parts

## 7.20.0
* 新增
  * 新版存储客户端库 storagev2 包，包含
    * 新的 [region](github.com/qiniu/go-sdk/v7/storagev2/region) 包设计，使用新的 `RegionsProvider` 和 `EndpointsProvider` 接口替代之前的 `Region` 结构体
    * 新的 [credentials](github.com/qiniu/go-sdk/v7/storagev2/credentials) 包设计，使用新的 `CredentialsProvider` 接口替代之前的 `Credentials` 结构体
    * 新的 [uptoken](github.com/qiniu/go-sdk/v7/storagev2/uptoken) 包设计，使用新的 `PutPolicy` 结构体，使用新的 `UpTokenProvider` 接口替代之前用字符串表示上传凭证的方式
    * 新的 [apis](github.com/qiniu/go-sdk/v7/storagev2/apis) 包设计，使用新的 `Storage` 结构体调用七牛 Kodo API

## 7.19.1
* 新增
  * PutPolicy 添加 transform 相关字段

## 7.19.0
* 新增
  * 支持归档直读存储


## 7.18.1
* 调整
  * PutPolicy 的 DetectMime 支持配置为 -1

## 7.18.0
* 新增
  * BucketManager  支持下载
  * BucketManager 支持获取和修改对象 meta data
  * 上传和下载支持单链限速
* 优化
  * 区域缓存信息 ID 和查询域名相关

## 7.17.1
* 优化
  * 调整在获取 Bucket 所在区域服务域名时的主备域名顺序
* 调整
  * 移除内置的亚太-首尔区域

## 7.17.0
* 优化
  * 对象存储，UC 服务相关请求支持主备重试

## 7.16.0
* 新增
    * 对象存储，`BucketManager` `BucketsV4` 获取该用户的指定区域内的空间信息，注意该 API 以分页形式返回 Bucket 列表
    * 对象存储，`BucketManager` `SetRemark` 设置空间的备注信息
* 更新
    * 对象存储，`BucketManager` `GetBucketInfo` 返回空间的备注信息

## 7.15.0
* 优化
    * 表单上传 Put 方法的 PutExtra 参数，支持传入 nil
    * Bucket 镜像源/配额方法内部请求使用 UC 域名
    * `BucketManager` `ListBucket` 和 `ListBucketContext` 方法内部接口由 /v2/list 调用调整为 /list
* 新增
    * `BucketManager` 新增批量方法 `BatchWithContext`
    * `BucketManager` 增加 Bucket 列举方法 `ListFileWithContext`

## 7.14.0
* Go SDK  对七牛回调请求的鉴权验证函数支持 Qiniu 签名
* UploadMananger 支持双活区域
* 优化分片上传 ctx 超时检测
* 表单 / 分片 v1 / 分片 v2 上传支持主备域名重试逻辑。
* 表单上传支持配置重试次数
* 处理 Config 配置固定 Region， 但不生效的问题(配置 Zone 生效)
* 移除雾存储区域：华东一区
* 增加亚太-首尔区域固定 Region

## 7.13.0
* 对象存储，管理类 API 发送请求时增加 [X-Qiniu-Date](https://developer.qiniu.com/kodo/3924/common-request-headers) （生成请求的时间） header

## 7.12.1
* 对象存储，补充 Stat API 可查看对象元数据信息

## 7.12.0
* 对象存储，新增支持 [深度归档存储类型](https://developer.qiniu.com/kodo/3956/kodo-category#deep_archive)
* 对象存储，全面支持 Qiniu 签名

## 7.11.1
* 优化容器环境下 pod 当前内存工作集 (working set) 使用量

## 7.11.0
* 新增直播云服务端管理能力，包括：直播空间管理、域名管理、推拉流地址管理、直播流管理和统计数据查询 API


## 7.10.1
* 优化了分片上传内存占用
* 修复部分已知问题

## 7.10.0
* 增加了 PutWithoutKeyAndSize API，在上传时支持可不指定 size 和 key
* 修复了 已知 UcQuery 解析问题
## 7.9.8
* 补充了查询 object 元信息返回部分字段

## 7.9.7
* 修复了表单上传 FormUploader 在内部重试情况下的已知问题

## 7.9.6
* 在需要指定存储服务 host 情况下兼容了只配置域名和同时指定域名和访问 protocol 的问题

## 7.9.5
优化几个已知小问题
* 支持指定空间管理域名，默认是公有云地址
* 支持下载 escape 编码文件
* 优化对一些错误情况处理

## 7.9.4
* 兼容不同格式上传token

## 7.9.3
* 修复在复用上传token时，过期时间叠加问题

## 7.9.2
* UploadPartInfo 结构体公开使用，可用于定制分片上传过程
* 保持兼容支持上传API extra.UpHost参数

## 7.9.1
* 修复buckets api 已知问题

## 7.9.0
* 从 github.com/qiniu/api.v7 迁移至 github.com/qiniu/go-sdk
