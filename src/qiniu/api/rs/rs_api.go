package rs

import (
	"io"
	"net/http"
	"encoding/base64"
	"qiniu/digest_auth"
	"qiniu/rpc"
	. "qiniu/api/conf"
)

// ----------------------------------------------------------

type Service struct {
	Conn rpc.Client
}

func New() Service {
	t := digest_auth.NewTransport(ACCESS_KEY, SECRET_KEY, nil)
	client := &http.Client{Transport: t}
	return Service{rpc.Client{client}}
}

func NewEx(t http.RoundTripper) Service {
	client := &http.Client{Transport: t}
	return Service{rpc.Client{client}}
}

// ----------------------------------------------------------

type Entry struct {
	Hash     string `json:"hash"`
	Fsize    int64  `json:"fsize"`
	PutTime  int64  `json:"putTime"`
	MimeType string `json:"mimeType"`
	Customer string `json:"customer"`
}

type PutRet struct {
	Hash string `json:"hash"`
}

func (rs Service) Put(l rpc.Logger, entryURI, mimeType string, f io.Reader, fsize int64) (ret PutRet, err error) {

	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	url := IO_HOST + "/rs-put/" + EncodeURI(entryURI) + "/mimeType/" + EncodeURI(mimeType)
	err = rs.Conn.CallWith64(l, &ret, url, mimeType, f, fsize)
	return
}


func (rs Service) Stat(l rpc.Logger, entryURI string) (entry Entry, err error) {
	err = rs.Conn.Call(l, &entry, RS_HOST+"/stat/"+EncodeURI(entryURI))
	return
}

// ----------------------------------------------------------

func (rs Service) Delete(l rpc.Logger, entryURI string) (err error) {
	return rs.Conn.Call(l, nil, RS_HOST+"/delete/"+EncodeURI(entryURI))
}

func (rs Service) Move(l rpc.Logger, entryURISrc, entryURIDest string) (err error) {
	return rs.Conn.Call(l, nil, RS_HOST+"/move/"+EncodeURI(entryURISrc)+"/"+EncodeURI(entryURIDest))
}

func (rs Service) Copy(l rpc.Logger, entryURISrc, entryURIDest string) (err error) {
	return rs.Conn.Call(l, nil, RS_HOST+"/copy/"+EncodeURI(entryURISrc)+"/"+EncodeURI(entryURIDest))
}

// ----------------------------------------------------------

func (rs Service) Mkbucket(l rpc.Logger, bucketName string) (err error) {
	return rs.Conn.Call(l, nil, RS_HOST+"/mkbucket/"+bucketName)
}

func (rs Service) Drop(l rpc.Logger, bucketName string) (err error) {
	return rs.Conn.Call(l, nil, RS_HOST+"/drop/"+bucketName)
}

func (rs Service) Buckets(l rpc.Logger) (buckets []string, err error) {
	err = rs.Conn.Call(l, &buckets, RS_HOST+"/buckets")
	return
}

// ----------------------------------------------------------

func EncodeURI(uri string) string {
	return base64.URLEncoding.EncodeToString([]byte(uri))
}

// ----------------------------------------------------------

