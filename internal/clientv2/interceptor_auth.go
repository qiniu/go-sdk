package clientv2

import (
	"github.com/qiniu/go-sdk/v7/auth"
	"net/http"
)

type AuthOptions struct {
	Credentials auth.Credentials //
	TokenType   auth.TokenType   // 不包含上传
}

type authInterceptor struct {
	options AuthOptions
}

func NewAuthInterceptor(options AuthOptions) Interceptor {
	return &authInterceptor{
		options: options,
	}
}

func (r *authInterceptor) Priority() InterceptorPriority {
	return InterceptorPriorityAuth
}

func (r *authInterceptor) Intercept(req *http.Request, handler Handler) (*http.Response, error) {
	if r == nil {
		return handler(req)
	}

	err := r.options.Credentials.AddToken(r.options.TokenType, req)
	if err != nil {
		return nil, err
	}

	return handler(req)
}
