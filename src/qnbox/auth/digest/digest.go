package digest

import (
	"io"
	"errors"
	"io/ioutil"
	"net/http"
	"encoding/base64"
	"crypto/sha1"
	"crypto/hmac"
	"qnbox/utils/bytes"
)


// ---------------------------------------------------------------------------------------

// Transport implements http.RoundTripper. When configured with a valid
// Config and Token it can be used to make authenticated HTTP requests.
//
//	c := NewClient(token, nil)
//	r, _, err := c.Get("http://example.org/url/requiring/auth")
//
type Transport struct {
	key string
	secret []byte

	// Transport is the HTTP transport to use when making requests.
	// It will default to http.DefaultTransport if nil.
	// (It should never be an oauth.Transport.)
	transport http.RoundTripper
}

func incBody(req *http.Request) bool {
	if req.Body == nil {
		return false
	}
	if ct, ok := req.Header["Content-Type"]; ok {
		switch ct[0] {
		case "application/x-www-form-urlencoded":
			return true
		}
	}
	return false
}


func Checksum(req *http.Request, key []byte, incbody bool) (digest string, err error) {

	h := hmac.New(sha1.New, key)

	u := req.URL
	data := u.Path
	if u.RawQuery != "" {
		data += "?" + u.RawQuery
	}
	io.WriteString(h, data + "\n")

	if incbody {
		b, _ := ioutil.ReadAll(req.Body)
		req.Body = bytes.NewReader(b)
		h.Write(b)
	}
	digest = base64.URLEncoding.EncodeToString(h.Sum(nil))
	return
}



// RoundTrip executes a single HTTP transaction using the Transport's
// Token as authorization headers.
func (t *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	digest, err := Checksum(req, t.secret, incBody(req))
	if err != nil {
		err = errors.New("Digest_Auth faild.Transport.RoundTrip")
		return
	}
	token := t.key + ":" + digest
	req.Header.Set("Authorization", "QBox " + token)
	return t.transport.RoundTrip(req)
}

func NewTransport(key, secret string, transport http.RoundTripper) *Transport {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &Transport{key, []byte(secret), transport}
}


// ---------------------------------------------------------------------------------------

