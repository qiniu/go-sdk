// Package downloader 提供七牛云对象存储的下载管理。
//
// [DownloadManager] 是推荐的下载入口，支持文件下载、流式下载和目录下载。
// 支持并发分片下载、断点续传和多种 URL 生成策略。
//
// # 基本用法
//
//	downloadManager := downloader.NewDownloadManager(&downloader.DownloadManagerOptions{
//	    Options: http_client.Options{Credentials: cred},
//	})
//
//	// 下载到文件（通过 ObjectOptions 指定 URL 生成策略）
//	size, err := downloadManager.DownloadToFile(ctx, "my-file.txt", "/tmp/file.txt", &downloader.ObjectOptions{
//	    DownloadURLsProvider: downloader.SignURLsProvider(
//	        downloader.NewStaticDomainBasedURLsProvider([]string{"cdn.example.com"}),
//	        downloader.NewCredentialsSigner(cred),
//	        nil,
//	    ),
//	})
//
//	// 下载到 Writer
//	size, err := downloadManager.DownloadToWriter(ctx, "my-file.txt", writer, nil)
//
//	// 下载整个目录
//	err := downloadManager.DownloadDirectory(ctx, "/tmp/backup", &downloader.DirectoryOptions{
//	    BucketName: "my-bucket",
//	})
//
// # URL 生成策略
//
// 通过 [DownloadURLsProvider] 接口控制下载 URL 的生成方式：
//
//   - [NewStaticDomainBasedURLsProvider]: 使用固定域名
//   - [NewDefaultSrcURLsProvider]: 自动查询源站域名
//   - [NewDomainsQueryURLsProvider]: 通过 API 查询域名
//   - [CombineDownloadURLsProviders]: 组合多个 Provider
//
// # URL 签名
//
// 使用 [Signer] 接口为下载 URL 添加认证签名：
//
//	signer := downloader.NewCredentialsSigner(cred)
//	signedProvider := downloader.SignURLsProvider(urlsProvider, signer, nil)
//
// # 并发下载
//
// 使用 [NewConcurrentDownloader] 通过 HTTP Range 请求并发下载大文件：
//
//	dest := downloader.NewConcurrentDownloader(&downloader.ConcurrentDownloaderOptions{
//	    Concurrency: 4,
//	    PartSize:    4 * 1024 * 1024,
//	})
package downloader
