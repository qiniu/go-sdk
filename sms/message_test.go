package sms_test

import (
	"os"
	"testing"

	"github.com/qiniu/go-sdk/v7/sms"
)

func TestMessage(t *testing.T) {
	args := sms.MessagesRequest{
		SignatureID: os.Getenv("signatureid"),
		TemplateID:  os.Getenv("templateid"),
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
		SignatureID: os.Getenv("signatureid"),
		TemplateID:  os.Getenv("templateid"),
		Mobile:      "15196436347",
	}
	ret, err := manager.SendSingleMessage(args)

	if err != nil {
		t.Fatalf("TestSingleMessage error: %v\n", err)
	}
	if len(ret.MessageID) == 0 {
		t.Fatal("TestSingleMessage() error: The MessageID cannot be empty")
	}

}

func TestOverseaMessage(t *testing.T) {
	args := sms.MessagesOverseaRequest{
		SignatureID: os.Getenv("signatureid"),
		TemplateID:  os.Getenv("templateidOversea"),
		Mobile:      "+17245678901",
	}
	ret, err := manager.SendOverseaMessage(args)

	if err != nil {
		t.Fatalf("SendOverseaMessage error: %v\n", err)
	}
	if len(ret.MessageID) == 0 {
		t.Fatal("SendOverseaMessage() error: The MessageID cannot be empty")
	}

}

func TestFulltextMessage(t *testing.T) {
	args := sms.MessagesFulltextRequest{
		Template_Type: "notification",
		Content:       os.Getenv("content"),
		Mobiles:       []string{"15196436347"},
	}
	ret, err := manager.SendFulltextMessage(args)

	if err != nil {
		t.Fatalf("TestFulltextMessage error: %v\n", err)
	}
	if len(ret.JobID) == 0 {
		t.Fatal("TestFulltextMessage() error: The JobID cannot be empty")
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
