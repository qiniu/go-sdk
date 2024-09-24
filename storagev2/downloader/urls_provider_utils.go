package downloader

import (
	"context"
	"net/url"
)

// GetURLs 从 DownloadURLsProvider 读取出所有 URL 返回
func GetURLs(ctx context.Context, provider DownloadURLsProvider, objectName string, options *GenerateOptions) ([]*url.URL, error) {
	var ok bool

	iter, err := provider.GetURLsIter(ctx, objectName, options)
	if err != nil {
		return nil, err
	}
	urls := make([]*url.URL, 0, 16)
	for {
		u := new(url.URL)
		ok, err = iter.Peek(u)
		if err != nil || !ok {
			break
		}
		urls = append(urls, u)
		iter.Next()
	}
	return urls, nil
}

// GetURLStrings 从 DownloadURLsProvider 读取出所有 URL 并转换成字符串返回
func GetURLStrings(ctx context.Context, provider DownloadURLsProvider, objectName string, options *GenerateOptions) ([]string, error) {
	urls, err := GetURLs(ctx, provider, objectName, options)
	if err != nil {
		return nil, err
	}
	strs := make([]string, len(urls))
	for i, u := range urls {
		strs[i] = u.String()
	}
	return strs, nil
}
