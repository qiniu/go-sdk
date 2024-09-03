package clientv2

import "net/http"

type bufferResponseInterceptor struct {
}

func NewBufferResponseInterceptor() Interceptor {
	return bufferResponseInterceptor{}
}

func (interceptor bufferResponseInterceptor) Priority() InterceptorPriority {
	return InterceptorPriorityBufferResponse
}

func (interceptor bufferResponseInterceptor) Intercept(req *http.Request, handler Handler) (resp *http.Response, err error) {
	toBufferResponse := req.Context().Value(bufferResponseContextKey{}) != nil
	resp, err = handler(req)
	if err == nil && toBufferResponse {
		err = bufferResponse(resp)
	}
	return
}
