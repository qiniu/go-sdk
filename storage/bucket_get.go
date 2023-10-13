package storage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	clientV1 "github.com/qiniu/go-sdk/v7/client"
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
		options = &GetObjectInput{
			DownloadDomains: nil,
			PresignUrl:      false,
			Range:           "",
		}
	}

	domain := ""
	if len(options.DownloadDomains) > 0 {
		// 使用用户配置域名
		domain = options.DownloadDomains[0]
	} else {
		// 查源站域名
		if rg, e := getRegionByV4(m.Mac.AccessKey, bucket, UCApiOptions{
			UseHttps:           m.Cfg.UseHTTPS,
			RetryMax:           m.options.RetryMax,
			HostFreezeDuration: m.options.HostFreezeDuration,
		}); e != nil {
			return nil, e
		} else if len(rg.regions) == 0 {
			return nil, errors.New("can't get region with bucket")
		} else {
			domain = rg.regions[0].IoSrcHost
		}
		options.PresignUrl = true
	}

	if len(domain) == 0 {
		return nil, errors.New("download domain is empty")
	}

	query := url.Values{}
	if options.TrafficLimit > 0 {
		query.Add("X-Qiniu-Traffic-Limit", strconv.FormatUint(options.TrafficLimit, 10))
	}
	downloadUrl := endpoint(m.Cfg.UseHTTPS, domain)
	if options.PresignUrl {
		deadline := time.Now().Unix() + 3*60
		downloadUrl = MakePrivateURLv2WithQuery(m.Mac, downloadUrl, key, query, deadline)
	} else {
		downloadUrl = MakePublicURLv2WithQuery(key, downloadUrl, query)
	}

	resp, err := m.getWithDownloadUrl(options.Context, downloadUrl, options.Range, options.TrafficLimit)
	if err == nil && resp == nil {
		return nil, errors.New("response is empty")
	}

	// err 和 resp 必须有一个有值
	var output *GetObjectOutput = nil
	if resp != nil {
		output = &GetObjectOutput{
			ContentType:   "",
			ContentLength: resp.ContentLength,
			ETag:          "",
			Metadata:      nil,
			LastModified:  time.Time{},
			Body:          resp.Body,
		}
	}

	if err != nil {
		return output, err
	}

	// err 为空， resp 必有值
	if resp.StatusCode/100 != 2 {
		return output, ResponseError(resp)
	}

	if resp.Header != nil {
		output.ContentType = resp.Header.Get("Content-Type")
		output.ETag = parseEtag(resp.Header.Get("ETag"))

		lm := resp.Header.Get("Last-Modified")
		if len(lm) > 0 {
			if t, e := time.Parse(time.RFC1123, lm); e == nil {
				output.LastModified = t
			}
		}

		metaData := make(map[string]string)
		for k, v := range resp.Header {
			if len(v) > 0 && strings.HasPrefix(strings.ToLower(k), "x-qn-meta-") {
				metaData[k] = v[0]
			}
		}
		output.Metadata = metaData
	}

	return output, nil
}

func (m *BucketManager) getWithDownloadUrl(ctx context.Context, downloadUrl string, contentRange string, trafficLimit uint64) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	req, err := http.NewRequest(http.MethodGet, downloadUrl, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	clientV1.AddHttpHeaderRange(req.Header, contentRange)

	return m.Client.Do(ctx, req)
}
