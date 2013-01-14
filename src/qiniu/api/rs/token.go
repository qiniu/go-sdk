package rs

import (
	"qiniu/digest_auth"
	. "qiniu/api/conf"
)

// ----------------------------------------------------------

type GetPolicy struct {
	Scope		string `json:"S"`
	Deadline	uint32 `json:"E"`
}

func (r *GetPolicy) Token() string {
	return digest_auth.SignJson(ACCESS_KEY, []byte(SECRET_KEY), r)
}

// ----------------------------------------------------------

type PutPolicy struct {
	Scope            string `json:"scope,omitempty"`
	CallbackUrl      string `json:"callbackUrl,omitempty"`
	CallbackBodyType string `json:"callbackBodyType,omitempty"`
	Customer         string `json:"customer,omitempty"`
	Deadline         uint32 `json:"deadline"` 			// 截止时间（以秒为单位）
	Escape           uint32 `json:"escape,omitempty"`	// 是否允许存在转义符号		
	AsyncOps         string `json:"asyncOps,omitempty"`
}

func (r *PutPolicy) Token() string {
	return digest_auth.SignJson(ACCESS_KEY, []byte(SECRET_KEY), r)
}

// ----------------------------------------------------------

