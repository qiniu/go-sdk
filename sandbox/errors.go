package sandbox

import (
	"encoding/json"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
)

// APIError 表示 API 返回的非预期 HTTP 响应。
type APIError struct {
	StatusCode int
	Body       []byte

	// Code 是从响应 body 中解析出的错误码（如果有）。
	Code string
	// Message 是从响应 body 中解析出的错误消息（如果有）。
	Message string
}

// Error 实现 error 接口。
func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("api error: status %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("api error: status %d, body: %s", e.StatusCode, string(e.Body))
}

// newAPIError 创建 APIError 并尝试从 JSON body 中解析结构化字段。
func newAPIError(statusCode int, body []byte) *APIError {
	e := &APIError{StatusCode: statusCode, Body: body}
	e.Code, e.Message = parseAPIErrorBody(body)
	return e
}

// parseAPIErrorBody 尝试从 JSON body 中解析 code 和 message 字段。
func parseAPIErrorBody(body []byte) (code, message string) {
	if len(body) == 0 {
		return "", ""
	}
	var parsed struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &parsed) == nil {
		return parsed.Code, parsed.Message
	}
	return "", ""
}

// isNotFoundError 判断错误是否为"未找到"类型。
func isNotFoundError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusNotFound
	}
	if connect.CodeOf(err) == connect.CodeNotFound {
		return true
	}
	return false
}
