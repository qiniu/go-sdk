package qnbox

import (
	"io"
	"time"
	"sync"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"strconv"
	"qnbox/rpc"
	"qnbox/errcode"
	"qnbox/auth/digest"
	"qnbox/auth/uptoken"
	"hash/crc32"
	"encoding/json"
	"encoding/base64"
)


type Config struct {
	Host map[string]string `json:"HOST"`

	Access_key string `json:"QBOX_ACCESS_KEY"`
	Secret_key string `json:"QBOX_SECRET_KEY"`
	BlockBits uint `json:"BLOCK_BITS"`
	RPutChunkSize int64 `json:"RPUT_CHUNK_SIZE"`
	RPutRetryTimes int `json:"RPUT_RETRY_TIMES"`

	Client string `json:"CLIENT"`
	ClientId string `json:"CLIENT_ID"`
	ClientSecret string `json:"CLIENT_SECRET"`

	RedirectURI string `json:"REDIRECT_URI"`
	AuthorizationEndPoint string `json:"AUTHORIZATION_ENDPOINT"`
	TokenEndPoint string `json:"TOKEN_ENDPOINT"`
}

type Service struct {
	Config
	Conn rpc.Client
}


func loadConfig(filename string) (c *Config) {
	var conf Config

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	err = json.Unmarshal(b, &conf)
	if err != nil {
		return
	}
	c = &conf
	return
}


func New(c Config, t http.RoundTripper) *Service {
	if c == nil {
		c = &Config{}
	}
	if t == nil {
		t = digest.NewTransport(c.Access_key, c.Secret_key, nil)
	}
	client := &http.Client{Transport: t}
	return &Service{c, rpc.Client{client}}
}

func EncodeURI(uri string) string {
	return base64.URLEncoding.EncodeToString([]byte(uri))
}


// ----------------------------------------------------------
// EU server

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

func (s *Service) GetWatermark(customer string) (ret Watermark, code int, err error) {

	params := map[string][]string{
		"customer": {customer},
	}
	code, err = s.Conn.CallWithForm(&ret, s.Host["eu"] + "/wmget", params)
	return
}

func (s *Service) SetWatermark(customer string, args *Watermark) (code int, err error) {

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
	return s.Conn.CallWithForm(nil, s.Host["eu"] + "/wmset", params)
}




// ---------------------------------------------------------------
// Public server

type BucketInfo struct {
	Source string	`json:"source" bson:"source"`
	Host string `json:"host" bson:"host"`
	Expires int		`json:"expires" bson:"expires"`
	Protected int `json:"protected" bson:"protected"`
	Separator string `json:"separator" bson:"separator"`
	Styles map[string]string `json:"styles" bson:"styles"`
}

func (s *Service) Image(bucketName string, srcSiteUrls []string, srcHost string, expires int) (code int, err error) {
	url := s.Host["pu"] + "/image/" + bucketName
	for _, srcSiteUrl := range srcSiteUrls {
		url += "/from/" + EncodeURI(srcSiteUrl)
	}
	if expires != 0 {
		url += "/expires/" + strconv.Itoa(expires)
	}
	if srcHost != "" {
		url += "/host/" + EncodeURI(srcHost)
	}
	return s.Conn.Call(nil, url)
}

func (s *Service) Unimage(bucketName string) (code int, err error) {
	return s.Conn.Call(nil, s.Host["pu"] + "/unimage/" + bucketName)
}

func (s *Service) Info(bucketName string) (info BucketInfo, code int, err error) {
	code, err = s.Conn.Call(&info, s.Host["pu"] + "/info/" + bucketName)
	return
}

func (s *Service) AccessMode(bucketName string, mode int) (code int, err error) {
	return s.Conn.Call(nil, s.Host["pu"] + "/accessMode/" + bucketName + "/mode/" + strconv.Itoa(mode))
}

func (s *Service) Separator(bucketName string, sep string) (code int, err error) {
	return s.Conn.Call(nil, s.Host["pu"] + "/separator/" + bucketName + "/sep/" + EncodeURI(sep))
}

func (s *Service) Style(bucketName string, name string, style string) (code int, err error) {
	return s.Conn.Call(nil, s.Host["pu"] + "/style/" + bucketName+"/name/" + EncodeURI(name) + "/style/" + EncodeURI(style))
}

func (s *Service) Unstyle(bucketName string, name string) (code int, err error) {
	return s.Conn.Call(nil, s.Host["pu"] + "/unstyle/" + bucketName + "/name/" + EncodeURI(name))
}

// ----------------------------------------------------------
// RS server


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


// ------------------------------------------------------------------------------------------
// UC server

func (s *Service) AntiLeechMode(bucket string, mode int) (code int, err error) {
	param := map[string][]string {
		"bucket": {bucket},
		"mode": {strconv.Itoa(mode)},
	}
	url := s.Host["uc"] + "/antiLeechMode"
	return s.Conn.CallWithForm(nil, url, param)
}

func (s *Service) AddAntiLeech(bucket string, mode int, pattern string) (code int, err error) {
	param := map[string][]string {
		"bucket": {bucket},
		"mode": {strconv.Itoa(mode)},
		"action": {"add"},
		"pattern": {pattern},
	}
	url := s.Host["uc"] + "/referAntiLeech"
	return s.Conn.CallWithForm(nil, url, param)
}

func (s *Service) CleanCache(bucket string) (code int, err error) {
	param := map[string][]string {
		"bucket": {bucket},
	}
	url := s.Host["uc"] + "/refreshBucket"
	return s.Conn.CallWithForm(nil, url, param)
}

func (s *Service) DelAntiLeech(bucket string, mode int, pattern string) (code int, err error) {
	param := map[string][]string {
		"bucket": {bucket},
		"mode": {strconv.Itoa(mode)},
		"action": {"del"},
		"pattern": {pattern},
	}
	url := s.Host["uc"] + "/referAntiLeech"
	return s.Conn.CallWithForm(nil, url, param)
}

/*

func (s *Service) SetImagePreviewStyle(name string, style string) (code int, err error) {

	params := map[string][]string{
		"name": {name},
	}
	ps := strings.Split(style, ";")
	ps0 := ps[0]
	if strings.HasPrefix(ps0, "square:") {
		params["mode"] = []string{"square"}
		params["size"] = []string{ps0[7:]}
	} else {
		pos := strings.Index(ps0, "x")
		if pos == -1 {
			code, err = errcode.InvalidArgs, errcode.EInvalidArgs
			return
		}
		width := ps0[:pos]
		height := ps0[pos+1:]
		if width != "" {
			params["width"] = []string{width}
		}
		if height != "" {
			params["height"] = []string{height}
		}
	}
	for i := 1; i < len(ps); i++ {
		pos := strings.Index(ps[i], ":")
		if pos == -1 {
			code, err = errcode.InvalidArgs, errcode.EInvalidArgs
			return
		}
		params[ps[i][:pos]] = []string{ps[i][pos+1:]}
	}
	code, err = s.Conn.CallWithForm(nil, s.Host["uc"] + "/setImagePreviewStyle", params)
	return
}
*/

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


// use Uptoken upload
func Upload(token, entryURI, mimeType string, bd io.Reader, bdlen int64) (ret PutRet, code int, err error) {

	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	
	url := s.Host["up"] + "/upload/" + EncodeURI(entryURI) + "/mimeType/" + EncodeURI(mimeType)
	code, err = s.Conn.CallWith64(&ret, url, "application/octet-stream", body, bodyLength)
	return
}