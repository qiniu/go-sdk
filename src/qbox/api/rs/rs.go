package rs

import (
	"io"
	"io/ioutil"
	"errors"
	"strconv"
	"strings"
	"time"
	"sync"
	"hash/crc32"
	"encoding/json"
	"net/http"
	. "qbox/api"
	"qbox/rpc"
	"qbox/errcode"
	"qbox/auth/digest"
)



type Service struct {
	*Config
	Conn rpc.Client
}


func New(c *Config, args... interface{}) (s *Service, err error) {
	var (
		t http.RoundTripper
	)
	if c == nil {
		err = errors.New("Must have a config file")
		return
	}
	for _,v := range args {
		switch v.(type) {
		case http.RoundTripper:
			t = v.(http.RoundTripper)
			break
		}
	}
	t = digest.NewTransport(c.Access_key, c.Secret_key, t)
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



type BlockputProgress struct {
	Ctx string
	Offset int64
	RestSize int64
	Checksum string
}


type RPtask struct {
	*Service
	EntryURI string
	Type string
	Size int64
	Body io.ReaderAt
	Progress []*BlockputProgress
}

type RPutRet struct {
	Ctx string  `json:"ctx"`
	Checksum string `json:"checksum"`
	Crc32 uint32 `json:"crc32"`
	Offset uint32 `json:"offset"`
}

func (t *RPtask) PutBlock(blockIdx int) (code int, err error) {
	var (
		ret RPutRet
		url string
	)
	h := crc32.NewIEEE()
	prog := t.Progress[blockIdx]
	offbase := int64(blockIdx << t.BlockBits)

	initProg := func(p *BlockputProgress) {
		if blockIdx == len(t.Progress) - 1 {
			p.RestSize = t.Size - offbase
		} else {
			p.RestSize = 1 << t.BlockBits
		}
		p.Offset = 0
		p.Ctx = ""
		p.Checksum = ""
	}

	if prog.Ctx == "" {
		initProg(prog)
	}

	for prog.RestSize > 0 {
		bdlen := t.RPutChunkSize
		if bdlen > prog.RestSize {
			bdlen = prog.RestSize
		}
		retry := t.RPutRetryTimes
	lzRetry:
		h.Reset()
		bd1 := io.NewSectionReader(t.Body, int64(offbase + prog.Offset), int64(bdlen))
		bd := io.TeeReader(bd1, h)
		if prog.Ctx == "" {
			url = t.Host["up"] + "/mkblk/" + strconv.FormatInt(prog.RestSize, 10)
		} else {
			url = t.Host["up"] + "/bput/" + prog.Ctx + "/" + strconv.FormatInt(prog.Offset, 10)
		}
		code, err = t.Conn.CallWith(&ret, url, "application/octet-stream", bd, int(bdlen))
		if err == nil {
			if ret.Crc32 == h.Sum32() {
				prog.Ctx = ret.Ctx
				prog.Offset += bdlen
				prog.RestSize -= bdlen
				continue
			} else {
				err = errors.New("ResumableBlockPut: Invalid Checksum")
			}
		}
		if code == errcode.InvalidCtx {
			initProg(prog)
			err = errcode.EInvalidCtx
			continue   // retry upload current block
		}
		if retry > 0 {
			retry--
			goto lzRetry
		}
		break
	}
	return
}

func (t *RPtask) Mkfile() (code int, err error) {
	var (
		ctx string
	)
	for k,p := range t.Progress {
		if k == len(t.Progress) - 1 {
			ctx += p.Ctx
		} else {
			ctx += p.Ctx + ","
		}
	}
	bd := []byte(ctx)
	url := t.Host["up"] + "/rs-mkfile/" + EncodeURI(t.EntryURI) + "/fsize/" + strconv.FormatInt(t.Size, 10)
	code, err = t.Conn.CallWith(nil, url, "", strings.NewReader(string(bd)), len(bd))
	return
}


func (s *Service) SaveProg(p interface{}, filename string) (err error) {
	b, err := json.Marshal(p)
	if err != nil {
		return
	}
	fn := s.DataPath + "/" + filename
	return ioutil.WriteFile(fn, b, 0644)
}

func (s *Service) LoadProg(p interface{}, filename string) (err error) {
	fn := s.DataPath + "/" + filename
	b, err := ioutil.ReadFile(fn)
	if err != nil {
		return
	}
	return json.Unmarshal(b,p)
}

func (s *Service) ResumablePut(entryURI, mimeType string, bd io.ReaderAt, bdlen int64) (code int, err error) {
	var (
		wg sync.WaitGroup
		failed bool
	)
	blockcnt := int((bdlen + (1 << s.BlockBits) - 1) >> s.BlockBits)
	p := make([]*BlockputProgress, blockcnt)
	for k,_ := range p {
		p[k] = &BlockputProgress{}
	}
	pgfile := entryURI + strconv.FormatInt(bdlen, 10)
	s.LoadProg(p, pgfile)
	t := &RPtask{s,entryURI,mimeType,bdlen,bd,p}

	wg.Add(blockcnt)
	for i := 0; i < blockcnt; i++ {
		blkIdx := i
		task := func() {
			defer wg.Done()
			code, err = t.PutBlock(blkIdx)
			if err != nil {
				failed = true
			}
		}
		go task()
	}
	wg.Wait()

	if failed {
		s.SaveProg(p, pgfile)
		return 400, errors.New("ResumableBlockPut haven't done")
	}
	return t.Mkfile()
}

