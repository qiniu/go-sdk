# Changelog


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
