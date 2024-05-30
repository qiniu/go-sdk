package downloader

import (
	"context"
	"io"
	"syscall"

	"github.com/qiniu/go-sdk/v7/storagev2/downloader/destination"
	"github.com/qiniu/go-sdk/v7/storagev2/errors"
)

type (
	DownloadManager struct {
		destinationDownloader DestinationDownloader
	}

	DownloadManagerOptions struct {
		// 目标下载器
		DestinationDownloader DestinationDownloader
	}

	// 对象下载参数
	ObjectOptions struct {
		DestinationDownloadOptions
		GenerateOptions
		SignOptions

		// 下载 URL 生成器
		DownloadURLsGenerator DownloadURLsGenerator

		// 下载 URL 签名
		Signer Signer
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
	return &DownloadManager{destinationDownloader}
}

func (downloadManager *DownloadManager) DownloadToFile(ctx context.Context, objectName, filePath string, options *ObjectOptions) error {
	dest, err := destination.NewFileDestination(filePath)
	if err != nil {
		return err
	}
	defer dest.Close()
	return downloadManager.downloadToDestination(ctx, objectName, dest, options)
}

func (downloadManager *DownloadManager) DownloadToWriter(ctx context.Context, objectName string, writer io.Writer, options *ObjectOptions) error {
	var dest destination.Destination
	if writeAtCloser, ok := writer.(destination.WriteAtCloser); ok {
		dest = destination.NewWriteAtCloserDestination(writeAtCloser, "")
	} else {
		dest = destination.NewWriteCloserDestination(&writeSeekCloser{writer}, "")
	}
	defer dest.Close()
	return downloadManager.downloadToDestination(ctx, objectName, dest, options)
}

func (downloadManager *DownloadManager) downloadToDestination(ctx context.Context, objectName string, dest destination.Destination, options *ObjectOptions) error {
	if options == nil {
		options = &ObjectOptions{}
	}
	downloadURLsGenerator := options.DownloadURLsGenerator
	if downloadURLsGenerator == nil {
		return errors.MissingRequiredFieldError{Name: "DownloadURLsGenerator"}
	}
	signer := options.Signer
	urls, err := downloadURLsGenerator.GenerateURLs(ctx, objectName, &options.GenerateOptions)
	if err != nil {
		return err
	}
	if signer != nil {
		for _, u := range urls {
			if err = signer.Sign(ctx, u, &options.SignOptions); err != nil {
				return err
			}
		}
	}
	_, err = downloadManager.destinationDownloader.Download(ctx, urls, dest, &options.DestinationDownloadOptions)
	return err
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
