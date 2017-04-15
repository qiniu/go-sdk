package fs

import (
	"strconv"
	"net/http"
	"encoding/base64"
	"github.com/qiniu/go-sdk/api"
	"github.com/qiniu/go-sdk/rpc"
	. "github.com/qiniu/go-sdk/api/conf"
)

// ----------------------------------------------------------

const (
	OutOfSpace        = 607 // FS: 空间满（User over quota）
	FileModified      = 608 // FS: 文件被修改（see fs.GetIfNotModified）
	Conflicted        = 609 // FS: 冲突
	NotAFile          = 610 // FS: 指定的 Entry 不是一个文件
	NotADirectory     = 611 // FS: 指定的 Entry 不是一个目录
	NoSuchEntry       = 612 // FS: 指定的 Entry 不存在或已经 Deleted
	NotADeletedEntry  = 613 // FS: 指定的 Entry 不是一个已经删除的条目
	EntryExists       = 614 // FS: 要创建的 Entry 已经存在
	CircularAction    = 615 // FS: 操作发生循环，无法完成
	NoSuchDirectory   = 616 // FS: Move 操作的 Parent Directory 不存在
	Locked            = 617 // FS: 要操作的 Entry 被锁，操作暂时无法进行
	DirectoryNotEmpty = 618 // FS: rmdir - directory not empty
	BadData           = 619 // FS: 数据已被破坏
	ConditionNotMeet  = 620 // FS: 条件不满足
)

var (
	EOutOfSpace        = api.Errno(OutOfSpace)
	EFileModified      = api.Errno(FileModified)
	EConflicted        = api.Errno(Conflicted)
	ENotAFile          = api.Errno(NotAFile)
	ENotADirectory     = api.Errno(NotADirectory)
	ENoSuchEntry       = api.Errno(NoSuchEntry)
	ENotADeletedEntry  = api.Errno(NotADeletedEntry)
	EEntryExists       = api.Errno(EntryExists)
	ECircularAction    = api.Errno(CircularAction)
	ENoSuchDirectory   = api.Errno(NoSuchDirectory)
	ELocked            = api.Errno(Locked)
	EDirectoryNotEmpty = api.Errno(DirectoryNotEmpty)
	EBadData           = api.Errno(BadData)
	EConditionNotMeet  = api.Errno(ConditionNotMeet)
)

func init() {
	api.RegisterErrno([]api.ErrnoMsg{
		{OutOfSpace, "out of space"},
		{FileModified, "file modified"},
		{Conflicted, "conflicted"},
		{NotAFile, "not a file"},
		{NotADirectory, "not a directory"},
		{NoSuchEntry, "no such file or directory"},
		{NotADeletedEntry, "not a deleted entry"},
		{EntryExists, "file exists"},
		{CircularAction, "circular action"},
		{NoSuchDirectory, "no such directory"},
		{Locked, "locked"},
		{DirectoryNotEmpty, "directory not empty"},
		{BadData, "bad data"},
		{ConditionNotMeet, "contition not meet"},
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

func New(t http.RoundTripper) Service {
	return Service{rpc.Client{&http.Client{Transport: t}}}
}

// ----------------------------------------------------------

const (
	File          = 0x0001
	Dir           = 0x0002
)

const (
	ShowDefault          = 0x0000
	ShowNormal           = 0x0001
	ShowDirOnly          = 0x0002
)

type Info struct {
	Used  int64 `json:"used"`
	Quota int64 `json:"quota"`
}

type MakeRet struct {
	Id string `json:"id"`
}

type PutRet struct {
	Id   string `json:"id"`
	Hash string `json:"hash"`
	Alt  string `json:"alt"`
}

type GetRet struct {
	Id  string `json:"id"`
	URL string `json:"url"`

	Hash     string `json:"hash"`
	Fsize    int64  `json:"fsize"`
	EditTime int64  `json:"editTime"`
	MimeType string `json:"mimeType"`
	Perm     uint32 `json:"perm"`
}

type Entry struct {
	Id      string `json:"id"`
	URI     string `json:"uri"`
	Type    int32  `json:"type"`
	Deleted int32  `json:"deleted"`

	Hash     string `json:"hash"`
	Fsize    int64  `json:"fsize"`
	EditTime int64  `json:"editTime"`
	MimeType string `json:"mimeType"`
	FPub     int    `json:"fpub"`
	Perm     uint32 `json:"perm"`
}

func (fs Service) Info() (info Info, code int, err error) {
	code, err = fs.Conn.Call(&info, FS_HOST+"/info")
	return
}

func (fs Service) Get(entryURI string) (data GetRet, code int, err error) {
	code, err = fs.Conn.Call(&data, FS_HOST+"/get/"+EncodeURI(entryURI))
	return
}

func (fs Service) GetIfNotModified(entryURI string, base string) (data GetRet, code int, err error) {
	code, err = fs.Conn.Call(&data, FS_HOST+"/get/"+EncodeURI(entryURI)+"/base/"+base)
	return
}

func (fs Service) Stat(entryURI string) (entry Entry, code int, err error) {
	code, err = fs.Conn.Call(&entry, FS_HOST+"/stat/"+EncodeURI(entryURI))
	return
}

func (fs Service) List(entryURI string) (entries []Entry, code int, err error) {
	code, err = fs.Conn.Call(&entries, FS_HOST+"/list/"+EncodeURI(entryURI))
	return
}

func (fs Service) ListWith(entryURI string, showType int) (entries []Entry, code int, err error) {
	code, err = fs.Conn.Call(&entries, FS_HOST+"/list/"+EncodeURI(entryURI)+"/showType/"+strconv.Itoa(showType))
	return
}

func (fs Service) mksth(entryURI string, method string) (id string, code int, err error) {
	var ret MakeRet
	code, err = fs.Conn.Call(&ret, FS_HOST+method+EncodeURI(entryURI))
	if code == 200 && ret.Id == "" {
		return "", api.UnexceptedResponse, api.EUnexceptedResponse
	}
	id = ret.Id
	return
}

func (fs Service) Mkdir(entryURI string) (id string, code int, err error) {
	return fs.mksth(entryURI, "/mkdir/")
}

func (fs Service) MkdirAll(entryURI string) (id string, code int, err error) {
	return fs.mksth(entryURI, "/mkdir_p/")
}

func (fs Service) Delete(entryURI string) (code int, err error) {
	return fs.Conn.Call(nil, FS_HOST+"/delete/"+EncodeURI(entryURI))
}

func (fs Service) Undelete(entryURI string) (code int, err error) {
	return fs.Conn.Call(nil, FS_HOST+"/undelete/"+EncodeURI(entryURI))
}

func (fs Service) Purge(entryURI string) (code int, err error) {
	return fs.Conn.Call(nil, FS_HOST+"/purge/"+EncodeURI(entryURI))
}

func (fs Service) Move(entryURISrc, entryURIDest string) (code int, err error) {
	return fs.Conn.Call(nil, FS_HOST+"/move/"+EncodeURI(entryURISrc)+"/"+EncodeURI(entryURIDest))
}

func (fs Service) Copy(entryURISrc, entryURIDest string) (code int, err error) {
	return fs.Conn.Call(nil, FS_HOST+"/copy/"+EncodeURI(entryURISrc)+"/"+EncodeURI(entryURIDest))
}

// ----------------------------------------------------------

type BatchRet struct {
	Data interface{} `json:"data"`
	Code int         `json:"code"`
}

type Batcher struct {
	op  []string
	ret []BatchRet
}

func (b *Batcher) mksth(entryURI string, method string) {
	var ret MakeRet
	b.op = append(b.op, method+EncodeURI(entryURI))
	b.ret = append(b.ret, BatchRet{Data: &ret})
}

func (b *Batcher) Mklink(entryURI string) {
	b.mksth(entryURI, "/mklink/")
}

func (b *Batcher) Mkdir(entryURI string) {
	b.mksth(entryURI, "/mkdir/")
}

func (b *Batcher) MkdirAll(entryURI string) {
	b.mksth(entryURI, "/mkdir_p/")
}

func (b *Batcher) operate(method string, entryURI string) {
	b.op = append(b.op, method+EncodeURI(entryURI))
	b.ret = append(b.ret, BatchRet{})
}

func (b *Batcher) operate2(method string, entryURISrc, entryURIDest string) {
	b.op = append(b.op, method+EncodeURI(entryURISrc)+"/"+EncodeURI(entryURIDest))
	b.ret = append(b.ret, BatchRet{})
}

func (b *Batcher) Delete(entryURI string) {
	b.operate("/delete/", entryURI)
}

func (b *Batcher) Undelete(entryURI string) {
	b.operate("/undelete/", entryURI)
}

func (b *Batcher) Purge(entryURI string) {
	b.operate("/purge/", entryURI)
}

func (b *Batcher) Move(entryURISrc, entryURIDest string) {
	b.operate2("/move/", entryURISrc, entryURIDest)
}

func (b *Batcher) Copy(entryURISrc, entryURIDest string) {
	b.operate2("/copy/", entryURISrc, entryURIDest)
}

func (b *Batcher) Reset() {
	b.op = nil
	b.ret = nil
}

func (b *Batcher) Len() int {
	return len(b.op)
}

func (b *Batcher) Do(fs Service) (ret []BatchRet, code int, err error) {
	code, err = fs.Conn.CallWithForm(&b.ret, FS_HOST+"/batch", map[string][]string{"op": b.op})
	ret = b.ret
	return
}

// ----------------------------------------------------------
