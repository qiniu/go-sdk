package api

import (
	"strings"
	"strconv"
	"../utils/errcode"
)

// ------------------------------------------------------------------------------------------

const (
	TooManyKeys        = 700 // UC: 太多AccessKey
	NotFound           = 701 // UC: 没有发现此AccessKey
)

var (
	ETooManyKeys       = errcode.RegisterError(TooManyKeys, "too many keys")
	ENotFound          = errcode.RegisterError(NotFound, "not found")
)

// ------------------------------------------------------------------------------------------

type AppInfo struct {
	Key string		`json:"key"`
	Secret string	`json:"secret"`
	Key2 string		`json:"key2"`
	Secret2 string	`json:"secret2"`
	AppId uint32	`json:"appId"`
}

func (s *Service) AppInfo(app string) (info AppInfo, code int, err error) {

	params := map[string][]string{
		"app": {app},
	}
	code, err = s.Conn.CallWithForm(&info, s.Host["uc"] + "/appInfo", params)
	return
}

type AccessInfo struct {
	Key string		`json:"key"`
	Secret string	`json:"secret"`
}

func (s *Service) NewAccess(app string) (info AccessInfo, code int, err error) {

	params := map[string][]string{
		"app": {app},
	}
	code, err = s.Conn.CallWithForm(&info, s.Host["uc"] + "/newAccess", params)
	return
}

func (s *Service) DeleteAccess(app string, accessKey string) (code int, err error) {

	params := map[string][]string{
		"app": {app},
		"key": {accessKey},
	}
	return s.Conn.CallWithForm(nil, s.Host["uc"] + "/deleteAccess", params)
}

func (s *Service) AntiLeechMode(bucket string, mode int) (code int, err error) {
	param := map[string][]string {
		"bucket": {bucket},
		"mode": {strconv.Itoa(mode)},
	}
	url := s.Host["uc"] + "/antiLeechMode"
	return s.Conn.CallWithForm(nil, url, param)
}

func (s *Service) AddAntiLeech(bucket string, mode int, pattern string) (code int, err error) {
	param := map[string][]string {
		"bucket": {bucket},
		"mode": {strconv.Itoa(mode)},
		"action": {"add"},
		"pattern": {pattern},
	}
	url := s.Host["uc"] + "/referAntiLeech"
	return s.Conn.CallWithForm(nil, url, param)
}

func (s *Service) CleanCache(bucket string) (code int, err error) {
	param := map[string][]string {
		"bucket": {bucket},
	}
	url := s.Host["uc"] + "/refreshBucket"
	return s.Conn.CallWithForm(nil, url, param)
}

func (s *Service) DelAntiLeech(bucket string, mode int, pattern string) (code int, err error) {
	param := map[string][]string {
		"bucket": {bucket},
		"mode": {strconv.Itoa(mode)},
		"action": {"del"},
		"pattern": {pattern},
	}
	url := s.Host["uc"] + "/referAntiLeech"
	return s.Conn.CallWithForm(nil, url, param)
}

func (s *Service) SetImagePreviewStyle(name string, style string) (code int, err error) {

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
			code, err = errcode.InvalidArgs, errcode.EInvalidArgs
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
			code, err = errcode.InvalidArgs, errcode.EInvalidArgs
			return
		}
		params[ps[i][:pos]] = []string{ps[i][pos+1:]}
	}
	code, err = s.Conn.CallWithForm(nil, s.Host["uc"] + "/setImagePreviewStyle", params)
	return
}

// ------------------------------------------------------------------------------------------

