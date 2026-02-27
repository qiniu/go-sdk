package sandbox

import (
	"fmt"
	"net/http"

	"connectrpc.com/connect"
)

// APIError 表示 API 返回的非预期 HTTP 响应。
type APIError struct {
	StatusCode int
	Body       []byte
}

// Error 实现 error 接口。
func (e *APIError) Error() string {
	return fmt.Sprintf("api error: status %d, body: %s", e.StatusCode, string(e.Body))
}

// isNotFoundError 判断错误是否为"未找到"类型。
func isNotFoundError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusNotFound
	}
	if connectErr := new(connect.Error); connect.CodeOf(err) == connect.CodeNotFound {
		_ = connectErr
		return true
	}
	return false
}
