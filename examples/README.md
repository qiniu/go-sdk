# Examples

每个子目录包含一个独立的示例程序，可通过 `go run ./examples/<name>/` 运行。

| 目录 | 说明 |
|------|------|
| bucket_image_unimage | 设置和取消空间镜像存储 |
| cdn_create_timestamp_antileech_url | 生成带时间戳的防盗链 URL |
| cdn_get_bandwidth_data | 查询 CDN 带宽数据 |
| cdn_get_flux_data | 查询 CDN 流量数据 |
| cdn_get_log_list | 获取 CDN 日志列表 |
| cdn_prefetch_urls | CDN 预热指定 URL |
| cdn_refresh_urls_and_dirs | CDN 刷新 URL 和目录 |
| create_uptoken | 生成上传令牌 |
| form_upload_simple | 表单方式上传文件 |
| prefop | 查询持久化处理状态 |
| resume_upload_advanced | 断点续传（支持进度记录） |
| resume_upload_simple | 断点续传（基础用法） |
| rs_async_fetch | 异步抓取网络资源到空间 |
| rs_batch_change_mime | 批量修改文件 MIME 类型 |
| rs_batch_change_type | 批量修改文件存储类型 |
| rs_batch_copy | 批量复制文件 |
| rs_batch_delete | 批量删除文件 |
| rs_batch_delete_after_days | 批量设置文件定时删除 |
| rs_batch_move | 批量移动文件 |
| rs_batch_stat | 批量查询文件信息 |
| rs_change_mime | 修改单个文件 MIME 类型 |
| rs_change_type | 修改单个文件存储类型 |
| rs_copy | 复制单个文件 |
| rs_delete | 删除单个文件 |
| rs_delete_after_days | 设置单个文件定时删除 |
| rs_download | 生成公开和私有下载 URL |
| rs_fetch | 抓取网络资源到空间 |
| rs_list_bucket | 列举空间中所有文件 |
| rs_list_bucket_context | 使用 Context 分页列举文件 |
| rs_list_files | 循环分页列举文件列表 |
| rs_move | 移动单个文件 |
| rs_prefetch | 预取文件到边缘节点 |
| rs_stat | 查询单个文件信息 |
| rtc | 实时通讯应用管理 |
| video_pfop | 视频转码和截图处理 |
