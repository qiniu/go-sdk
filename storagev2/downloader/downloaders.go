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
	"strings"
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

	// 下载器选项
	DownloaderOptions struct {
		Client   clientv2.Client   // HTTP 客户端，如果不配置则使用默认的 HTTP 客户端
		RetryMax int               // 最大重试次数，如果不配置，默认为 9
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

	// 并发下载器选项
	ConcurrentDownloaderOptions struct {
		DownloaderOptions
		Concurrency       uint                                // 并发度
		PartSize          uint64                              // 分片大小
		ResumableRecorder resumablerecorder.ResumableRecorder // 可恢复记录仪
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

// 创建并发下载器
func NewConcurrentDownloader(options *ConcurrentDownloaderOptions) DestinationDownloader {
	if options == nil {
		options = &ConcurrentDownloaderOptions{}
	}
	concurrency := options.Concurrency
	if concurrency == 0 {
		concurrency = 4
	}
	partSize := options.PartSize
	if partSize == 0 {
		partSize = 16 * 1024 * 1024
	}
	client := clientv2.NewClient(options.Client, clientv2.NewSimpleRetryInterceptor(options.toSimpleRetryConfig()), retryWhenTokenOutOfDateInterceptor{})
	return &concurrentDownloader{concurrency, partSize, client, options.ResumableRecorder}
}

func (downloader concurrentDownloader) Download(ctx context.Context, urlsIter URLsIter, dest destination.Destination, options *DestinationDownloadOptions) (uint64, error) {
	if options == nil {
		options = &DestinationDownloadOptions{}
	}
	headResponse, err := headRequest(ctx, urlsIter, options.Header, downloader.client)
	if err != nil {
		return 0, err
	} else if headResponse == nil {
		return 0, errors.New("no url tried")
	}
	if onResponseHeader := options.OnResponseHeader; onResponseHeader != nil {
		onResponseHeader(headResponse.Header)
	}

	var offset uint64
	switch headResponse.StatusCode {
	case http.StatusOK:
	case http.StatusPartialContent:
		var unused1, unused2 int64
		contentRange := headResponse.Header.Get("Content-Range")
		if _, err = fmt.Sscanf(contentRange, "bytes %d-%d/%d", &offset, &unused1, &unused2); err != nil {
			return 0, err
		}
	default:
		return 0, clientv1.ResponseError(headResponse)
	}
	etag := parseEtag(headResponse.Header.Get("Etag"))
	if headResponse.ContentLength < 0 || // 无法确定文件实际大小，发出一个请求下载整个文件，不再使用并行下载
		headResponse.Header.Get("Accept-Ranges") != "bytes" { // 必须返回 Accept-Ranges 头，否则不认为可以分片下载
		var progress func(uint64)
		if onDownloadingProgress := options.OnDownloadingProgress; onDownloadingProgress != nil {
			progress = func(downloaded uint64) {
				onDownloadingProgress(&DownloadingProgress{Downloaded: downloaded})
			}
		}
		return downloadToPartReader(ctx, urlsIter, etag, options.Header, downloader.client, dest, progress)
	}
	needToDownload := uint64(headResponse.ContentLength)

	var (
		readableMedium            resumablerecorder.ReadableResumableRecorderMedium
		writeableMedium           resumablerecorder.WriteableResumableRecorderMedium
		resumableRecorderOpenArgs *resumablerecorder.ResumableRecorderOpenArgs
	)
	if resumableRecorder := downloader.resumableRecorder; resumableRecorder != nil {
		var destinationID string
		destinationID, err = dest.DestinationID()
		if err == nil && destinationID != "" {
			resumableRecorderOpenArgs = &resumablerecorder.ResumableRecorderOpenArgs{
				ETag:          etag,
				DestinationID: destinationID,
				PartSize:      downloader.partSize,
				TotalSize:     needToDownload,
				Offset:        offset,
			}
			readableMedium = resumableRecorder.OpenForReading(resumableRecorderOpenArgs)
			if readableMedium != nil {
				defer readableMedium.Close()
			} else if file := dest.GetFile(); file != nil {
				if err = file.Truncate(0); err != nil { // 无法恢复进度，目标文件清空
					return 0, err
				}
			}
		}
	}

	parts, err := dest.Split(needToDownload, downloader.partSize, &destination.SplitOptions{Medium: readableMedium})
	if err != nil {
		return 0, err
	}
	if readableMedium != nil {
		readableMedium.Close()
		readableMedium = nil
	}
	if resumableRecorder := downloader.resumableRecorder; resumableRecorder != nil && resumableRecorderOpenArgs != nil {
		writeableMedium = resumableRecorder.OpenForAppending(resumableRecorderOpenArgs)
		if writeableMedium == nil {
			writeableMedium = resumableRecorder.OpenForCreatingNew(resumableRecorderOpenArgs)
		}
		if writeableMedium != nil {
			defer writeableMedium.Close()
		}
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(int(downloader.concurrency))
	var (
		downloadingProgress      = newDownloadingPartsProgress()
		downloadingProgressMutex sync.Mutex
	)
	for _, part := range parts {
		p := part
		urlsIterClone := urlsIter.Clone()
		g.Go(func() error {
			n, err := downloader.downloadToPart(ctx, urlsIterClone, etag, offset, options.Header, p, writeableMedium, &downloadingProgressMutex, func(downloaded uint64) {
				downloadingProgress.setPartDownloadingProgress(p.Offset(), downloaded)
				if onDownloadingProgress := options.OnDownloadingProgress; onDownloadingProgress != nil {
					onDownloadingProgress(&DownloadingProgress{Downloaded: downloadingProgress.totalDownloaded(), TotalSize: needToDownload})
				}
			})
			if n > 0 {
				downloadingProgress.partDownloaded(p.Offset(), n)
				if onDownloadingProgress := options.OnDownloadingProgress; onDownloadingProgress != nil {
					onDownloadingProgress(&DownloadingProgress{Downloaded: downloadingProgress.totalDownloaded(), TotalSize: needToDownload})
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
	if resumableRecorder := downloader.resumableRecorder; resumableRecorder != nil && resumableRecorderOpenArgs != nil && err == nil {
		resumableRecorder.Delete(resumableRecorderOpenArgs)
	}
	return downloadingProgress.totalDownloaded(), err
}

func (downloader concurrentDownloader) downloadToPart(
	ctx context.Context, urlsIter URLsIter, etag string, originalOffset uint64, headers http.Header,
	part destination.Part, writeableMedium resumablerecorder.WriteableResumableRecorderMedium,
	downloadingProgressMutex *sync.Mutex, onDownloadingProgress func(downloaded uint64)) (uint64, error) {
	var (
		n                           uint64
		err                         error
		size                        = part.Size()
		offset                      = part.Offset()
		haveRead                    = part.HaveDownloaded()
		downloadingProgressCallback func(uint64)
	)
	if onDownloadingProgress != nil {
		downloadingProgressCallback = func(downloaded uint64) {
			if downloadingProgressMutex != nil {
				downloadingProgressMutex.Lock()
				defer downloadingProgressMutex.Unlock()
			}
			onDownloadingProgress(downloaded)
		}
	}
	for size > haveRead {
		n, err = downloadToPartReaderWithOffsetAndSize(ctx, urlsIter, etag, originalOffset+offset+haveRead, size-haveRead,
			headers, downloader.client, part, downloadingProgressCallback)
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
	ctx context.Context, urlsIter URLsIter, etag string, offset, size uint64, headers http.Header,
	client clientv2.Client, part destination.PartWriter, onDownloadingProgress func(downloaded uint64)) (uint64, error) {
	headers = cloneHeader(headers)
	setRange(headers, offset, offset+size)
	return _downloadToPartReader(ctx, urlsIter, headers, etag, client, part, onDownloadingProgress)
}

func downloadToPartReader(
	ctx context.Context, urlsIter URLsIter, etag string, headers http.Header,
	client clientv2.Client, part destination.PartWriter, onDownloadingProgress func(downloaded uint64)) (uint64, error) {
	if headers.Get("Range") == "" {
		headers = cloneHeader(headers)
		setAcceptGzip(headers)
	}
	return _downloadToPartReader(ctx, urlsIter, headers, etag, client, part, onDownloadingProgress)
}

func _downloadToPartReader(
	ctx context.Context, urlsIter URLsIter, headers http.Header, etag string,
	client clientv2.Client, part destination.PartWriter, onDownloadingProgress func(downloaded uint64)) (uint64, error) {
	var (
		response      *http.Response
		u             url.URL
		n             uint64
		ok, haveReset bool
		err, peekErr  error
	)

	for {
		if ok, peekErr = urlsIter.Peek(&u); peekErr != nil {
			return 0, peekErr
		} else if !ok {
			if haveReset {
				break
			} else {
				urlsIter.Reset()
				haveReset = true
				continue
			}
		}
		req := http.Request{
			Method: http.MethodGet,
			URL:    &u,
			Header: headers,
			Body:   http.NoBody,
		}
		ctx = context.WithValue(ctx, urlsIterContextKey{}, urlsIter)
		if response, err = client.Do(req.WithContext(ctx)); err != nil {
			if !retrier.IsErrorRetryable(err) {
				return 0, err
			}
			urlsIter.Next()
			continue
		}
		var (
			bodyReader io.Reader = response.Body
			bodyCloser io.Closer = response.Body
		)
		if etag == parseEtag(response.Header.Get("Etag")) {
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
				bodyCloser.Close()
				err = errors.New("unrecognized content-encoding")
			}
		} else {
			bodyCloser.Close()
			err = errors.New("etag dismatch")
		}
		urlsIter.Next()
	}
	return 0, err
}

func headRequest(ctx context.Context, urlsIter URLsIter, headers http.Header, client clientv2.Client) (response *http.Response, err error) {
	var (
		u             url.URL
		ok, haveReset bool
	)
	if headers.Get("Accept-Encoding") != "" {
		headers = cloneHeader(headers)
		headers.Del("Accept-Encoding")
	}
	for {
		if ok, err = urlsIter.Peek(&u); err != nil {
			return
		} else if !ok {
			if haveReset {
				break
			} else {
				urlsIter.Reset()
				haveReset = true
				continue
			}
		}
		req := http.Request{
			Method: http.MethodHead,
			URL:    &u,
			Header: headers,
			Body:   http.NoBody,
		}
		if response, err = client.Do(req.WithContext(ctx)); err != nil {
			if !retrier.IsErrorRetryable(err) {
				return
			}
			urlsIter.Next()
			continue
		}
		break
	}
	if response != nil && response.Body != nil {
		response.Body.Close()
	}
	return
}

func setAcceptGzip(headers http.Header) {
	headers.Set("Accept-Encoding", "gzip")
}

func setRange(headers http.Header, from, end uint64) {
	headers.Set("Range", fmt.Sprintf("bytes=%d-%d", from, end-1))
	headers.Del("Accept-Encoding")
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
func parseEtag(etag string) string {
	etag = strings.TrimPrefix(etag, "\"")
	etag = strings.TrimSuffix(etag, "\"")
	etag = strings.TrimSuffix(etag, ".gz")
	return etag
}

func cloneHeader(h http.Header) http.Header {
	if h == nil {
		return make(http.Header)
	}

	// Find total number of values.
	nv := 0
	for _, vv := range h {
		nv += len(vv)
	}
	sv := make([]string, nv) // shared backing array for headers' values
	h2 := make(http.Header, len(h))
	for k, vv := range h {
		n := copy(sv, vv)
		h2[k] = sv[:n:n]
		sv = sv[n:]
	}
	return h2
}
