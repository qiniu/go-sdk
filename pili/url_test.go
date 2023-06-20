//go:build unit
// +build unit

package pili

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManager_URL(t *testing.T) {
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: "mockAK", SecretKey: "mockSK"})

	url1 := manager.url("/v2/hubs")
	ast.Equal("http://pili.qiniuapi.com/v2/hubs", url1)

	url1 = manager.url("/v2/hubs/%s/streams", "test")
	ast.Equal("http://pili.qiniuapi.com/v2/hubs/test/streams", url1)

	query := url.Values{}
	setQuery(query, "foo", "bar")
	setQuery(query, "num", 0)
	setQuery(query, "boolean", false)
	url1 = manager.url("/v2/hubs/%s/streams", "test", query)
	ast.Equal("http://pili.qiniuapi.com/v2/hubs/test/streams?foo=bar", url1)

	url1 = manager.url("/v2/hubs/%s/streams", "test", query)
	ast.Equal("http://pili.qiniuapi.com/v2/hubs/test/streams?foo=bar", url1)

	url1 = manager.url("/v2/hubs/%s/security", "test")
	ast.Equal("http://pili.qiniuapi.com/v2/hubs/test/security", url1)

	url1 = manager.url("/v2/hubs/%s/domains/%s", "test", "testDomain")
	ast.Equal("http://pili.qiniuapi.com/v2/hubs/test/domains/testDomain", url1)
}

func TestGenerateURL(t *testing.T) {
	ast := assert.New(t)
	hub := "hub"
	domain := "domain.com"
	streamTitle := "streamTitle"

	publishURL := RTMPPublishURL(hub, domain, streamTitle)
	ast.Equal("rtmp://domain.com/hub/streamTitle", publishURL)

	publishURL = SRTPublishURL(hub, domain, streamTitle)
	ast.Equal("srt://domain.com:1935?streamid=#!::h=hub/streamTitle,m=publish,domain=domain.com", publishURL)

	playURL := RTMPPlayURL(hub, domain, streamTitle)
	ast.Equal("rtmp://domain.com/hub/streamTitle", playURL)

	playURL = HLSPlayURL(hub, domain, streamTitle)
	ast.Equal("https://domain.com/hub/streamTitle.m3u8", playURL)

	playURL = HDLPlayURL(hub, domain, streamTitle)
	ast.Equal("https://domain.com/hub/streamTitle.flv", playURL)
}

func TestSignPublishURL(t *testing.T) {
	ast := assert.New(t)

	type instance struct {
		publishURL         string
		signPublishURLArgs SignPublishURLArgs
		resultURL          string
		err                error
	}

	instances := []instance{
		{
			// 静态鉴权
			publishURL: "rtmp://publish.domain.com/hub/streamTitle?xyz=custom&key=abc",
			signPublishURLArgs: SignPublishURLArgs{
				SecurityType: SecurityTypeStatic,
				PublishKey:   "test123",
			},
			resultURL: "rtmp://publish.domain.com/hub/streamTitle?key=test123&xyz=custom",
			err:       nil,
		},
		{
			// 限时鉴权
			publishURL: "rtmp://publish.domain.com/hub/streamTitle?xyz=custom&token=abc",
			signPublishURLArgs: SignPublishURLArgs{
				SecurityType: SecurityTypeExpiry,
				PublishKey:   "test123",
				ExpireAt:     1634745600,
			},
			resultURL: "rtmp://publish.domain.com/hub/streamTitle?expire=1634745600&xyz=custom&token=fH7U5UgXdClDfeGmUKc6wrNzlho=",
			err:       nil,
		},
		{
			// 限时鉴权SK
			publishURL: "rtmp://publish.domain.com/testhub/teststreamtitle",
			signPublishURLArgs: SignPublishURLArgs{
				SecurityType: SecurityTypeExpirySK,
				ExpireAt:     1584522520,
				AccessKey:    "7O7hf7Ld1RrC_fpZdFvU8aCgOPuhw2K4eapYOdII",
				SecretKey:    "312ae9gd2BrCfpTdF4U8aIg9Puh62K4eEGY72Ea_",
			},
			resultURL: "rtmp://publish.domain.com/testhub/teststreamtitle?e=1584522520&token=7O7hf7Ld1RrC_fpZdFvU8aCgOPuhw2K4eapYOdII:NfI2OWGCMdFDTLOfeUd-zSPVrFY=",
			err:       nil,
		},
		{
			// 动态鉴权
			publishURL: "rtmp://publish.domain.com/hub/streamTitle",
			signPublishURLArgs: SignPublishURLArgs{
				SecurityType: SecurityTypeDynamic,
				PublishKey:   "test123",
				Nonce:        123,
			},
			resultURL: "rtmp://publish.domain.com/hub/streamTitle?nonce=123&token=NFFIx_Hi5MdyJGj3Aubsf6jbma8=",
			err:       nil,
		},
		{
			// SRT + 限时鉴权
			publishURL: "srt://publish.domain.com?streamid=#!::h=hub/streamTitle,m=publish,domain=publish.domain.com,xyz=custom",
			signPublishURLArgs: SignPublishURLArgs{
				SecurityType: SecurityTypeExpiry,
				PublishKey:   "test123",
				ExpireAt:     1634745600,
			},
			resultURL: "srt://publish.domain.com:1935?streamid=#!::h=hub/streamTitle,domain=publish.domain.com,expire=1634745600,m=publish,xyz=custom,token=GSmK7wau9MZ0rBCV-jDLmG4fUhM=",
			err:       nil,
		},
	}

	for i, inst := range instances {
		resultURL, err := SignPublishURL(inst.publishURL, inst.signPublishURLArgs)
		ast.Equal(inst.err, err, fmt.Sprintf("index:%d args:%v", i, inst))
		if err == nil {
			ast.Equal(inst.resultURL, resultURL, fmt.Sprintf("index:%d args:%v", i, inst))
		}
	}
}

func TestSignPlayURL(t *testing.T) {
	ast := assert.New(t)

	type instance struct {
		playURL         string
		signPlayURLArgs SignPlayURLArgs
		resultURL       string
		err             error
	}

	instances := []instance{
		{
			// 时间戳防盗链开始时间限制
			playURL: "https://play.domain.com/hub/streamTitle",
			signPlayURLArgs: SignPlayURLArgs{
				SecurityType: SecurityTypeTsStartMD5,
				Key:          "test123",
				Timestamp:    1634745600,
				Rule:         "$(key)$(path)$(_t)",
				TsPart:       "t",
				SignPart:     "sign",
			},
			resultURL: "https://play.domain.com/hub/streamTitle?sign=b1a151275da51ce22f50dd97df3cdc86&t=61703d00",
			err:       nil,
		},
		{
			// 时间戳防盗链结束时间限制
			playURL: "https://play.domain.com/hub/streamTitle",
			signPlayURLArgs: SignPlayURLArgs{
				SecurityType: SecurityTypeTsExpireMD5,
				Key:          "test123",
				Timestamp:    1634745600,
				Rule:         "$(key)$(path)$(_t)",
				TsPart:       "t",
				TsBase:       10,
				SignPart:     "sign",
			},
			resultURL: "https://play.domain.com/hub/streamTitle?sign=2ddebda4405e928153a80163f6c49ee0&t=1634745600",
			err:       nil,
		},
	}

	for i, inst := range instances {
		resultURL, err := SignPlayURL(inst.playURL, inst.signPlayURLArgs)
		ast.Equal(inst.err, err, fmt.Sprintf("index:%d args:%v", i, inst))
		if err == nil {
			ast.Equal(inst.resultURL, resultURL, fmt.Sprintf("index:%d args:%v", i, inst))
		}
	}
}
