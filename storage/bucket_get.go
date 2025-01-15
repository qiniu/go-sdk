package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/downloader"
	"github.com/qiniu/go-sdk/v7/storagev2/region"
)

type GetObjectInput struct {
	Context         context.Context // 下载所用的 Context
	DownloadDomains []string        // 下载域名列表，如果不填则使用默认源站域名，下载域名可以接受直接填写<HOST>，或是 <protocol>://<HOST> 的格式，如果设置了 protocol 则忽略 UseHttps 的设置；当前仅使用第一个域名
	PresignUrl      bool            // 下载域名是否需要签名，如果使用源站域名则总是签名
	Range           string          // 获取范围，格式同 HTTP 协议的 Range Header
	TrafficLimit    uint64          // 下载单链限速，单位：bit/s；范围：819200 - 838860800（即800Kb/s - 800Mb/s），如果超出该范围将返回 400 错误
}

type GetObjectOutput struct {
	ContentType   string            // 获取 MIME TYPE
	ContentLength int64             // 获取返回的数据量，如果是 -1 表示未知
	ETag          string            // 获取对象的 Etag
	Metadata      map[string]string // 获取自定义元数据
	LastModified  time.Time         // 获取对象最后一次修改时间
	Body          io.ReadCloser     // 获取对象数据
}

var _ io.ReadCloser = (*GetObjectOutput)(nil)

func (g *GetObjectOutput) Read(p []byte) (n int, err error) {
	if g.Body == nil {
		return 0, errors.New("read: body is empty")
	}
	return g.Body.Read(p)
}

func (g *GetObjectOutput) Close() error {
	if g.Body == nil {
		return errors.New("close: body is empty")
	}
	return g.Body.Close()
}

type (
	trafficLimitDownloadURLsProvider struct {
		base         downloader.DownloadURLsProvider
		trafficLimit uint64
	}
	trafficLimitURLsIter struct {
		iter         downloader.URLsIter
		trafficLimit uint64
	}
)

func (p trafficLimitURLsIter) Peek(u *url.URL) (bool, error) {
	if ok, err := p.iter.Peek(u); err != nil {
		return ok, err
	} else if !ok {
		return false, nil
	} else {
		if u.RawQuery != "" {
			u.RawQuery += "&"
		}
		u.RawQuery += fmt.Sprintf("X-Qiniu-Traffic-Limit=%d", p.trafficLimit)
		return true, nil
	}
}

func (p trafficLimitURLsIter) Next() {
	p.iter.Next()
}

func (p trafficLimitURLsIter) Reset() {
	p.iter.Reset()
}

func (p trafficLimitURLsIter) Clone() downloader.URLsIter {
	return trafficLimitURLsIter{p.iter.Clone(), p.trafficLimit}
}

func (p trafficLimitDownloadURLsProvider) GetURLsIter(ctx context.Context, objectName string, options *downloader.GenerateOptions) (downloader.URLsIter, error) {
	if urlsIter, err := p.base.GetURLsIter(ctx, objectName, options); err != nil {
		return nil, err
	} else {
		return trafficLimitURLsIter{urlsIter, p.trafficLimit}, nil
	}
}

// Get
//
//	@Description: 下载文件
//	@receiver m BucketManager
//	@param bucket 文件所在 bucket
//	@param key 文件的 key
//	@param options 下载可选配置
//	@return *GetObjectOutput 响应，注：GetObjectOutput 和 error 可能同时存在，有 GetObjectOutput 时请尝试 close
//	@return error 请求错误信息
func (m *BucketManager) Get(bucket, key string, options *GetObjectInput) (*GetObjectOutput, error) {
	if options == nil {
		options = &GetObjectInput{}
	}
	ctx := options.Context
	if ctx == nil {
		ctx = context.Background()
	}
	bucketHosts, err := getUcEndpointProvider(m.Cfg.UseHTTPS, nil).GetEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	var accessKey string
	if m.Mac != nil {
		accessKey = m.Mac.AccessKey
	}
	urlsProvider := downloader.NewDefaultSrcURLsProvider(accessKey, &downloader.DefaultSrcURLsProviderOptions{
		BucketRegionsQueryOptions: region.BucketRegionsQueryOptions{},
		BucketHosts:               bucketHosts,
	})
	urlsProvider = m.applyPresignOnUrlsProvider(m.applyTrafficLimitOnUrlsProvider(urlsProvider, options.TrafficLimit))
	if len(options.DownloadDomains) > 0 {
		staticDomainBasedURLsProvider := downloader.NewStaticDomainBasedURLsProvider(options.DownloadDomains)
		staticDomainBasedURLsProvider = m.applyTrafficLimitOnUrlsProvider(staticDomainBasedURLsProvider, options.TrafficLimit)
		if options.PresignUrl {
			staticDomainBasedURLsProvider = m.applyPresignOnUrlsProvider(staticDomainBasedURLsProvider)
		}
		urlsProvider = downloader.CombineDownloadURLsProviders(staticDomainBasedURLsProvider, urlsProvider)
	}
	reqHeaders := make(http.Header)
	if options.Range != "" {
		reqHeaders.Set("Range", options.Range)
	}
	var (
		getObjectOutput GetObjectOutput
		headerChan      = make(chan struct{})
		errChan         = make(chan error)
		areChansClosed  int32
	)
	defer func() {
		atomic.StoreInt32(&areChansClosed, 1)
		close(headerChan)
		close(errChan)
	}()

	objectOptions := downloader.ObjectOptions{
		DestinationDownloadOptions: downloader.DestinationDownloadOptions{
			Header: reqHeaders,
			OnResponseHeader: func(h http.Header) {
				defer func() {
					if atomic.LoadInt32(&areChansClosed) == 0 {
						headerChan <- struct{}{}
					}
				}()
				getObjectOutput.ContentType = h.Get("Content-Type")
				getObjectOutput.ETag = parseEtag(h.Get("ETag"))

				lm := h.Get("Last-Modified")
				if len(lm) > 0 {
					if t, e := time.Parse(time.RFC1123, lm); e == nil {
						getObjectOutput.LastModified = t
					}
				}

				metaData := make(map[string]string)
				for k, v := range h {
					if len(v) > 0 && strings.HasPrefix(strings.ToLower(k), "x-qn-meta-") {
						metaData[k] = v[0]
					}
				}
				getObjectOutput.Metadata = metaData
			}},
		GenerateOptions: downloader.GenerateOptions{
			BucketName:          bucket,
			UseInsecureProtocol: !m.Cfg.UseHTTPS,
		},
		DownloadURLsProvider: urlsProvider,
	}

	pipeR, pipeW := io.Pipe()
	getObjectOutput.Body = pipeR
	go func() {
		n, err := m.downloadManager.DownloadToWriter(ctx, key, pipeW, &objectOptions)
		getObjectOutput.ContentLength = int64(n)
		if atomic.LoadInt32(&areChansClosed) == 0 {
			errChan <- err
		}
		pipeW.CloseWithError(err)
	}()

	select {
	case <-headerChan:
		return &getObjectOutput, nil
	case err := <-errChan:
		return &getObjectOutput, err
	}
}

func (m *BucketManager) applyTrafficLimitOnUrlsProvider(urlsProvider downloader.DownloadURLsProvider, trafficLimit uint64) downloader.DownloadURLsProvider {
	if trafficLimit > 0 {
		urlsProvider = trafficLimitDownloadURLsProvider{urlsProvider, trafficLimit}
	}
	return urlsProvider
}

func (m *BucketManager) applyPresignOnUrlsProvider(urlsProvider downloader.DownloadURLsProvider) downloader.DownloadURLsProvider {
	signOptions := downloader.SignOptions{TTL: 3 * time.Minute}
	return downloader.SignURLsProvider(urlsProvider, downloader.NewCredentialsSigner(m.Mac), &signOptions)
}
