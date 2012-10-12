package up

import (
	"io"
	"net/http"
	"strconv"
	"sync"
	"errors"
	"hash/crc32"
	"encoding/base64"
	"qbox/api"
	"qbox/utils/bytes"
	"qbox/utils/rpc"
	. "qbox/api/conf"
)

// ----------------------------------------------------------

const (
	InvalidCtx = 701 // UP: 无效的上下文(bput)，可能情况：Ctx非法或者已经被淘汰（太久未使用）
)



type Service struct {
	Tasks chan func()
	Conn rpc.Client
}


func New(taskQsize, threadSize int, t http.RoundTripper) *Service {
	tasks := make(chan func(), taskQsize)
	for i := 0; i < threadSize; i++ {
		go worker(tasks)
	}
	return &Service{tasks, rpc.Client{&http.Client{Transport: t}}}
}



func worker(tasks chan func()) {
	for {
		task := <-tasks
		task()
	}
}


// ----------------------------------------------------------

type PutRet struct {
	Ctx      string `json:"ctx"`
	Checksum string `json:"checksum"`
	Crc32    uint32 `json:"crc32"`
	Offset   uint32 `json:"offset"`
}

func (up *Service) Mkblock(blockSize int, body io.Reader, bodyLength int) (ret PutRet, code int, err error) {
	url := UP_HOST + "/mkblk/" + strconv.Itoa(blockSize)
	code, err = up.Conn.CallWith(&ret, url, "application/octet-stream", body, bodyLength)
	return
}

func (up *Service) Blockput(ctx string, offset int, body io.Reader, bodyLength int) (ret PutRet, code int, err error) {
	url := UP_HOST + "/bput/" + ctx + "/" +strconv.Itoa(offset)
	code, err = up.Conn.CallWith(&ret, url, "application/octet-stream", body, bodyLength)
	return
}


// ----------------------------------------------------------

type BlockputProgress struct {
	Ctx      string
	Offset   int
	RestSize int
	Err      error
}

func (up *Service) ResumableBlockput(
	f io.ReaderAt, blockIdx int, blkSize, chunkSize, retryTimes int,
	prog *BlockputProgress, notify func(blockIdx int, prog *BlockputProgress)) (ret PutRet, code int, err error) {

	offbase := int64(blockIdx) << BLOCK_BITS
	h := crc32.NewIEEE()

	var bodyLength int

	if prog.Ctx == "" {

		if chunkSize < blkSize {
			bodyLength = chunkSize
		} else {
			bodyLength = blkSize
		}

		body1 := io.NewSectionReader(f, offbase, int64(bodyLength))
		body := io.TeeReader(body1, h)

		ret, code, err = up.Mkblock(blkSize, body, bodyLength)
		if err != nil {
//			err = errors.New("ResumableBlockput: Mkblock failed")
			return
		}

		if ret.Crc32 != h.Sum32() {
			err = errors.New("ResumableBlockput: invalid checksum")
			return
		}

		prog.Ctx = ret.Ctx
		prog.Offset = bodyLength
		prog.RestSize = blkSize - bodyLength

		if notify != nil {
			notify(blockIdx, prog)
		}

	} else if prog.Offset+prog.RestSize != blkSize {

		code, err = 400, errors.New("ResumableBlockput: invalid blksize")
		return
	}

	for prog.RestSize > 0 {

		if chunkSize < prog.RestSize {
			bodyLength = chunkSize
		} else {
			bodyLength = prog.RestSize
		}

		retry := retryTimes

	lzRetry:

		body1 := io.NewSectionReader(f, offbase+int64(prog.Offset), int64(bodyLength))
		h.Reset()
		body := io.TeeReader(body1, h)

		ret, code, err = up.Blockput(prog.Ctx, prog.Offset, body, bodyLength)
		if err == nil {
			if ret.Crc32 == h.Sum32() {
				prog.Ctx = ret.Ctx
				prog.Offset += bodyLength
				prog.RestSize -= bodyLength
				notify(blockIdx, prog)
				continue
			} else {
				err = errors.New("ResumableBlockput: Invalid checksum")
			}
		} else {
//			err = errors.New("ResumableBlockput: Blockput failed")
			if code == InvalidCtx {
				prog.Ctx = ""
				notify(blockIdx, prog)
				break
			}
		}
		if retry > 0 {
			retry--
			goto lzRetry
		}
		break
	}
	return
}

// ----------------------------------------------------------
// cmd = "/rs-mkfile/" | "/fs-mkfile/"

func (up *Service) Mkfile(
	ret interface{}, cmd, entry string,
	fsize int64, params, callbackParams string, checksums []string) (code int, err error) {

	if callbackParams != "" {
		params += "/params/" + rpc.EncodeURI(callbackParams)
	}
	n := len(checksums)
	body := make([]byte, 20*n)
	for i, checksum := range checksums {
		ret, err2 := base64.URLEncoding.DecodeString(checksum)
		if err2 != nil {
			code, err = 400, errors.New("Makfile: decode checksums error")
			return
		}
		copy(body[i*20:], ret)
	}
	url := UP_HOST + cmd + rpc.EncodeURI(entry) + "/fsize/" +strconv.FormatInt(fsize, 10) + params
	code, err = up.Conn.CallWith(ret, url, "application/octet-stream", bytes.NewReader(body), len(body))
	return
}

// ----------------------------------------------------------



// ----------------------------------------------------------

func BlockCount(fsize int64) int {

	blockMask := int64((1 << BLOCK_BITS) - 1)
	return int((fsize + blockMask) >> BLOCK_BITS)
}

func (up *Service) Put(
	f io.ReaderAt, fsize int64, checksums []string, progs []BlockputProgress,
	blockNotify func(blockIdx int, checksum string),
	chunkNotify func(blockIdx int, prog *BlockputProgress)) (code int, err error) {

	blockCnt := BlockCount(fsize)
	if len(checksums) != blockCnt || len(progs) != blockCnt {
		code, err = 400, errors.New("Put: Invalid blockCnt")
		return
	}

	var wg sync.WaitGroup
	wg.Add(blockCnt)

	last := blockCnt - 1
	blockSize := 1 << BLOCK_BITS

	var failed bool
	for i := 0; i < blockCnt; i++ {
		if checksums[i] == "" {
			blockIdx := i
			blockSize1 := blockSize
			if i == last {
				offbase := int64(blockIdx) << BLOCK_BITS
				blockSize1 = int(fsize - offbase)
			}
			task := func() {
				defer wg.Done()
				ret, _, err2 := up.ResumableBlockput(
					f, blockIdx, blockSize1, PUT_CHUNK_SIZE, PUT_RETRY_TIMES, &progs[blockIdx], chunkNotify)
				if err2 != nil {
					failed = true
				} else {
					checksums[blockIdx] = ret.Checksum
					if blockNotify != nil {
						blockNotify(blockIdx, ret.Checksum)
					}
				}
				progs[blockIdx].Err = err2
			}
			up.Tasks <- task
		} else {
			wg.Done()
		}
	}

	wg.Wait()
	if failed {
		code, err = api.FunctionFail, errors.New("Put: ResumableBlockput haven't done")
	} else {
		code = 200
	}
	return
}
