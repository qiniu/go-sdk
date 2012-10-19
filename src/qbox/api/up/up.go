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
	Ctx string  `json:"ctx"`
	Checksum string `json:"checksum"`
	Crc32 uint32 `json:"crc32"`
	Offset int64 `json:"offset"`
}


func SaveProgress(p interface{}, filename string) (err error) {
	b, err := json.Marshal(p)
	if err != nil {
		return
	}
	if err = ioutil.WriteFile(filename, b, 0644); err != nil {
		os.Truncate(filename,0)
	}
	return
}

func LoadProgress(p interface{}, filename string) (err error) {
	if fi, err1 := os.Stat(filename); err1 != nil || fi.Size() == 0 {
		return err1
	}
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	if err = json.Unmarshal(b,p); err != nil {
		os.Truncate(filename,0)
	}
	return
}



type RPtask struct {
	*Service
	EntryURI string
	Type string
	Size int64
	Customer, Meta string
	CallbackParams string
	Body io.ReaderAt
	Progress []BlockputProgress
	ChunkNotify, BlockNotify func(blockIdx int, prog *BlockputProgress)
}

func (s *Service) NewTask(
	entryURI, mimeType string, customer, meta, params string,
	r io.ReaderAt, size int64) (t *RPtask) {

	blockcnt := int((size + (1 << s.BlockBits) - 1) >> s.BlockBits)
	p := make([]BlockputProgress, blockcnt)
	return &RPtask{s,entryURI,mimeType,size,customer,meta,params,r,p,nil,nil}
}



func (t *RPtask) Run(taskQsize, threadSize int, progfile string,
	chunkNotify, blockNotify func(blockIdx int, prog *BlockputProgress)) (code int, err error) {

	var (
		wg sync.WaitGroup
		failed bool
	)
	worker := func(tasks chan func()) {
		for {
			task := <- tasks
			task()
		}
	}
	blockcnt := len(t.Progress)
	t.ChunkNotify = chunkNotify
	t.BlockNotify = blockNotify

	// Load progress cache file, if it exists
	if progfile != "" {
		if err = LoadProgress(&t.Progress,progfile); err != nil {
			return errcode.InternalError, err
		}
	}

	if taskQsize == 0 {
		taskQsize = blockcnt
	}
	if threadSize == 0 {
		threadSize = blockcnt
	}

	tasks := make(chan func(), taskQsize)
	for i := 0; i < threadSize; i++ {
		go worker(tasks)
	}

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
		tasks <- task
	}
	wg.Wait()

	if failed {
		if err = SaveProgress(&t.Progress,progfile); err == nil {
			err = errors.New("ResumableBlockput haven't done")
		}
		return 400, err
	}
	return t.Mkfile()
}


func (t *RPtask) PutBlock(blockIdx int) (code int, err error) {
	var (
		url string
		restsize, blocksize int64
	)

	h := crc32.NewIEEE()
	prog := &t.Progress[blockIdx]
	offbase := int64(blockIdx << t.BlockBits)

	// blocksize
	if blockIdx == len(t.Progress) - 1 {
		blocksize = t.Size - offbase
	} else {
		blocksize = int64(1 << t.BlockBits)
	}

	initProg := func(p *BlockputProgress) {
		p.Offset = 0
		p.Ctx = ""
		p.Crc32 = 0
		p.Checksum = ""
		restsize = blocksize
	}

	if prog.Ctx == "" {
		initProg(prog)
	}

	for restsize > 0 {
		bdlen := t.RPutChunkSize
		if bdlen > restsize {
			bdlen = restsize
		}
		retry := t.RPutRetryTimes
	lzRetry:
		h.Reset()
		bd1 := io.NewSectionReader(t.Body, int64(offbase + prog.Offset), int64(bdlen))
		bd := io.TeeReader(bd1, h)
		if prog.Ctx == "" {
			url = t.Host["up"] + "/mkblk/" + strconv.FormatInt(blocksize, 10)
		} else {
			url = t.Host["up"] + "/bput/" + prog.Ctx + "/" + strconv.FormatInt(prog.Offset, 10)
		}
		code, err = t.Conn.CallWith(prog, url, "application/octet-stream", bd, int(bdlen))
		if err == nil {
			if prog.Crc32 == h.Sum32() {
				restsize = blocksize - prog.Offset
				if t.ChunkNotify != nil {
					t.ChunkNotify(blockIdx,prog)
				}
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
	if t.BlockNotify != nil {
		t.BlockNotify(blockIdx,prog)
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
	url := t.Host["up"] + "/rs-mkfile/" + EncodeURI(t.EntryURI)
	url += "/fsize/" + strconv.FormatInt(t.Size, 10)
	if t.Meta != "" {
		url += "/meta/" + EncodeURI(t.Meta)
	}
	if t.Customer != "" {
		url += "/customer/" + t.Customer
	}
	if t.CallbackParams != "" {
		url += "/params/" + t.CallbackParams
	}
	code, err = t.Conn.CallWith(nil, url, "", strings.NewReader(string(bd)), len(bd))
	return
}

func (s *Service) ResumablePut(
	entryURI, mimeType string, bd io.ReaderAt, bdlen int64, customer, meta, params string, 
	chunkNotify, blockNotify func(blockIdx int, prog *BlockputProgress)) (code int, err error) {

	var (
		wg sync.WaitGroup
		failed bool
	)
	blockcnt := int((bdlen + (1 << s.BlockBits) - 1) >> s.BlockBits)
	p := make([]BlockputProgress, blockcnt)

	// If progress cache file exists, then load progress
	pgfile := s.DataPath + string(os.PathSeparator) + entryURI + strconv.FormatInt(bdlen, 10)
	LoadProgress(&p,pgfile)

	t := &RPtask{s,entryURI,mimeType,bdlen,customer,meta,params,bd,p,chunkNotify,blockNotify}


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
		SaveProgress(&p,pgfile)
		return 400, errors.New("ResumableBlockPut haven't done")
	}
	code, err = t.Mkfile()
	if code/100 == 2 {
		os.Remove(pgfile)
	}
	return
}
