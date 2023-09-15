package clientv2

import (
	"context"
	"net/http"
	"sort"
)

type intercetorsContextKey struct{}

func WithInterceptors(req *http.Request, interceptors ...Interceptor) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), intercetorsContextKey{}, interceptorList(interceptors)))
}

func getIntercetorsFromRequest(req *http.Request) interceptorList {
	if req == nil {
		return interceptorList{}
	}
	interceptors, ok := req.Context().Value(intercetorsContextKey{}).(interceptorList)
	if !ok {
		return interceptorList{}
	}
	sort.Sort(interceptors)
	return interceptors
}
