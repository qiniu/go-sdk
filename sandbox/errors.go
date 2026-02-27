package sandbox

import "fmt"

// APIError 表示 API 返回的非预期 HTTP 响应。
type APIError struct {
	StatusCode int
	Body       []byte
}

// Error 实现 error 接口。
func (e *APIError) Error() string {
	return fmt.Sprintf("api error: status %d, body: %s", e.StatusCode, string(e.Body))
}
