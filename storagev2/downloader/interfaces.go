package downloader

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/downloader/destination"
)

type (
	// 生成对象下载 URL 接口
	DownloadURLsGenerator interface {
		GenerateURLs(context.Context, string, *GenerateOptions) ([]*url.URL, error)
	}

	// 对下载 URL 签名
	Signer interface {
		Sign(context.Context, *url.URL, *SignOptions) error
	}

	// 目标下载选项
	DestinationDownloadOptions struct {
		// 对象请求 Header
		Header http.Header

		// 对象下载进度
		OnDownloadingProgress func(downloaded, totalSize uint64)
	}

	// 目标下载器
	DestinationDownloader interface {
		Download(context.Context, []*url.URL, destination.Destination, *DestinationDownloadOptions) (uint64, error)
	}

	// 对象下载 URL 生成选项
	GenerateOptions struct {
		// 对象名称，可选
		BucketName string

		// 文件处理命令，可选
		Command string

		// 是否使用 HTTP 协议，默认为不使用
		UseInsecureProtocol bool
	}

	// 对象签名选项
	SignOptions struct {
		// 签名有效期，如果不填写，默认为 3 分钟
		Ttl time.Duration
	}
)
