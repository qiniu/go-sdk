package api

import (
	"io"
	"strconv"
	"errors"
	"sync"
	"strings"
	"hash/crc32"
)


const (
	InvalidCtx = 701 // 无效的上下文
)


// Resumable put task

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
		if code == InvalidCtx {
			initProg(prog)
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



func (s *Service) ResumablePut(entryURI, mimeType string, bd io.ReaderAt, bdlen int64) (code int, err error) {
	var (
		wg sync.WaitGroup
		failed bool
	)
	blockcnt := int((bdlen + (1 << s.BlockBits) - 1) >> s.BlockBits)
	p := make([]*BlockputProgress, blockcnt)
	t := &RPtask{s,entryURI,mimeType,bdlen,bd,p}

	wg.Add(blockcnt)
	for i := 0; i < blockcnt; i++ {
		t.Progress[i] = &BlockputProgress{}
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
		return 400, errors.New("ResumableBlockPut haven't done")
	}
	return t.Mkfile()
}