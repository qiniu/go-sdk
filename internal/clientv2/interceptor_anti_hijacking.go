package clientv2

import (
	"net/http"

	"github.com/qiniu/go-sdk/v7/storagev2/retrier"
)

type antiHijackingInterceptor struct {
}

func NewAntiHijackingInterceptor() Interceptor {
	return &antiHijackingInterceptor{}
}

func (interceptor *antiHijackingInterceptor) Priority() InterceptorPriority {
	return InterceptorPriorityAntiHijacking
}

func (interceptor *antiHijackingInterceptor) Intercept(req *http.Request, handler Handler) (response *http.Response, err error) {
	response, err = handler(req)
	if err != nil {
		return
	}
	reqId := response.Header.Get("x-reqid")
	log := response.Header.Get("x-log")
	if reqId == "" && log == "" {
		return nil, retrier.ErrMaliciousResponse
	}
	return
}
