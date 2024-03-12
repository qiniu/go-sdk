package clientv2

import (
	"net/http"

	"github.com/qiniu/go-sdk/v7/auth"
)

type AuthConfig struct {
	Credentials *auth.Credentials //
	TokenType   auth.TokenType    // 不包含上传
}

type authInterceptor struct {
	config AuthConfig
}

func NewAuthInterceptor(config AuthConfig) Interceptor {
	return &authInterceptor{
		config: config,
	}
}

func (interceptor *authInterceptor) Priority() InterceptorPriority {
	return InterceptorPriorityAuth
}

func (interceptor *authInterceptor) Intercept(req *http.Request, handler Handler) (*http.Response, error) {
	if interceptor == nil || req == nil {
		return handler(req)
	}

	if credentials := interceptor.config.Credentials; credentials != nil {
		err := credentials.AddToken(interceptor.config.TokenType, req)
		if err != nil {
			return nil, err
		}
	}

	return handler(req)
}
