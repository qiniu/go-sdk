package sms_test

import (
	"testing"

	"github.com/qiniu/go-sdk/v7/sms"
)

func TestMessage(t *testing.T) {
	args := sms.MessagesRequest{
		SignatureID: "1603574419001716736",
		TemplateID:  "1643211372198117376",
		Mobiles:     []string{"15196436347", "15680767295"},
	}
	ret, err := manager.SendMessage(args)

	if err != nil {
		t.Fatalf("SendMessage error: %v\n", err)
	}
	if len(ret.JobID) == 0 {
		t.Fatal("SendMessage() error: The JobID cannot be empty")
	}

}
func TestSingleMessage(t *testing.T) {
	args := sms.MessagesSingleRequest{
		SignatureID: "1603574419001716736",
		TemplateID:  "1630050990797369344",
		Mobile:      "15196436347",
	}
	ret, err := manager.SendSingleMessage(args)

	if err != nil {
		t.Fatalf("TestSingleMessage error: %v\n", err)
	}
	if len(ret.MessageId) == 0 {
		t.Fatal("TestSingleMessage() error: The MessageId cannot be empty")
	}

}

func TestOverseaMessage(t *testing.T) {
	args := sms.MessagesOverseaRequest{
		SignatureID: "1603574419001716736",
		TemplateID:  "1643890365607985152",
		Mobile:      "17245678901",
	}
	ret, err := manager.SendOverseaMessage(args)

	if err != nil {
		t.Fatalf("SendOverseaMessage error: %v\n", err)
	}
	if len(ret.MessageId) == 0 {
		t.Fatal("SendOverseaMessage() error: The MessageId cannot be empty")
	}

}

func TestFulltextMessage(t *testing.T) {
	args := sms.MessagesFulltextRequest{
		Template_Type: "notification",
		Content:       "【test】您的订单编号${code}",
		Mobiles:       []string{"15196436347"},
	}
	ret, err := manager.SendFulltextMessage(args)

	if err != nil {
		t.Fatalf("TestFulltextMessage error: %v\n", err)
	}
	if len(ret.JobID) == 0 {
		t.Fatal("TestFulltextMessage() error: The JobId cannot be empty")
	}

}
func TestQueryMessage(t *testing.T) {
	query := sms.QueryMessageRequest{}

	ret, err := manager.QueryMessage(query)

	if err != nil {
		t.Fatalf("QueryMessage() error: %v\n", err)
	}

	if len(ret.Items) == 0 {
		t.Fatal("QueryMessage() error: message cannot be empty")
	}

	if ret.Page == 0 {
		t.Fatal("QueryMMessage() error: page cannot be 0")
	}

}
