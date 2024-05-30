package downloader

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	clientv1 "github.com/qiniu/go-sdk/v7/client"
	"github.com/qiniu/go-sdk/v7/internal/clientv2"
	"github.com/qiniu/go-sdk/v7/storagev2/backoff"
	"github.com/qiniu/go-sdk/v7/storagev2/chooser"
	"github.com/qiniu/go-sdk/v7/storagev2/downloader/destination"
	resumablerecorder "github.com/qiniu/go-sdk/v7/storagev2/downloader/resumable_recorder"
	"github.com/qiniu/go-sdk/v7/storagev2/resolver"
	"github.com/qiniu/go-sdk/v7/storagev2/retrier"
	"golang.org/x/sync/errgroup"
)

type (
	concurrentDownloader struct {
		concurrency       uint
		partSize          uint64
		client            clientv2.Client
		resumableRecorder resumablerecorder.ResumableRecorder
	}

	DownloaderOptions struct {
		Client   clientv2.Client   // HTTP 客户端，如果不配置则使用默认的 HTTP 客户端
		RetryMax int               // 最大重试次数
		Backoff  backoff.Backoff   // 重试时间间隔 v2，优先级高于 RetryInterval
		Resolver resolver.Resolver // 主备域名解析器
		Chooser  chooser.Chooser   // IP 选择器

		BeforeResolve func(*http.Request)                                         // 域名解析前回调函数
		AfterResolve  func(*http.Request, []net.IP)                               // 域名解析后回调函数
		ResolveError  func(*http.Request, error)                                  // 域名解析错误回调函数
		BeforeBackoff func(*http.Request, *retrier.RetrierOptions, time.Duration) // 退避前回调函数
		AfterBackoff  func(*http.Request, *retrier.RetrierOptions, time.Duration) // 退避后回调函数
		BeforeRequest func(*http.Request, *retrier.RetrierOptions)                // 请求前回调函数
		AfterResponse func(*http.Response, *retrier.RetrierOptions, error)        // 请求后回调函数
	}

	ConcurrentDownloaderOptions struct {
		DownloaderOptions
		Concurrency       uint
		PartSize          uint64
		ResumableRecorder resumablerecorder.ResumableRecorder
	}
)

func (options *DownloaderOptions) toSimpleRetryConfig() clientv2.SimpleRetryConfig {
	retryMax := options.RetryMax
	if retryMax <= 0 {
		retryMax = 9
	}
	return clientv2.SimpleRetryConfig{
		RetryMax: retryMax,
		Resolver: options.Resolver,
		Chooser:  options.Chooser,
		Backoff:  options.Backoff,
		ShouldRetry: func(req *http.Request, resp *http.Response, err error) bool {
			if err != nil {
				return retrier.IsErrorRetryable(err)
			}
			return resp.StatusCode >= 500
		},
		BeforeResolve: options.BeforeResolve,
		AfterResolve:  options.AfterResolve,
		ResolveError:  options.ResolveError,
		BeforeBackoff: options.BeforeBackoff,
		AfterBackoff:  options.AfterBackoff,
		BeforeRequest: options.BeforeRequest,
		AfterResponse: options.AfterResponse,
	}
}

func NewConcurrentDownloader(options *ConcurrentDownloaderOptions) DestinationDownloader {
	if options == nil {
		options = &ConcurrentDownloaderOptions{}
	}
	concurrency := options.Concurrency
	if concurrency == 0 {
		concurrency = 1
	}
	partSize := options.PartSize
	if partSize == 0 {
		partSize = 16 * 1024 * 1024
	}
	client := clientv2.NewClient(options.Client, clientv2.NewSimpleRetryInterceptor(options.toSimpleRetryConfig()))
	return &concurrentDownloader{concurrency, partSize, client, options.ResumableRecorder}
}

func (downloader concurrentDownloader) Download(ctx context.Context, urls []*url.URL, destination destination.Destination, options *DestinationDownloadOptions) (uint64, error) {
	if options == nil {
		options = &DestinationDownloadOptions{}
	}
	headResponse, err := headRequest(ctx, urls, options.Header, downloader.client)
	if err != nil {
		return 0, err
	}
	offset := uint64(0)
	switch headResponse.StatusCode {
	case http.StatusOK:
	case http.StatusPartialContent:
		var n1, n2 int64
		contentRange := headResponse.Header.Get("Content-Range")
		if _, err = fmt.Sscanf(contentRange, "bytes %d-%d/%d", &offset, &n1, &n2); err != nil {
			return 0, err
		}
	default:
		return 0, clientv1.ResponseError(headResponse)
	}
	etag := headResponse.Header.Get("Etag")
	if etag == "" {
		return 0, errors.New("no etag returned")
	}
	if headResponse.ContentLength < 0 { // 无法确定文件实际大小，发出一个请求下载整个文件，不再使用并行下载
		var progress func(uint64)
		if onDownloadingProgress := options.OnDownloadingProgress; onDownloadingProgress != nil {
			progress = func(downloaded uint64) {
				onDownloadingProgress(downloaded, 0)
			}
		}
		return downloadToPartReader(ctx, urls, options.Header, etag, downloader.client, destination, progress)
	}
	needToDownload := uint64(headResponse.ContentLength)

	var (
		readableMedium               resumablerecorder.ReadableResumableRecorderMedium
		writeableMedium              resumablerecorder.WriteableResumableRecorderMedium
		resumableRecorderOpenOptions *resumablerecorder.ResumableRecorderOpenOptions
	)
	if resumableRecorder := downloader.resumableRecorder; resumableRecorder != nil {
		var destinationKey string
		destinationKey, err = destination.DestinationKey()
		if err == nil && destinationKey != "" {
			downloadURLs := make([]string, len(urls))
			for i, url := range urls {
				downloadURLs[i] = url.String()
			}
			resumableRecorderOpenOptions = &resumablerecorder.ResumableRecorderOpenOptions{
				ETag:           etag,
				DestinationKey: destinationKey,
				PartSize:       downloader.partSize,
				TotalSize:      needToDownload,
				DownloadURLs:   downloadURLs,
			}
			readableMedium = resumableRecorder.OpenForReading(resumableRecorderOpenOptions)
			if readableMedium != nil {
				defer readableMedium.Close()
			}
		}
	}

	parts, err := destination.Slice(needToDownload, downloader.partSize, readableMedium)
	if err != nil {
		return 0, err
	}
	if readableMedium != nil {
		readableMedium.Close()
		readableMedium = nil
	}
	if resumableRecorder := downloader.resumableRecorder; resumableRecorder != nil && resumableRecorderOpenOptions != nil {
		writeableMedium = resumableRecorder.OpenForAppending(resumableRecorderOpenOptions)
		if writeableMedium == nil {
			writeableMedium = resumableRecorder.OpenForCreatingNew(resumableRecorderOpenOptions)
		}
		if writeableMedium != nil {
			defer writeableMedium.Close()
		}
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(int(downloader.concurrency))
	downloadingProgress := newDownloadingPartsProgress()
	for _, part := range parts {
		p := part
		g.Go(func() error {
			n, err := downloader.downloadToPart(ctx, urls, options.Header, etag, p, writeableMedium, func(downloaded uint64) {
				downloadingProgress.setPartDownloadingProgress(p.Offset(), downloaded)
				if onDownloadingProgress := options.OnDownloadingProgress; onDownloadingProgress != nil {
					onDownloadingProgress(downloadingProgress.totalDownloaded(), needToDownload)
				}
			})
			if n > 0 {
				downloadingProgress.partDownloaded(p.Offset(), n)
				if onDownloadingProgress := options.OnDownloadingProgress; onDownloadingProgress != nil {
					onDownloadingProgress(downloadingProgress.totalDownloaded(), needToDownload)
				}
			}
			return err
		})
	}
	err = g.Wait()
	if writeableMedium != nil {
		writeableMedium.Close()
		writeableMedium = nil
	}
	if resumableRecorder := downloader.resumableRecorder; resumableRecorder != nil && resumableRecorderOpenOptions != nil && err == nil {
		resumableRecorder.Delete(resumableRecorderOpenOptions)
	}
	return downloadingProgress.totalDownloaded(), err
}

func (downloader concurrentDownloader) downloadToPart(
	ctx context.Context, urls []*url.URL, headers http.Header, etag string,
	part destination.Part, writeableMedium resumablerecorder.WriteableResumableRecorderMedium, onDownloadingProgress func(downloaded uint64)) (uint64, error) {
	var (
		n        uint64
		err      error
		size     = part.Size()
		offset   = part.Offset()
		haveRead = part.HaveDownloaded()
	)
	for size > haveRead {
		n, err = downloadToPartReaderWithOffsetAndSize(ctx, urls, headers, etag, offset+haveRead, size-haveRead, downloader.client, part, onDownloadingProgress)
		if n > 0 {
			haveRead += n
			continue
		}
		break
	}
	if haveRead > 0 && writeableMedium != nil {
		writeableMedium.Write(&resumablerecorder.ResumableRecord{
			Offset:      offset,
			PartSize:    size,
			PartWritten: haveRead,
		})
	}
	return haveRead, err
}

func downloadToPartReaderWithOffsetAndSize(
	ctx context.Context, urls []*url.URL, headers http.Header, etag string, offset, size uint64,
	client clientv2.Client, part destination.PartReader, onDownloadingProgress func(downloaded uint64)) (uint64, error) {
	headers = cloneHeader(headers)
	headers.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+size-1))
	return _downloadToPartReader(ctx, urls, headers, etag, client, part, onDownloadingProgress)
}

func downloadToPartReader(
	ctx context.Context, urls []*url.URL, headers http.Header, etag string,
	client clientv2.Client, part destination.PartReader, onDownloadingProgress func(downloaded uint64)) (uint64, error) {
	headers = cloneHeader(headers)
	setAcceptGzip(headers)
	return _downloadToPartReader(ctx, urls, headers, etag, client, part, onDownloadingProgress)
}

func _downloadToPartReader(
	ctx context.Context, urls []*url.URL, headers http.Header, etag string,
	client clientv2.Client, part destination.PartReader, onDownloadingProgress func(downloaded uint64)) (uint64, error) {
	var (
		response *http.Response
		n        uint64
		err      error
	)

	for _, url := range urls {
		req := http.Request{
			Method: http.MethodGet,
			URL:    url,
			Header: headers,
			Body:   http.NoBody,
		}
		if response, err = client.Do(req.WithContext(ctx)); err != nil {
			if errorInfo, ok := err.(*clientv1.ErrorInfo); ok && errorInfo.Code < 500 {
				return 0, err
			}
		} else if response.Header.Get("Etag") == etag {
			var (
				bodyReader io.Reader = response.Body
				bodyCloser io.Closer = response.Body
			)
			switch response.Header.Get("Content-Encoding") {
			case "gzip":
				if bodyReader, err = gzip.NewReader(bodyReader); err != nil {
					bodyCloser.Close()
					return 0, err
				}
				fallthrough
			case "":
				n, err = part.CopyFrom(bodyReader, onDownloadingProgress)
				bodyCloser.Close()
				if n > 0 {
					return n, err
				}
			default:
				err = errors.New("unrecognized content-encoding")
				bodyCloser.Close()
			}
		} else {
			err = errors.New("etag dismatch")
		}
	}
	return 0, err
}

func headRequest(ctx context.Context, urls []*url.URL, headers http.Header, client clientv2.Client) (response *http.Response, err error) {
	headers = cloneHeader(headers)
	setAcceptGzip(headers)

	for _, url := range urls {
		req := http.Request{
			Method: http.MethodHead,
			URL:    url,
			Header: headers,
			Body:   http.NoBody,
		}
		if response, err = client.Do(req.WithContext(ctx)); err != nil {
			if errorInfo, ok := err.(*clientv1.ErrorInfo); ok && errorInfo.Code < 500 {
				return
			}
		} else {
			break
		}
	}
	if response != nil && response.Body != nil {
		response.Body.Close()
	}
	return
}

func cloneHeader(h http.Header) http.Header {
	if h == nil {
		return make(http.Header)
	}

	nv := 0
	for _, vv := range h {
		nv += len(vv)
	}
	sv := make([]string, nv)
	h2 := make(http.Header, len(h))
	for k, vv := range h {
		if vv == nil {
			h2[k] = nil
			continue
		}
		n := copy(sv, vv)
		h2[k] = sv[:n:n]
		sv = sv[n:]
	}
	return h2
}

func setAcceptGzip(headers http.Header) {
	headers.Set("Accept-Encoding", "gzip, deflate")
}

type downloadingPartsProgress struct {
	downloaded  uint64
	downloading map[uint64]uint64
	lock        sync.Mutex
}

func newDownloadingPartsProgress() *downloadingPartsProgress {
	return &downloadingPartsProgress{
		downloading: make(map[uint64]uint64),
	}
}

func (progress *downloadingPartsProgress) setPartDownloadingProgress(offset, downloaded uint64) {
	progress.lock.Lock()
	defer progress.lock.Unlock()

	progress.downloading[offset] = downloaded
}

func (progress *downloadingPartsProgress) partDownloaded(offset, partSize uint64) {
	progress.lock.Lock()
	defer progress.lock.Unlock()

	delete(progress.downloading, offset)
	progress.downloaded += partSize
}

func (progress *downloadingPartsProgress) totalDownloaded() uint64 {
	progress.lock.Lock()
	defer progress.lock.Unlock()

	downloaded := progress.downloaded
	for _, b := range progress.downloading {
		downloaded += b
	}
	return downloaded
}
