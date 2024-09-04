package downloader

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/downloader/destination"
	"github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/objects"
	"github.com/qiniu/go-sdk/v7/storagev2/region"
	"golang.org/x/sync/errgroup"
)

type (
	// 下载管理器
	DownloadManager struct {
		destinationDownloader    DestinationDownloader
		objectsManager           *objects.ObjectsManager
		downloadURLsProvider     DownloadURLsProvider
		downloadURLsProviderOnce sync.Once
		options                  httpclient.Options
	}

	// 下载管理器选项
	DownloadManagerOptions struct {
		// HTTP 客户端选项
		httpclient.Options

		// 目标下载器
		DestinationDownloader DestinationDownloader

		// 分片列举版本，如果不填写，默认为 V1
		ListerVersion objects.ListerVersion

		// 下载 URL 生成器
		DownloadURLsProvider DownloadURLsProvider
	}

	// 对象下载参数
	ObjectOptions struct {
		DestinationDownloadOptions
		GenerateOptions

		// 下载 URL 生成器
		DownloadURLsProvider DownloadURLsProvider
	}

	// 目录下载参数
	DirectoryOptions struct {
		// 是否使用 HTTP 协议，默认为不使用
		UseInsecureProtocol bool

		// 空间名称
		BucketName string

		// 对象前缀
		ObjectPrefix string

		// 下载并发度
		ObjectConcurrency int

		// 下载 URL 生成器
		DownloadURLsProvider DownloadURLsProvider

		// 下载前回调函数
		BeforeObjectDownload func(objectName string, objectOptions *ObjectOptions)

		// 下载进度
		OnDownloadingProgress func(objectName string, progress *DownloadingProgress)

		// 对象下载成功后回调
		OnObjectDownloaded func(objectName string, info *DownloadedObjectInfo)

		// 是否下载指定对象
		ShouldDownloadObject func(objectName string) bool

		// 分隔符，默认为 /
		PathSeparator string
	}

	// 已经下载的对象信息
	DownloadedObjectInfo struct {
		Size uint64 // 对象大小
	}

	writeSeekCloser struct {
		w io.Writer
	}
)

// 创建下载管理器
func NewDownloadManager(options *DownloadManagerOptions) *DownloadManager {
	if options == nil {
		options = &DownloadManagerOptions{}
	}
	destinationDownloader := options.DestinationDownloader
	if destinationDownloader == nil {
		destinationDownloader = NewConcurrentDownloader(nil)
	}
	objectsManager := objects.NewObjectsManager(&objects.ObjectsManagerOptions{
		Options:       options.Options,
		ListerVersion: options.ListerVersion,
	})
	return &DownloadManager{
		destinationDownloader: destinationDownloader,
		objectsManager:        objectsManager,
		options:               options.Options,
	}
}

// 下载对象到文件
func (downloadManager *DownloadManager) DownloadToFile(ctx context.Context, objectName, filePath string, options *ObjectOptions) (uint64, error) {
	dest, err := destination.NewFileDestination(filePath)
	if err != nil {
		return 0, err
	}
	defer dest.Close()
	return downloadManager.downloadToDestination(ctx, objectName, dest, options)
}

// 下载对象到 io.Writer
func (downloadManager *DownloadManager) DownloadToWriter(ctx context.Context, objectName string, writer io.Writer, options *ObjectOptions) (uint64, error) {
	var dest destination.Destination
	if writeAtCloser, ok := writer.(destination.WriteAtCloser); ok {
		dest = destination.NewWriteAtCloserDestination(writeAtCloser, "")
	} else {
		dest = destination.NewWriteCloserDestination(&writeSeekCloser{writer}, "")
	}
	defer dest.Close()
	return downloadManager.downloadToDestination(ctx, objectName, dest, options)
}

func (downloadManager *DownloadManager) downloadToDestination(ctx context.Context, objectName string, dest destination.Destination, options *ObjectOptions) (uint64, error) {
	if options == nil {
		options = &ObjectOptions{}
	}
	downloadURLsProvider := options.DownloadURLsProvider
	if downloadURLsProvider == nil {
		if err := downloadManager.initDownloadURLsProvider(ctx); err != nil {
			return 0, err
		}
	}
	if downloadURLsProvider == nil {
		downloadURLsProvider = downloadManager.downloadURLsProvider
	}
	if downloadURLsProvider == nil {
		return 0, errors.MissingRequiredFieldError{Name: "DownloadURLsProvider"}
	}
	urls, err := downloadURLsProvider.GetURLsIter(ctx, objectName, &options.GenerateOptions)
	if err != nil {
		return 0, err
	}
	return downloadManager.destinationDownloader.Download(ctx, urls, dest, &options.DestinationDownloadOptions)
}

// 下载目录
func (downloadManager *DownloadManager) DownloadDirectory(ctx context.Context, targetDirPath string, options *DirectoryOptions) error {
	var err error

	if options == nil {
		options = &DirectoryOptions{}
	}
	if options.BucketName == "" {
		return &errors.MissingRequiredFieldError{Name: "BucketName"}
	}
	objectConcurrency := options.ObjectConcurrency
	if objectConcurrency == 0 {
		objectConcurrency = 4
	}
	pathSeparator := options.PathSeparator
	if pathSeparator == "" {
		pathSeparator = "/"
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(objectConcurrency)

	lister := downloadManager.objectsManager.Bucket(options.BucketName).List(ctx, &objects.ListObjectsOptions{
		Prefix: options.ObjectPrefix,
	})
	defer lister.Close()

	var object objects.ObjectDetails
	for lister.Next(&object) {
		objectName := object.Name
		if options.ShouldDownloadObject != nil && !options.ShouldDownloadObject(objectName) {
			continue
		}

		relativePath := strings.TrimPrefix(objectName, options.ObjectPrefix)
		if pathSeparator != string(filepath.Separator) {
			relativePath = strings.Replace(relativePath, pathSeparator, string(filepath.Separator), -1)
		}
		fullPath := filepath.Join(targetDirPath, relativePath)
		if relativePath == "" || strings.HasSuffix(relativePath, string(filepath.Separator)) {
			if err = os.MkdirAll(fullPath, 0700); err != nil {
				return err
			}
		} else {
			if err = os.MkdirAll(filepath.Dir(fullPath), 0700); err != nil {
				return err
			}
			g.Go(func() error {
				var destinationDownloadOptions DestinationDownloadOptions
				if onDownloadingProgress := options.OnDownloadingProgress; onDownloadingProgress != nil {
					destinationDownloadOptions.OnDownloadingProgress = func(progress *DownloadingProgress) {
						onDownloadingProgress(objectName, progress)
					}
				}
				objectOptions := ObjectOptions{
					DestinationDownloadOptions: destinationDownloadOptions,
					GenerateOptions: GenerateOptions{
						BucketName:          options.BucketName,
						UseInsecureProtocol: options.UseInsecureProtocol,
					},
					DownloadURLsProvider: options.DownloadURLsProvider,
				}
				if options.BeforeObjectDownload != nil {
					options.BeforeObjectDownload(objectName, &objectOptions)
				}
				n, err := downloadManager.DownloadToFile(ctx, objectName, fullPath, &objectOptions)
				if err == nil && options.OnObjectDownloaded != nil {
					options.OnObjectDownloaded(objectName, &DownloadedObjectInfo{Size: n})
				}
				return err
			})
		}
	}
	if err = lister.Error(); err != nil {
		return err
	}
	return g.Wait()
}

func (downloadManager *DownloadManager) initDownloadURLsProvider(ctx context.Context) (err error) {
	if downloadManager.downloadURLsProvider == nil {
		if credentialsProvider := downloadManager.getCredentialsProvider(); credentialsProvider != nil {
			downloadManager.downloadURLsProviderOnce.Do(func() {
				var creds *credentials.Credentials
				creds, err = credentialsProvider.Get(ctx)
				if err != nil {
					return
				}
				bucketRegionsQueryOptions := region.BucketRegionsQueryOptions{
					UseInsecureProtocol: downloadManager.options.UseInsecureProtocol,
					Client:              downloadManager.options.BasicHTTPClient,
					Resolver:            downloadManager.options.Resolver,
					Chooser:             downloadManager.options.Chooser,
					BeforeResolve:       downloadManager.options.BeforeResolve,
					AfterResolve:        downloadManager.options.AfterResolve,
					ResolveError:        downloadManager.options.ResolveError,
					BeforeBackoff:       downloadManager.options.BeforeBackoff,
					AfterBackoff:        downloadManager.options.AfterBackoff,
					BeforeRequest:       downloadManager.options.BeforeRequest,
					AfterResponse:       downloadManager.options.AfterResponse,
					AccelerateUploading: downloadManager.options.AccelerateUploading,
				}
				if hostRetryConfig := downloadManager.options.HostRetryConfig; hostRetryConfig != nil {
					bucketRegionsQueryOptions.RetryMax = hostRetryConfig.RetryMax
					bucketRegionsQueryOptions.Backoff = hostRetryConfig.Backoff
				}
				downloadManager.downloadURLsProvider = SignURLsProvider(
					NewDefaultSrcURLsProvider(creds.AccessKey, &DefaultSrcURLsProviderOptions{
						BucketRegionsQueryOptions: bucketRegionsQueryOptions,
						BucketHosts:               httpclient.DefaultBucketHosts(),
					}),
					NewCredentialsSigner(creds),
					&SignOptions{TTL: 3 * time.Minute},
				)
			})
		}
	}
	return
}

func (downloadManager *DownloadManager) getCredentialsProvider() credentials.CredentialsProvider {
	credentialsProvider := downloadManager.options.Credentials
	if credentialsProvider == nil {
		if defaultCreds := credentials.Default(); defaultCreds != nil {
			credentialsProvider = defaultCreds
		}
	}
	return credentialsProvider
}

func (w *writeSeekCloser) Write(p []byte) (int, error) {
	return w.w.Write(p)
}

func (w *writeSeekCloser) Seek(offset int64, whence int) (int64, error) {
	if seeker, ok := w.w.(io.Seeker); ok {
		return seeker.Seek(offset, whence)
	}
	return 0, syscall.ESPIPE
}

func (w *writeSeekCloser) Close() error {
	if closer, ok := w.w.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
