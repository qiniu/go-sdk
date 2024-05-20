package clientv2

import (
	"net/http"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/backoff"
	"github.com/qiniu/go-sdk/v7/storagev2/retrier"
)

type RetryConfig struct {
	RetryMax      int                  // 最大重试次数
	RetryInterval func() time.Duration // 重试时间间隔 v1
	Backoff       backoff.Backoff      // 重试时间间隔 v2，优先级高于 RetryInterval
	ShouldRetry   func(req *http.Request, resp *http.Response, err error) bool
	Retrier       retrier.Retrier // 重试器
}

func (c *RetryConfig) init() {
	if c == nil {
		return
	}

	if c.RetryMax < 0 {
		c.RetryMax = 0
	}
}
