package uc

import (
	"strings"
	"net/http"
	"qbox/api"
	"qbox/utils/rpc"
	. "qbox/api/conf"
)

// ------------------------------------------------------------------------------------------

const (
	TooManyKeys        = 700 // UC: 太多AccessKey
	NotFound           = 701 // UC: 没有发现此AccessKey
)

var (
	ETooManyKeys       = api.RegisterError(TooManyKeys, "too many keys")
	ENotFound          = api.RegisterError(NotFound, "not found")
)

// ------------------------------------------------------------------------------------------

type Service struct {
	Conn rpc.Client
}

func New(t http.RoundTripper) *Service {
	return &Service{ rpc.Client{ &http.Client{Transport: t} } }
}

// ------------------------------------------------------------------------------------------

type AppInfo struct {
	Key string		`json:"key"`
	Secret string	`json:"secret"`
	Key2 string		`json:"key2"`
	Secret2 string	`json:"secret2"`
	AppId uint32	`json:"appId"`
}

func (r *Service) AppInfo(app string) (info AppInfo, code int, err error) {

	params := map[string][]string{
		"app": {app},
	}
	code, err = r.Conn.CallWithForm(&info, UC_HOST+"/appInfo", params)
	return
}

type AccessInfo struct {
	Key string		`json:"key"`
	Secret string	`json:"secret"`
}

func (r *Service) NewAccess(app string) (info AccessInfo, code int, err error) {

	params := map[string][]string{
		"app": {app},
	}
	code, err = r.Conn.CallWithForm(&info, UC_HOST+"/newAccess", params)
	return
}

func (r *Service) DeleteAccess(app string, accessKey string) (code int, err error) {

	params := map[string][]string{
		"app": {app},
		"key": {accessKey},
	}
	code, err = r.Conn.CallWithForm(nil, UC_HOST+"/deleteAccess", params)
	return
}

// ------------------------------------------------------------------------------------------

/*
Request:
	POST /setImagePreviewStyle?name=<Name>&mode=square&size=<Size> [&q=<Quality>&sharpen=<Sharpen>]
	POST /setImagePreviewStyle?name=<Name>&height=<Size> [&q=<Quality>&sharpen=<Sharpen>]
	POST /setImagePreviewStyle?name=<Name>&width=<Size> [&q=<Quality>&sharpen=<Sharpen>]
	POST /setImagePreviewStyle?name=<Name>&width=<Width>&height=<Height> [&q=<Quality>&sharpen=<Sharpen>]
Style:
	square:<Size>;q:<Quality>;sharpen:<Sharpen>
	<Width>x;q:<Quality>;sharpen:<Sharpen>
	x<Height>;q:<Quality>;sharpen:<Sharpen>
	<Width>x<Height>;q:<Quality>;sharpen:<Sharpen>
*/
func (r *Service) SetImagePreviewStyle(name string, style string) (code int, err error) {

	params := map[string][]string{
		"name": {name},
	}
	ps := strings.Split(style, ";")
	ps0 := ps[0]
	if strings.HasPrefix(ps0, "square:") {
		params["mode"] = []string{"square"}
		params["size"] = []string{ps0[7:]}
	} else {
		pos := strings.Index(ps0, "x")
		if pos == -1 {
			code, err = api.InvalidArgs, api.EInvalidArgs
			return
		}
		width := ps0[:pos]
		height := ps0[pos+1:]
		if width != "" {
			params["width"] = []string{width}
		}
		if height != "" {
			params["height"] = []string{height}
		}
	}
	for i := 1; i < len(ps); i++ {
		pos := strings.Index(ps[i], ":")
		if pos == -1 {
			code, err = api.InvalidArgs, api.EInvalidArgs
			return
		}
		params[ps[i][:pos]] = []string{ps[i][pos+1:]}
	}
	code, err = r.Conn.CallWithForm(nil, UC_HOST+"/setImagePreviewStyle", params)
	return
}

// ------------------------------------------------------------------------------------------

