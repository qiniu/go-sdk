package up

import (
	"io"
	"os"
	"io/ioutil"
	"errors"
	"sync"
	"strings"
	"strconv"
	"net/http"
	"encoding/json"
	"hash/crc32"
	. "qbox/api"
	"qbox/rpc"
	"qbox/errcode"
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
	Progress []*RPutRet
}

type RPutRet struct {
	Ctx string  `json:"ctx"`
	Checksum string `json:"checksum"`
	Crc32 uint32 `json:"crc32"`
	Offset int64 `json:"offset"`
}

func (t *RPtask) PutBlock(blockIdx int) (code int, err error) {
	var (
		url string
		restsize, blocksize int64
	)

	h := crc32.NewIEEE()
	prog := t.Progress[blockIdx]
	offbase := int64(blockIdx << t.BlockBits)

	// default blocksize
	blocksize = int64(1 << t.BlockBits)

	initProg := func(p *RPutRet) {
		if blockIdx == len(t.Progress) - 1 {
			blocksize = t.Size - offbase
		}
		p.Offset = 0
		p.Ctx = ""
		p.Crc32 = ""
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
				prog.Offset = ret.Offset
				prog.RestSize = blocksize - ret.Offset
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

func (t *RPtask) Mkfile(customer, meta, callbackparams string) (code int, err error) {
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
	url := t.Host["up"] + "/rs-mkfile/" + EncodeURI(t.EntryURI)
	url += "/fsize/" + strconv.FormatInt(t.Size, 10)
	if meta != "" {
		url += "/meta/" + EncodeURI(meta)
	}
	if customer != "" {
		url += "/customer/" + customer
	}
	if callbackparams != "" {
		url += "/params/" + callbackparams
	}
	code, err = t.Conn.CallWith(nil, url, "", strings.NewReader(string(bd)), len(bd))
	return
}

func (s *Service) SaveProg(p interface{}, filename string) (err error) {
	b, err := json.Marshal(p)
	if err != nil {
		return
	}
	fn := s.DataPath + "/" + filename
	if err = ioutil.WriteFile(fn, b, 0644); err != nil {
		os.Remove(fn)
	}
	return
}

func (s *Service) LoadProg(p interface{}, filename string) (err error) {
	fn := s.DataPath + "/" + filename
	b, err := ioutil.ReadFile(fn)
	if err != nil {
		return
	}
	if err = json.Unmarshal(b,p); err != nil {
		os.Remove(fn)
	}
	return
}

func (s *Service) ResumablePut(entryURI, mimeType string, bd io.ReaderAt, bdlen int64, customer, meta, params string) (code int, err error) {
	var (
		wg sync.WaitGroup
		failed bool
	)
	blockcnt := int((bdlen + (1 << s.BlockBits) - 1) >> s.BlockBits)
	p := make([]*BlockputProgress, blockcnt)
	for k,_ := range p {
		p[k] = &BlockputProgress{}
	}

	// If progress cache file exists, then load progress
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
		// save resumable put progress
		s.SaveProg(p, pgfile)
		return 400, errors.New("ResumableBlockPut haven't done")
	}
	return t.Mkfile(customer, meta, params)
}

/*
// use Uptoken upload
func Upload(token, entryURI, mimeType string, bd io.Reader, bdlen int64) (ret UpRet, code int, err error) {

	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	
	url := s.Host["up"] + "/upload/" + api.EncodeURI(entryURI) + "/mimeType/" + api.EncodeURI(mimeType)
	code, err = s.Conn.CallWith64(&ret, url, "application/octet-stream", body, bodyLength)
	return
}*/