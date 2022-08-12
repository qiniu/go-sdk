package storage

import (
	"context"
	"errors"
	api "github.com/qiniu/go-sdk/v7"
	"github.com/qiniu/go-sdk/v7/internal/hostprovider"
	"time"
)

// ParsePutTime 提供了将PutTime转换为 time.Time 的功能
func ParsePutTime(putTime int64) (t time.Time) {
	t = time.Unix(0, putTime*100)
	return
}

// IsContextExpired 检查分片上传的ctx是否过期，提前一天让它过期
// 因为我们认为如果断点继续上传的话，最长需要1天时间
func IsContextExpired(blkPut BlkputRet) bool {
	if blkPut.Ctx == "" {
		return false
	}
	target := time.Unix(blkPut.ExpiredAt, 0).AddDate(0, 0, -1)
	now := time.Now()
	return now.After(target)
}

func shouldUploadRetry(err error) bool {
	if err == nil {
		return false
	}

	errInfo, ok := err.(*ErrorInfo)
	if !ok {
		return true
	}

	return errInfo.Code > 499 && errInfo.Code < 600 && errInfo.Code != 573 && errInfo.Code != 579
}

func doUploadAction(hostProvider hostprovider.HostProvider, retryMax int, action func(host string) error) error {
	for i := 1; ; i++ {
		host, err := hostProvider.Provider()
		if err != nil {
			return err
		}

		err = action(host)
		if err == nil {
			return nil
		}

		if errors.Is(err, context.Canceled) {
			return err
		}

		if i >= retryMax {
			return api.NewError(ErrMaxUpRetry, err.Error())
		}

		if !shouldUploadRetry(err) {
			return err
		}

		// 重试，冻结当前 host
		_ = hostProvider.Freeze(host, err, 10*60)
	}
}
