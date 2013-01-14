package digest_auth

import (
	"encoding/base64"
	"encoding/json"
	"crypto/sha1"
	"crypto/hmac"
)

// ----------------------------------------------------------

func Sign(key string, secret []byte, b []byte) []byte {

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

func SignJson(key string, secret []byte, data interface{}) string {

	b, _ := json.Marshal(data)
	return string(Sign(key, secret, b))
}

// ----------------------------------------------------------

