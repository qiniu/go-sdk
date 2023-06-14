package clientv2

import (
	"net/http"
)

const (
	InterceptorPriorityDefault     InterceptorPriority = 100
	InterceptorPriorityRetryHosts  InterceptorPriority = 200
	InterceptorPriorityRetrySimple InterceptorPriority = 300
	InterceptorPrioritySetHeader   InterceptorPriority = 400
	InterceptorPriorityNormal      InterceptorPriority = 500
	InterceptorPriorityAuth        InterceptorPriority = 600
	InterceptorPriorityDebug       InterceptorPriority = 700
)

type InterceptorPriority int

type Interceptor interface {
	// Priority 数字越小优先级越高
	Priority() InterceptorPriority

	// Intercept 拦截处理函数
	Intercept(req *http.Request, handler Handler) (*http.Response, error)
}

type Interceptors []Interceptor

func (is Interceptors) Less(i, j int) bool {
	return is[i].Priority() < is[j].Priority()
}

func (is Interceptors) Swap(i, j int) {
	is[i], is[j] = is[j], is[i]
}

func (is Interceptors) Len() int {
	return len(is)
}

type simpleInterceptor struct {
	priority InterceptorPriority
	handler  func(req *http.Request, handler Handler) (*http.Response, error)
}

func (s *simpleInterceptor) Priority() InterceptorPriority {
	return s.priority
}

func (s *simpleInterceptor) Intercept(req *http.Request, handler Handler) (*http.Response, error) {
	if s == nil || s.handler == nil {
		return handler(req)
	}
	return s.handler(req, handler)
}

func NewSimpleInterceptor(interceptorHandler func(req *http.Request, handler Handler) (*http.Response, error)) Interceptor {
	return NewSimpleInterceptorWithPriority(InterceptorPriorityNormal, interceptorHandler)
}

func NewSimpleInterceptorWithPriority(priority InterceptorPriority, interceptorHandler func(req *http.Request, handler Handler) (*http.Response, error)) Interceptor {
	if priority <= 0 {
		priority = 100
	}

	return &simpleInterceptor{
		priority: priority,
		handler:  interceptorHandler,
	}
}
