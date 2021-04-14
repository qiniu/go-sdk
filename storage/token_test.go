package storage

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestResignToken(t *testing.T) {
	p := PutPolicy{}
	token1 := p.UploadToken(mac)
	token2 := p.UploadToken(mac)
	if token1 != token2 {
		t.Fail()
	}
}

func TestForceSaveKeyFalse(t *testing.T) {
	p := PutPolicy{}
	pj, _ := json.Marshal(p)
	s := string(pj)
	t.Log(s)
	if strings.Contains(s, "forceSaveKey") {
		t.Fail()
	}
}

func TestForceSaveKeyTrue(t *testing.T) {
	p := PutPolicy{}
	p.ForceSaveKey = true
	pj, _ := json.Marshal(p)
	s := string(pj)
	t.Log(s)
	if !strings.Contains(s, "forceSaveKey") {
		t.Fail()
	}
}
