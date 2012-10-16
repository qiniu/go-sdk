package api

import (
	"io/ioutil"
	"net/http"
	"encoding/json"
	"encoding/base64"
	"../utils/rpc"
)


type Config struct {
	Host map[string]string `json:"HOST"`

	Access_key string `json:"QBOX_ACCESS_KEY"`
	Secret_key string `json:"QBOX_SECRET_KEY"`
	BlockBits uint `json:"BLOCK_BITS"`
	RPutChunkSize int64 `json:"RPUT_CHUNK_SIZE"`
	RPutRetryTimes int `json:"RPUT_RETRY_TIMES"`

	Client string `json:"CLIENT"`
	ClientId string `json:"CLIENT_ID"`
	ClientSecret string `json:"CLIENT_SECRET"`

	RedirectURI string `json:"REDIRECT_URI"`
	AuthorizationEndPoint string `json:"AUTHORIZATION_ENDPOINT"`
	TokenEndPoint string `json:"TOKEN_ENDPOINT"`
}

type Service struct {
	Config
	Conn rpc.Client
}


func loadConfig(filename string) (c *Config) {
	var conf Config

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	err = json.Unmarshal(b, &conf)
	if err != nil {
		return
	}
	c = &conf
	return
}


func New(c Config, t http.RoundTripper) *Service {
	if t == nil {
		t = http.DefaultTransport
	}
	client := &http.Client{Transport: t}
	return &Service{c, rpc.Client{client}}
}

func EncodeURI(uri string) string {
	return base64.URLEncoding.EncodeToString([]byte(uri))
}