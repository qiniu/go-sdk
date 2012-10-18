package rs

import (
	"io"
	"errors"
	"strconv"
	"time"
	"net/http"
	. "qbox/api"
	"qbox/rpc"
)



type Service struct {
	*Config
	Conn rpc.Client
}


func New(c *Config, t http.RoundTripper) (s *Service, err error) {
	if c == nil {
		err = errors.New("Must have a config file")
		return
	}
	if t == nil {
		t = http.DefaultTransport
	}
	client := &http.Client{Transport: t}
	s = &Service{c, rpc.Client{client}}
	return
}



type PutRet struct {
	Hash string `json:"hash"`
}

type GetRet struct {
	URL      string `json:"url"`
	Hash     string `json:"hash"`
	MimeType string `json:"mimeType"`
	Fsize    int64  `json:"fsize"`
	Expiry   int64  `json:"expires"`
}

type Entry struct {
	Hash     string `json:"hash"`
	Fsize    int64  `json:"fsize"`
	PutTime  int64  `json:"putTime"`
	MimeType string `json:"mimeType"`
}

func (s *Service) Put(
	entryURI, mimeType string, body io.Reader, bodyLength int64) (ret PutRet, code int, err error) {

	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	url := s.Host["io"] + "/rs-put/" + EncodeURI(entryURI) + "/mimeType/" + EncodeURI(mimeType)
	code, err = s.Conn.CallWith64(&ret, url, "application/octet-stream", body, bodyLength)
	return
}

func (s *Service) Get(entryURI, base, attName string, expires int) (data GetRet, code int, err error) {
	url := s.Host["rs"] + "/get/" + EncodeURI(entryURI)
	if base != "" {
		url += "/base/" + base
	}
	if attName != "" {
		url += "/attName/" + EncodeURI(attName)
	}
	if expires > 0 {
		url += "/expires/" + strconv.Itoa(expires)
	}
	code, err = s.Conn.Call(&data, url)
	if code/100 == 2 {
		data.Expiry += time.Now().Unix()
	}
	return
}

func (s *Service) Stat(entryURI string) (entry Entry, code int, err error) {
	code, err = s.Conn.Call(&entry, s.Host["rs"] + "/stat/" + EncodeURI(entryURI))
	return
}

func (s *Service) Delete(entryURI string) (code int, err error) {
	return s.Conn.Call(nil, s.Host["rs"] + "/delete/" + EncodeURI(entryURI))
}

func (s *Service) Mkbucket(bucketname string) (code int, err error) {
	return s.Conn.Call(nil, s.Host["rs"] + "/mkbucket/" + bucketname)
}

func (s *Service) Drop(entryURI string) (code int, err error) {
	return s.Conn.Call(nil, s.Host["rs"] + "/drop/" + entryURI)
}

func (s *Service) Move(entryURISrc, entryURIDest string) (code int, err error) {
	return s.Conn.Call(nil, s.Host["rs"] + "/move/" + EncodeURI(entryURISrc) + "/" + EncodeURI(entryURIDest))
}

func (s *Service) Copy(entryURISrc, entryURIDest string) (code int, err error) {
	return s.Conn.Call(nil, s.Host["rs"] + "/copy/" + EncodeURI(entryURISrc) + "/" + EncodeURI(entryURIDest))
}

func (s *Service) Publish(domain, table string) (code int, err error) {
	return s.Conn.Call(nil, s.Host["rs"] + "/publish/" + EncodeURI(domain) + "/from/" + table)
}

func (s *Service) Unpublish(domain string) (code int, err error) {
	return s.Conn.Call(nil, s.Host["rs"] + "/unpublish/" + EncodeURI(domain))
}

// ----------------------------------------------------------

type BatchRet struct {
	Data  interface{} `json:"data"`
	Code  int         `json:"code"`
	Error string      `json:"error"`
}

type Batcher struct {
	op  []string
	ret []BatchRet
}

func (b *Batcher) operate(entryURI string, method string) {
	b.op = append(b.op, method+EncodeURI(entryURI))
	b.ret = append(b.ret, BatchRet{})
}

func (b *Batcher) operate2(entryURISrc, entryURIDest string, method string) {
	b.op = append(b.op, method+EncodeURI(entryURISrc)+"/"+EncodeURI(entryURIDest))
	b.ret = append(b.ret, BatchRet{})
}

func (b *Batcher) Delete(entryURI string) {
	b.operate(entryURI, "/delete/")
}

func (b *Batcher) Move(entryURISrc, entryURIDest string) {
	b.operate2(entryURISrc, entryURIDest, "/move/")
}

func (b *Batcher) Copy(entryURISrc, entryURIDest string) {
	b.operate2(entryURISrc, entryURIDest, "/copy/")
}

func (b *Batcher) Reset() {
	b.op = nil
	b.ret = nil
}

func (b *Batcher) Len() int {
	return len(b.op)
}

func (b *Batcher) Do(s *Service) (ret []BatchRet, code int, err error) {
	code, err = s.Conn.CallWithForm(&b.ret, s.Host["rs"] + "/batch", map[string][]string{"op": b.op})
	ret = b.ret
	return
}