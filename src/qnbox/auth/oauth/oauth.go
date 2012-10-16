// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The oauth package provides support for making
// OAuth2-authenticated HTTP requests.
//
// Example usage:
//
//	// Specify your configuration. (typically as a global variable)
//	var config = &oauth.Config{
//		ClientId:     YOUR_CLIENT_ID,
//		ClientSecret: YOUR_CLIENT_SECRET,
//		Scope:        "https://www.googleapis.com/auth/buzz",
//		AuthURL:      "https://accounts.google.com/o/oauth2/auth",
//		TokenURL:     "https://accounts.google.com/o/oauth2/token",
//		RedirectURL:  "http://you.example.org/handler",
//	}
//
//	// A landing page redirects to the OAuth provider to get the auth code.
//	func landing(w http.ResponseWriter, r *http.Request) {
//		http.Redirect(w, r, config.AuthCodeURL("foo"), http.StatusFound)
//	}
//
//	// The user will be redirected back to this handler, that takes the
//	// "code" query parameter and Exchanges it for an access token.
//	func handler(w http.ResponseWriter, r *http.Request) {
//		t := &oauth.Transport{Config: config}
//		t.Exchange(r.FormValue("code"))
//		// The Transport now has a valid Token. Create an *http.Client
//		// with which we can make authenticated API requests.
//		c := t.Client()
//		c.Post(...)
//		// ...
//		// btw, r.FormValue("state") == "foo"
//	}
//
package oauth

import (
	"time"
	"errors"
	"strings"
	"encoding/json"
	"net/http"
	"net/url"
)

const (
	InvalidArgs        = 400  // Bad input parameter. Error message should indicate which one and why.
	UnexceptedResponse = 9998 // 非预期的输出。see api.UnexceptedResponse
)

// -----------------------------------------------------------------------------------------
// class Config

// Config is the configuration of an OAuth consumer.
type Config struct {
	ClientId     string
	ClientSecret string
	Scope        string
	AuthURL      string
	TokenURL     string
	RedirectURL  string // Defaults to out-of-band mode if empty.
	Agent        string
}

// AuthCodeURL returns a URL that the end-user should be redirected to,
// so that they may obtain an authorization code.
func (c *Config) AuthCodeURL(state string) string {
	url_, err := url.Parse(c.AuthURL)
	if err != nil {
		panic("AuthURL malformed: " + err.Error())
	}
	q := url.Values(map[string][]string{
		"response_type": {"code"},
		"client_id":     {c.ClientId},
		"redirect_uri":  {c.redirectURL()},
		"scope":         {c.Scope},
		"state":         {state},
	}).Encode()
	if url_.RawQuery == "" {
		url_.RawQuery = q
	} else {
		url_.RawQuery += "&" + q
	}
	return url_.String()
}

func (c *Config) redirectURL() string {
	if c.RedirectURL != "" {
		return c.RedirectURL
	}
	return "oob"
}

// -----------------------------------------------------------------------------------------
// class Transport

// Token contains an end-user's tokens.
// This is the data you must store to persist authentication.
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenExpiry  int64  `json:"expires_in"`
}

type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorCode        int    `json:"error_code"`
	ErrorDescription string `json:"error_description"`
}

// Transport implements http.RoundTripper. When configured with a valid
// Config and Token it can be used to make authenticated HTTP requests.
//
//	t := &oauth.Transport{Config:config}
//      t.Exchange(code)
//      // t now contains a valid Token
//	r, _, err := t.Client().Get("http://example.org/url/requiring/auth")
//
// It will automatically refresh the Token if it can,
// updating the supplied Token in place.
type Transport struct {
	*Config
	*Token

	// Transport is the HTTP transport to use when making requests.
	// It will default to http.DefaultTransport if nil.
	// (It should never be an oauth.Transport.)
	Transport http.RoundTripper
}


func NewTransport(cfg *Config, tok *Token, transport http.RoundTripper) *Transport {
	return &Transport{cfg, tok, transport}
}





// Exchange takes user & passwd and gets access Token from the remote server.
func (t *Transport) ExchangeByPassword(user string, passwd string) (tok *Token, code int, err error) {
	if t.Config == nil {
		return nil, InvalidArgs, errors.New("no Config supplied")
	}
	tok = new(Token)
	code, err = t.updateToken(tok, map[string][]string{
		"grant_type": {"password"},
		"username":   {user},
		"password":   {passwd},
		"scope":      {t.Scope},
	})
	if err == nil {
		t.Token = tok
	}
	return
}

// Exchange takes user & passwd and gets access Token from the remote server.
func (t *Transport) ExchangeByPasswordEx(user, passwd string, params map[string][]string) (tok *Token, code int, err error) {
	if t.Config == nil {
		return nil, InvalidArgs, errors.New("no Config supplied")
	}
	tok = new(Token)
	params["grant_type"] = []string{"password"}
	params["username"] = []string{user}
	params["password"] = []string{passwd}
	params["scope"] = []string{t.Scope}
	code, err = t.updateToken(tok, params)
	if err == nil {
		t.Token = tok
	}
	return
}

func (t *Transport) ExchangeByRefreshToken(refreshToken string) (tok *Token, code int, err error) {
	if t.Config == nil {
		return nil, InvalidArgs, errors.New("no Config supplied")
	}
	tok = new(Token)
	code, err = t.updateToken(tok, map[string][]string{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	})
	if err == nil {
		t.Token = tok
	}
	return
}

func (t *Transport) ExchangeByRefreshTokenEx(refreshToken string, params map[string][]string) (tok *Token, code int, err error) {
	if t.Config == nil {
		return nil, InvalidArgs, errors.New("no Config supplied")
	}
	tok = new(Token)
	params["grant_type"] = []string{"refresh_token"}
	params["refresh_token"] = []string{refreshToken}
	code, err = t.updateToken(tok, params)
	if err == nil {
		t.Token = tok
	}
	return
}

// Exchange takes a code and gets access Token from the remote server.
func (t *Transport) Exchange(code string) (tok *Token, code1 int, err error) {
	if t.Config == nil {
		return nil, InvalidArgs, errors.New("no Config supplied")
	}
	tok = new(Token)
	code1, err = t.updateToken(tok, map[string][]string{
		"grant_type":   {"authorization_code"},
		"redirect_uri": {t.redirectURL()},
		"scope":        {t.Scope},
		"code":         {code},
	})
	if err == nil {
		t.Token = tok
	}
	return
}

// RoundTrip executes a single HTTP transaction using the Transport's
// Token as authorization headers.
func (t *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	if t.Config == nil {
		return nil, errors.New("no Config supplied")
	}
	if t.Token == nil {
		return nil, errors.New("no Token supplied")
	}

	// Refresh the Token if it has expired.
	if t.expired() {
		if _, err := t.refresh(); err != nil {
			return nil, err
		}
	}

	// Make the HTTP request.
	req.Header.Set("Authorization", "Bearer "+t.AccessToken)
	return t.transport().RoundTrip(req)
}

func (t *Token) expired() bool {
	if t.TokenExpiry == 0 {
		return false
	}
	return t.TokenExpiry <= seconds()
}

// Client returns an *http.Client that makes OAuth-authenticated requests.
func (t *Transport) Client() *http.Client {
	return &http.Client{Transport: t}
}

func (t *Transport) transport() http.RoundTripper {
	if t.Transport != nil {
		return t.Transport
	}
	return http.DefaultTransport
}

func (t *Transport) refresh() (code int, err error) {
	return t.updateToken(t.Token, map[string][]string{
		"grant_type":    {"refresh_token"},
		"refresh_token": {t.RefreshToken},
	})
}

func (t *Transport) updateToken(tok *Token, form map[string][]string) (code int, err error) {

	form["client_id"] = []string{t.ClientId}

	req, err := http.NewRequest("POST", t.TokenURL, strings.NewReader(url.Values(form).Encode()))
	if err != nil {
		return
	}

	userAgent := ""
	if len(form["user_agent"]) == 0 {
		if t.Agent != "" {
			userAgent = t.Agent
		}
	} else {
		userAgent = form["user_agent"][0]
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}

	if len(form["remote_ip"]) > 0 {
		req.Header.Set("X-Forwarded-For", form["remote_ip"][0])
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Form = form
	r, err := (&http.Client{Transport: t.transport()}).Do(req)
	//r, err := (&http.Client{Transport: t.transport()}).PostForm(t.TokenURL, form)
	if err != nil {
		return
	}

	defer r.Body.Close()
	if r.StatusCode != 200 {
		code = r.StatusCode
		var errReceiver ErrorResponse
		json.NewDecoder(r.Body).Decode(&errReceiver)
		if errReceiver.ErrorCode != 0 {
			code = errReceiver.ErrorCode
		}
		if errReceiver.Error != "" {
			err = errors.New(errReceiver.Error)
		} else {
			err = errors.New("invalid response: " + r.Status)
		}
		return
	}
	if err = json.NewDecoder(r.Body).Decode(tok); err != nil {
		return UnexceptedResponse, err
	}
	if tok.TokenExpiry != 0 {
		tok.TokenExpiry += seconds()
	}
	return 200, nil
}

func seconds() int64 {
	return time.Now().Unix()
}

