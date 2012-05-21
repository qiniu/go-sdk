package rs

import (
	"io"
	"time"
	"strconv"
	"net/http"
	"encoding/base64"
	"qbox/api"
	"qbox/rpc"
	. "qbox/api/conf"
)

// ----------------------------------------------------------

const (
	FileModified = 608 // RS: 文件被修改（see fs.GetIfNotModified）
	NoSuchEntry  = 612 // RS: 指定的 Entry 不存在或已经 Deleted
	EntryExists  = 614 // RS: 要创建的 Entry 已经存在
)

var (
	EFileModified      = api.Errno(FileModified)
	ENoSuchEntry       = api.Errno(NoSuchEntry)
	EEntryExists       = api.Errno(EntryExists)
)

func init() {
	api.RegisterErrno([]api.ErrnoMsg{
		{FileModified, "file modified"},
		{NoSuchEntry, "no such file or directory"},
		{EntryExists, "file exists"},
	})
}

// ----------------------------------------------------------

func EncodeURI(uri string) string {
	return base64.URLEncoding.EncodeToString([]byte(uri))
}

// ----------------------------------------------------------

type Service struct {
	Conn rpc.Client
}

func New(t http.RoundTripper) *Service {
	client := &http.Client{Transport: t}
	return &Service{rpc.Client{client}}
}

// ----------------------------------------------------------

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

func (rs *Service) Put(
	entryURI, mimeType string, body io.Reader, bodyLength int64) (ret PutRet, code int, err error) {

	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	url := IO_HOST + "/rs-put/" + EncodeURI(entryURI) + "/mimeType/" + EncodeURI(mimeType)
	code, err = rs.Conn.CallWith64(&ret, url, "application/octet-stream", body, bodyLength)
	return
}

func (rs *Service) Get(entryURI string, attName string) (data GetRet, code int, err error) {
	return rs.GetWithExpires(entryURI, attName, -1)
}

func (rs *Service) GetWithExpires(entryURI string, attName string, expires int) (data GetRet, code int, err error) {

	url := RS_HOST + "/get/" + EncodeURI(entryURI)
	if attName != "" {
		url = url + "/attName/" + EncodeURI(attName)
	}
	if expires > 0 {
		url = url + "/expires/" + strconv.Itoa(expires)
	}

	code, err = rs.Conn.Call(&data, url)
	if code == 200 {
		data.Expiry += seconds()
	}
	return
}

func (rs *Service) GetIfNotModified(entryURI string, attName string, base string) (data GetRet, code int, err error) {

	url := RS_HOST + "/get/" + EncodeURI(entryURI) + "/base/" + base
	if attName != "" {
		url = url + "/attName/" + EncodeURI(attName)
	}

	code, err = rs.Conn.Call(&data, url)
	if code == 200 {
		data.Expiry += seconds()
	}
	return
}

func (rs *Service) Stat(entryURI string) (entry Entry, code int, err error) {
	code, err = rs.Conn.Call(&entry, RS_HOST+"/stat/"+EncodeURI(entryURI))
	return
}

func (rs *Service) Delete(entryURI string) (code int, err error) {
	return rs.Conn.Call(nil, RS_HOST+"/delete/"+EncodeURI(entryURI))
}

func (rs *Service) Drop(entryURI string) (code int, err error) {
	return rs.Conn.Call(nil, RS_HOST+"/drop/"+entryURI)
}

func (rs *Service) Move(entryURISrc, entryURIDest string) (code int, err error) {
	return rs.Conn.Call(nil, RS_HOST+"/move/"+EncodeURI(entryURISrc)+"/"+EncodeURI(entryURIDest))
}

func (rs *Service) Copy(entryURISrc, entryURIDest string) (code int, err error) {
	return rs.Conn.Call(nil, RS_HOST+"/copy/"+EncodeURI(entryURISrc)+"/"+EncodeURI(entryURIDest))
}

func (rs *Service) Publish(domain, table string) (code int, err error) {
	return rs.Conn.Call(nil, RS_HOST+"/publish/"+EncodeURI(domain)+"/from/"+table)
}

func (rs *Service) Unpublish(domain string) (code int, err error) {
	return rs.Conn.Call(nil, RS_HOST+"/unpublish/"+EncodeURI(domain))
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

func (b *Batcher) Do(rs *Service) (ret []BatchRet, code int, err error) {
	code, err = rs.Conn.CallWithForm(&b.ret, RS_HOST+"/batch", map[string][]string{"op": b.op})
	ret = b.ret
	return
}

// ----------------------------------------------------------

func seconds() int64 {
	return time.Now().Unix()
}

// ----------------------------------------------------------

