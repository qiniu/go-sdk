package api

import (
	"io"
	"time"
	"strconv"
	"../utils/errcode"
)


const (
	FileModified = 608 // RS: 文件被修改（see fs.GetIfNotModified）
	NoSuchEntry  = 612 // RS: 指定的 Entry 不存在或已经 Deleted
	EntryExists  = 614 // RS: 要创建的 Entry 已经存在
)

var (
	EFileModified      = errcode.Errno(FileModified)
	ENoSuchEntry       = errcode.Errno(NoSuchEntry)
	EEntryExists       = errcode.Errno(EntryExists)
)


func init() {
	errcode.RegisterErrno([]errcode.ErrnoMsg{
		{FileModified, "file modified"},
		{NoSuchEntry, "no such file or directory"},
		{EntryExists, "file exists"},
	})
}


// ----------------------------------------------------------------

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

func (s *Service) Get(entryURI string, attName string) (data GetRet, code int, err error) {
	return s.GetWithExpires(entryURI, attName, -1)
}

func (s *Service) GetWithExpires(entryURI string, attName string, expires int) (data GetRet, code int, err error) {

	url := s.Host["rs"] + "/get/" + EncodeURI(entryURI)
	if attName != "" {
		url = url + "/attName/" + EncodeURI(attName)
	}
	if expires > 0 {
		url = url + "/expires/" + strconv.Itoa(expires)
	}

	code, err = s.Conn.Call(&data, url)
	if code == 200 {
		data.Expiry += seconds()
	}
	return
}

func (s *Service) GetIfNotModified(entryURI string, attName string, base string) (data GetRet, code int, err error) {

	url := s.Host["rs"] + "/get/" + EncodeURI(entryURI) + "/base/" + base
	if attName != "" {
		url = url + "/attName/" + EncodeURI(attName)
	}

	code, err = s.Conn.Call(&data, url)
	if code == 200 {
		data.Expiry += seconds()
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

// ----------------------------------------------------------

func seconds() int64 {
	return time.Now().Unix()
}

// ----------------------------------------------------------