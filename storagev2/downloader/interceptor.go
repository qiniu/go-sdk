package downloader

import (
	"net/http"

	"github.com/qiniu/go-sdk/v7/internal/clientv2"
)

type (
	retryWhenTokenOutOfDateInterceptor struct{}
	urlsIterContextKey                 struct{}
)

func (interceptor retryWhenTokenOutOfDateInterceptor) Priority() clientv2.InterceptorPriority {
	return clientv2.InterceptorPriorityAuth
}

func (interceptor retryWhenTokenOutOfDateInterceptor) Intercept(req *http.Request, handler clientv2.Handler) (resp *http.Response, err error) {
	if urlsIter, ok := req.Context().Value(urlsIterContextKey{}).(URLsIter); ok {
		if _, err = urlsIter.Peek(req.URL); err != nil {
			return
		}
	}
	resp, err = handler(req)
	return
}
