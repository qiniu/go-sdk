// Package sms 提供七牛云短信服务的 Go 客户端。
//
// 官方文档: https://developer.qiniu.com/sms
//
// # 创建客户端
//
//	manager := sms.NewManager(mac)
//
// # 发送短信
//
//	resp, err := manager.SendMessage(sms.MessagesRequest{
//	    SignatureID: "签名ID",
//	    TemplateID:  "模板ID",
//	    Mobiles:     []string{"13800138000"},
//	    Parameters:  map[string]interface{}{"code": "1234"},
//	})
//
// # 签名管理
//
//   - [Manager.CreateSignature] / [Manager.QuerySignature] / [Manager.DeleteSignature]
//
// # 模板管理
//
//   - [Manager.CreateTemplate] / [Manager.QueryTemplate] / [Manager.DeleteTemplate]
package sms
