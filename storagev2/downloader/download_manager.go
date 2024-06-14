package downloader

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/qiniu/go-sdk/v7/storagev2/downloader/destination"
	"github.com/qiniu/go-sdk/v7/storagev2/errors"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/objects"
	"golang.org/x/sync/errgroup"
)

type (
	DownloadManager struct {
		destinationDownloader DestinationDownloader
		objectsManager        *objects.ObjectsManager
	}

	DownloadManagerOptions struct {
		// HTTP 客户端选项
		httpclient.Options

		// 目标下载器
		DestinationDownloader DestinationDownloader

		// 分片列举版本，如果不填写，默认为 V1
		ListerVersion objects.ListerVersion
	}

	// 对象下载参数
	ObjectOptions struct {
		DestinationDownloadOptions
		GenerateOptions

		// 下载 URL 生成器
		DownloadURLsGenerator DownloadURLsProvider
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
		DownloadURLsGenerator DownloadURLsProvider

		// 下载前回调函数
		BeforeObjectDownload func(objectName string, objectOptions *ObjectOptions)

		// 下载进度
		OnDownloadingProgress func(objectName string, downloaded, totalSize uint64)

		// 对象下载成功后回调
		OnObjectDownloaded func(objectName string, size uint64)

		// 是否下载指定对象
		ShouldDownloadObject func(objectName string) bool
	}

	writeSeekCloser struct {
		w io.Writer
	}
)

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
	return &DownloadManager{destinationDownloader, objectsManager}
}

func (downloadManager *DownloadManager) DownloadToFile(ctx context.Context, objectName, filePath string, options *ObjectOptions) (uint64, error) {
	dest, err := destination.NewFileDestination(filePath)
	if err != nil {
		return 0, err
	}
	defer dest.Close()
	return downloadManager.downloadToDestination(ctx, objectName, dest, options)
}

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
	downloadURLsGenerator := options.DownloadURLsGenerator
	if downloadURLsGenerator == nil {
		return 0, errors.MissingRequiredFieldError{Name: "DownloadURLsGenerator"}
	}
	urls, err := downloadURLsGenerator.GetURLs(ctx, objectName, &options.GenerateOptions)
	if err != nil {
		return 0, err
	}
	n, err := downloadManager.destinationDownloader.Download(ctx, urls, dest, &options.DestinationDownloadOptions)
	return n, err
}

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
		objectConcurrency = 1
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
		relativePath := strings.TrimPrefix(objectName, options.ObjectPrefix)
		fullPath := filepath.Join(targetDirPath, relativePath)
		if strings.HasSuffix(relativePath, "/") {
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
					destinationDownloadOptions.OnDownloadingProgress = func(downloaded, totalSize uint64) {
						onDownloadingProgress(objectName, downloaded, totalSize)
					}
				}
				objectOptions := ObjectOptions{
					DestinationDownloadOptions: destinationDownloadOptions,
					GenerateOptions: GenerateOptions{
						BucketName:          options.BucketName,
						UseInsecureProtocol: options.UseInsecureProtocol,
					},
					DownloadURLsGenerator: options.DownloadURLsGenerator,
				}
				if options.ShouldDownloadObject != nil && !options.ShouldDownloadObject(objectName) {
					return nil
				}
				if options.BeforeObjectDownload != nil {
					options.BeforeObjectDownload(objectName, &objectOptions)
				}
				n, err := downloadManager.DownloadToFile(ctx, objectName, fullPath, &objectOptions)
				if err == nil && options.OnObjectDownloaded != nil {
					options.OnObjectDownloaded(objectName, n)
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
