package downloader

import (
	"net/http"

	"github.com/qiniu/go-sdk/v7/internal/clientv2"
)

type (
	retryWhenTokenOutOfDateInterceptor struct{}
	urlProviderContextKey              struct{}
)

func (interceptor retryWhenTokenOutOfDateInterceptor) Priority() clientv2.InterceptorPriority {
	return clientv2.InterceptorPriorityAuth
}

func (interceptor retryWhenTokenOutOfDateInterceptor) Intercept(req *http.Request, handler clientv2.Handler) (resp *http.Response, err error) {
	if urlProvider, ok := req.Context().Value(urlProviderContextKey{}).(URLProvider); ok {
		if err = urlProvider.GetURL(req.URL); err != nil {
			return nil, err
		}
	}
	resp, err = handler(req)
	return
}
