package sms

import (
	"fmt"
	"net/url"
)

type Message struct {
	MessageID string    `json:"message_id"`
	JobID     string    `json:"job_id"`
	Mobile    string    `json:"mobile"`
	Content   string    `json:"content"`
	Status    string    `json:"status"`
	Type      string    `json:"type"` //短信类型
	Error     string    `json:"error"`
	Count     int       `json:"count"`
	CreatedAt timestamp `json:"createat"`
	DelivrdAt timestamp `json:"delivrat"`
}
type timestamp string

// MessagesRequest 短信消息
type MessagesRequest struct {
	SignatureID string                 `json:"signature_id"`
	TemplateID  string                 `json:"template_id"`
	Mobiles     []string               `json:"mobiles"`
	Parameters  map[string]interface{} `json:"parameters"`
}
type MessagesSingleRequest struct {
	SignatureID string                 `json:"signature_id"`
	TemplateID  string                 `json:"template_id"`
	Mobile      string                 `json:"mobile"`
	Parameters  map[string]interface{} `json:"parameters"`
}
type MessagesOverseaRequest struct {
	SignatureID string                 `json:"signature_id"`
	TemplateID  string                 `json:"template_id"`
	Mobile      string                 `json:"mobile"`
	Parameters  map[string]interface{} `json:"parameters"`
}
type MessagesFulltextRequest struct {
	Template_Type string   `json:"template_type"`
	Content       string   `json:"content"`
	Mobiles       []string `json:"mobiles"`
}

// QueryMessageRequest 查询模板参数
type QueryMessageRequest struct {
	JobID      string   `json:"job_id"`
	Page       int      `json:"page"`      // 页码，默认为 1
	PageSize   int      `json:"page_size"` // 分页大小，默认为 20
	Start      int      `json:"start"`     //时间戳，开始时间
	End        int      `json:"end"`       //时间戳，结束时间
	Type       string   `json:"type"`      //短信类型
	TemplateID string   `json:"template_id"`
	MessageID  string   `json:"message_id"`
	Mobiles    []string `json:"mobiles"`
	Status     string   `json:"status"`
}

type MessagePagination struct {
	Page     int       `json:"page"`      // 页码，默认为 1
	PageSize int       `json:"page_size"` // 分页大小，默认为 20
	Items    []Message `json:"items"`     // item
}

// MessagesResponse 发送短信响应
type MessagesResponse struct {
	JobID string `json:"job_id"`
}
type MessageSingleResponse struct {
	MessageID string `json:"message_id"`
}
type MessageOverseaResponse struct {
	MessageID string `json:"message_id"`
}
type MessagesFulltextResponse struct {
	JobID string `json:"job_id"`
}
type QueryMessageResponse struct {
	Page     int       `json:"page"`      // 页码，默认为 1
	PageSize int       `json:"page_size"` // 分页大小，默认为 20
	Items    []Message `json:"items"`     //
}

// SendMessage 发送短信
func (m *Manager) SendMessage(args MessagesRequest) (ret MessagesResponse, err error) {
	url := fmt.Sprintf("%s%s", Host, "/v1/message")
	err = m.client.CallWithJSON(&ret, url, args)
	return
}

//SendSingleMessage发送单条短信
func (m *Manager) SendSingleMessage(args MessagesSingleRequest) (ret MessageSingleResponse, err error) {
	url := fmt.Sprintf("%s%s", Host, "/v1/message/single")
	err = m.client.CallWithJSON(&ret, url, args)
	return
}

//SendOverseaMessage发送国际/港澳台短信
func (m *Manager) SendOverseaMessage(args MessagesOverseaRequest) (ret MessageOverseaResponse, err error) {
	url := fmt.Sprintf("%s%s", Host, "/v1/message/oversea")
	err = m.client.CallWithJSON(&ret, url, args)
	return
}

//SendFulltextMessage发送国际/港澳台短信
func (m *Manager) SendFulltextMessage(args MessagesFulltextRequest) (ret MessagesFulltextResponse, err error) {
	url := fmt.Sprintf("%s%s", Host, "/v1/message/fulltext")
	err = m.client.CallWithJSON(&ret, url, args)
	return
}

//QueryMessage 查询短信
func (m *Manager) QueryMessage(args QueryMessageRequest) (pagination MessagePagination, err error) {
	values := url.Values{}

	if args.Page > 0 {
		values.Set("page", fmt.Sprintf("%d", args.Page))
	}

	if args.PageSize > 0 {
		values.Set("page_size", fmt.Sprintf("%d", args.PageSize))
	}

	url := fmt.Sprintf("%s%s?%s", Host, "/v1/messages", values.Encode())
	err = m.client.GetCall(&pagination, url)
	return
}
