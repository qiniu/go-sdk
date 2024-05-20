package clientv2

import (
	"net/http"

	"github.com/qiniu/go-sdk/v7/auth"
)

type AuthConfig struct {
	// 鉴权参数
	Credentials *auth.Credentials
	// 鉴权类型，不包含上传
	TokenType auth.TokenType
	// 签名前回调函数
	BeforeSign func(*http.Request)
	// 签名后回调函数
	AfterSign func(*http.Request)
	// 签名失败回调函数
	SignError func(*http.Request, error)
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
		if interceptor.config.BeforeSign != nil {
			interceptor.config.BeforeSign(req)
		}
		if err := credentials.AddToken(interceptor.config.TokenType, req); err != nil {
			if interceptor.config.SignError != nil {
				interceptor.config.SignError(req, err)
			}
			return nil, err
		} else if interceptor.config.AfterSign != nil {
			interceptor.config.AfterSign(req)
		}
	}

	return handler(req)
}
