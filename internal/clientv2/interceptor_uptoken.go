package clientv2

import (
	"net/http"

	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"
)

type UpTokenConfig struct {
	// 上传凭证
	UpToken uptoken.UpTokenProvider
}

type uptokenInterceptor struct {
	config UpTokenConfig
}

func NewUpTokenInterceptor(config UpTokenConfig) Interceptor {
	return &uptokenInterceptor{
		config: config,
	}
}

func (interceptor *uptokenInterceptor) Priority() InterceptorPriority {
	return InterceptorPriorityAuth
}

func (interceptor *uptokenInterceptor) Intercept(req *http.Request, handler Handler) (*http.Response, error) {
	if interceptor == nil || req == nil {
		return handler(req)
	}

	if upToken := interceptor.config.UpToken; upToken != nil {
		if upToken, err := upToken.GetUpToken(req.Context()); err != nil {
			return nil, err
		} else {
			req.Header.Set("Authorization", "UpToken "+upToken)
		}
	}

	return handler(req)
}
