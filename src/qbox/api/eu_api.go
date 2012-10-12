package eu

import (
	"strconv"
	"net/http"
	"qbox/api"
	"qbox/utils/rpc"
	. "qbox.us/api/conf"
)

// ----------------------------------------------------------

const (
	NoSuchEntry  = 612 // 指定的 Entry 不存在或已经 Deleted
)

var (
	ENoSuchEntry  = api.RegisterError(NoSuchEntry, "no such file or directory")
)

// ----------------------------------------------------------

type Service struct {
	Conn rpc.Client
}

func New(t http.RoundTripper) Service {
	client := &http.Client{Transport: t}
	return Service{rpc.Client{client}}
}

// ----------------------------------------------------------

const (
	DefaultPointSize	= 0
	DefaultDx			= 10
	DefaultDy			= 10
)

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

func (p *Service) GetWatermark(customer string) (ret Watermark, code int, err error) {

	params := map[string][]string{
		"customer": {customer},
	}
	code, err = p.Conn.CallWithForm(&ret, EU_HOST+"/wmget", params)
	return
}

func (p *Service) SetWatermark(customer string, args *Watermark) (code int, err error) {

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
	code, err = p.Conn.CallWithForm(nil, EU_HOST+"/wmset", params)
	return
}

