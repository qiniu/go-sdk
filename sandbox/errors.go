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

	// Reqid 是从响应头 X-Reqid 中提取的请求 ID，用于链路追踪和日志排查。
	Reqid string
	// Code 是从响应 body 中解析出的错误码（如果有）。
	Code string
	// Message 是从响应 body 中解析出的错误消息（如果有）。
	Message string
}

// Error 实现 error 接口。
func (e *APIError) Error() string {
	prefix := fmt.Sprintf("api error: status %d", e.StatusCode)
	if e.Reqid != "" {
		prefix += ", reqid: " + e.Reqid
	}
	if e.Message != "" {
		return prefix + ": " + e.Message
	}
	if len(e.Body) > 0 {
		return prefix + ", body: " + string(e.Body)
	}
	return prefix
}

// newAPIError 从 HTTP 响应创建 APIError，提取 X-Reqid 头并解析 JSON body。
func newAPIError(resp *http.Response, body []byte) *APIError {
	e := &APIError{
		StatusCode: resp.StatusCode,
		Body:       body,
		Reqid:      resp.Header.Get("X-Reqid"),
	}
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
