package api

import (
	"strconv"
)

// ----------------------------------------------------------

const (
	DefaultPointSize	= 0
	DefaultDx		= 10
	DefaultDy		= 10
)

// ----------------------------------------------------------

type Watermark struct {
	Font      string `json:"font"`
	Fill      string `json:"fill"`
	Text      string `json:"text"`
	Bucket    string `json:"bucket"`
	Dissolve  string `json:"dissolve"`
	Gravity   string `json:"gravity"`
	FontSize  int    `json:"fontsize"`	// 0 表示默认。单位: 缇，等于 1/20 磅
	Dx        int    `json:"dx"`
	Dy        int    `json:"dy"`
}

func (s *Service) GetWatermark(customer string) (ret Watermark, code int, err error) {

	params := map[string][]string{
		"customer": {customer},
	}
	code, err = s.Conn.CallWithForm(&ret, s.Host["eu"] + "/wmget", params)
	return
}

func (s *Service) SetWatermark(customer string, args *Watermark) (code int, err error) {

	params := map[string][]string{
		"text": {args.Text},
		"dx": {strconv.Itoa(args.Dx)},
		"dy": {strconv.Itoa(args.Dy)},
	}
	if customer != "" {
		params["customer"] = []string{customer}
	}
	if args.Font != "" {
		params["font"] = []string{args.Font}
	}
	if args.FontSize != 0 {
		params["fontsize"] = []string{strconv.Itoa(args.FontSize)}
	}
	if args.Fill != "" {
		params["fill"] = []string{args.Fill}
	}
	if args.Bucket != "" {
		params["bucket"] = []string{args.Bucket}
	}
	if args.Dissolve != "" {
		params["dissolve"] = []string{args.Dissolve}
	}
	if args.Gravity != "" {
		params["gravity"] = []string{args.Gravity}
	}
	return s.Conn.CallWithForm(nil, s.Host["eu"] + "/wmset", params)
}

