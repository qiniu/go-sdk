//go:build integration
// +build integration

package pili

import (
	"context"
	"math/rand"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManager_GetDomainsList(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	tests := map[string]struct {
		inputHub    string
		expectedErr string
	}{
		"get domains list": {inputHub: TestHub, expectedErr: ""},
		"no hub name":      {inputHub: "", expectedErr: "Key: 'GetDomainsListRequest.Hub' Error:Field validation for 'Hub' failed on the 'required' tag"},
		"unknown hub name": {inputHub: "unknown_hub", expectedErr: "hub not found"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			manager := NewManager(ManagerConfig{APIHost: ApiHost, AccessKey: AccessKey, SecretKey: SecretKey})
			response, err := manager.GetDomainsList(context.Background(), GetDomainsListRequest{Hub: tt.inputHub})
			CheckErr(ast, err, tt.expectedErr)
			if err == nil {
				t.Logf("response :[%+v]", *response)
			}
		})
	}
}

func TestManager_GetDomainInfo(t *testing.T) {
	if SkipTest() || TestDomain == "" {
		t.SkipNow()
	}
	ast := assert.New(t)
	tests := map[string]struct {
		inputHub    string
		inputDomain string
		expectedErr string
	}{
		"get domain info":           {inputHub: TestHub, inputDomain: TestDomain, expectedErr: ""},
		"mismatched domain and hub": {inputHub: TestHub, inputDomain: "www.qiniu.com", expectedErr: "no such domain"},
		"unknown hub":               {inputHub: "unknown_hub", inputDomain: TestDomain, expectedErr: "hub not found"},
		"no hub":                    {inputHub: "", inputDomain: TestDomain, expectedErr: "Key: 'GetDomainInfoRequest.Hub' Error:Field validation for 'Hub' failed on the 'required' tag"},
		"no domain":                 {inputHub: TestHub, inputDomain: "", expectedErr: "Key: 'GetDomainInfoRequest.Domain' Error:Field validation for 'Domain' failed on the 'required' tag"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			manager := NewManager(ManagerConfig{APIHost: ApiHost, AccessKey: AccessKey, SecretKey: SecretKey})
			response, err := manager.GetDomainInfo(context.Background(), GetDomainInfoRequest{
				Hub:    tt.inputHub,
				Domain: tt.inputDomain,
			})
			CheckErr(ast, err, tt.expectedErr)
			if err == nil {
				t.Log(response)
				ast.Equal(response.Domain, tt.inputDomain)
			}
		})
	}
}

func TestManager_Bind_Unbind_Domains(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{APIHost: ApiHost, AccessKey: AccessKey, SecretKey: SecretKey})
	domain := "testbind-" + strconv.Itoa(rand.Intn(9999)) + ".fake.qiniu.com"

	err := manager.BindDomain(context.Background(), BindDomainRequest{
		Hub:    TestHub,
		Domain: domain,
		Type:   DomainLiveHLS,
	})
	ast.Nil(err)
	err = manager.UnbindDomain(context.Background(), UnbindDomainRequest{
		Hub:    TestHub,
		Domain: domain,
	})
	ast.Nil(err)
}

func TestManager_ErrorBindDomains(t *testing.T) {
	if SkipTest() || TestDomain == "" {
		t.SkipNow()
	}
	ast := assert.New(t)
	tests := map[string]struct {
		inputHub    string
		inputDomain string
		expectedErr string
	}{
		"existed domain": {inputHub: TestHub, inputDomain: TestDomain, expectedErr: "domain confict"},
		"invalid domain": {inputHub: TestHub, inputDomain: "notdomain", expectedErr: "invalid domain"},
		"unknown hub":    {inputHub: "unknown_hub", inputDomain: "testbind-" + strconv.Itoa(rand.Intn(9999)) + ".fake.qiniu.com", expectedErr: "hub not found"},
		"no hub":         {inputHub: "", inputDomain: "testbind-" + strconv.Itoa(rand.Intn(9999)) + ".fake.qiniu.com", expectedErr: "Key: 'BindDomainRequest.Hub' Error:Field validation for 'Hub' failed on the 'required' tag"},
		"no domain":      {inputHub: TestHub, inputDomain: "", expectedErr: "Key: 'BindDomainRequest.Domain' Error:Field validation for 'Domain' failed on the 'required' tag"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			manager := NewManager(ManagerConfig{APIHost: ApiHost, AccessKey: AccessKey, SecretKey: SecretKey})

			err := manager.BindDomain(context.Background(), BindDomainRequest{
				Hub:    tt.inputHub,
				Domain: tt.inputDomain,
				Type:   DomainLiveHLS,
			})
			CheckErr(ast, err, tt.expectedErr)
		})
	}
}

func TestManager_ErrUnbindDomains(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	tests := map[string]struct {
		inputHub    string
		inputDomain string
		expectedErr string
	}{
		"unknown domain": {inputHub: TestHub, inputDomain: "unknown.fake.qiniu.com", expectedErr: "not found"},
		"unknown hub":    {inputHub: "unknown_hub", inputDomain: "testbind-" + strconv.Itoa(rand.Intn(9999)) + ".fake.qiniu.com", expectedErr: "not found"},
		"no hub":         {inputHub: "", inputDomain: "testbind-" + strconv.Itoa(rand.Intn(9999)) + ".fake.qiniu.com", expectedErr: "Key: 'UnbindDomainRequest.Hub' Error:Field validation for 'Hub' failed on the 'required' tag"},
		"no domain":      {inputHub: TestHub, inputDomain: "", expectedErr: "Key: 'UnbindDomainRequest.Domain' Error:Field validation for 'Domain' failed on the 'required' tag"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			manager := NewManager(ManagerConfig{APIHost: ApiHost, AccessKey: AccessKey, SecretKey: SecretKey})
			err := manager.UnbindDomain(context.Background(), UnbindDomainRequest{
				Hub:    tt.inputHub,
				Domain: tt.inputDomain,
			})
			CheckErr(ast, err, tt.expectedErr)
		})
	}
}

func TestManager_BindVodDomain(t *testing.T) {
	if SkipTest() || TestVodDomain == "" {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{APIHost: ApiHost, AccessKey: AccessKey, SecretKey: SecretKey})
	err := manager.BindVodDomain(context.Background(), BindVodDomainRequest{
		Hub:       TestHub,
		VodDomain: TestVodDomain,
	})
	ast.Nil(err)
}

func TestManager_DomainCert(t *testing.T) {
	if SkipTest() || TestCertName == "" {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{APIHost: ApiHost, AccessKey: AccessKey, SecretKey: SecretKey})
	err := manager.SetDomainCert(context.Background(), SetDomainCertRequest{
		Hub:      TestHub,
		Domain:   TestDomain,
		CertName: TestCertName,
	})
	ast.Nil(err)
}

func TestManager_DomainUrlRewrite(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{APIHost: ApiHost, AccessKey: AccessKey, SecretKey: SecretKey})

	rules := []DomainURLRewriteRule{
		{Pattern: "live.qiniu.com/pili/(.+)", Replace: "live.qiniu.com/live-pili/${1}"},
		{Pattern: "live-origin.qiniu.com/pili/(.+)", Replace: "live-origin.qiniu.com/live-pili/${1}"},
		{Pattern: "(.+)/pili/(.+)/playlist.m3u8", Replace: "${1}/live-pili/${2}.m3u8"},
	}
	err := manager.SetDomainURLRewrite(context.Background(), SetDomainURLRewriteRequest{
		Hub:    TestHub,
		Domain: TestDomain,
		Rules:  rules,
	})
	ast.Nil(err)
}
