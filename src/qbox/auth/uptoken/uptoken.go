package uptoken


import (
	"time"
	"net/http"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
)

const (
	EXPIRES_TIME = 3600
)


type Transport struct {
	Uptoken string
	transport http.RoundTripper
}


type AuthPolicy struct {
	Scope            string `json:"scope"`
	CallbackUrl      string `json:"callbackUrl"`
	ReturnUrl		 string `json:"returnUrl"`
	Customer         string `json:"customer"`
	Deadline         uint32 `json:"deadline"` // 截止时间（以秒为单位）
}

func MakeAuthToken(key, secret []byte, auth *AuthPolicy) []byte {

	if auth.Deadline == 0 {
		auth.Deadline = uint32(time.Now().Unix()) + uint32(EXPIRES_TIME)
	}

	b, _ := json.Marshal(auth)
	blen := base64.URLEncoding.EncodedLen(len(b))

	nkey := len(key)
	ret := make([]byte, nkey+30+blen)

	base64.URLEncoding.Encode(ret[nkey+30:], b)

	h := hmac.New(sha1.New, secret)
	h.Write(ret[nkey+30:])
	digest := h.Sum(nil)

	copy(ret, key)
	ret[nkey] = ':'
	base64.URLEncoding.Encode(ret[nkey+1:], digest)
	ret[nkey+29] = ':'

	return ret
}



func MakeAuthTokenString(key, secret string, auth *AuthPolicy) string {

	token := MakeAuthToken([]byte(key), []byte(secret), auth)
	return string(token)
}


func (t *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	req.Header.Set("Authorization", t.Uptoken)
	return t.transport.RoundTrip(req)
}

func NewTransport(uptoken string, transport http.RoundTripper) *Transport {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &Transport{"UpToken " + uptoken, transport}
}