package storage

import (
	"bytes"
	"context"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/qiniu/go-sdk/v7/client"
	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
	"github.com/qiniu/go-sdk/v7/storagev2/apis"
	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"
)

// PutExtra 为表单上传的额外可选项
type PutExtra struct {
	// 可选。
	// 用户自定义参数：key 以"x:"开头，而且 value 不能为空 eg: key为x:qqq
	// 自定义 meta：key 以"x-qn-meta-"开头，而且 value 不能为空 eg: key为x-qn-meta-aaa
	Params map[string]string

	UpHost string

	TryTimes int // 可选。尝试次数

	// 主备域名冻结时间（默认：600s），当一个域名请求失败（单个域名会被重试 TryTimes 次），会被冻结一段时间，使用备用域名进行重试，在冻结时间内，域名不能被使用，当一个操作中所有域名竣备冻结操作不在进行重试，返回最后一次操作的错误。
	HostFreezeDuration time.Duration

	// 可选，当为 "" 时候，服务端自动判断。
	MimeType string

	// 上传事件：进度通知。这个事件的回调函数应该尽可能快地结束。
	OnProgress func(fsize, uploaded int64)
}

func (extra *PutExtra) init() {
	if extra.TryTimes == 0 {
		extra.TryTimes = settings.TryTimes
	}
	if extra.HostFreezeDuration <= 0 {
		extra.HostFreezeDuration = 10 * 60 * time.Second
	}
}

// PutRet 为七牛标准的上传回复内容。
// 如果使用了上传回调或者自定义了returnBody，那么需要根据实际情况，自己自定义一个返回值结构体
type PutRet struct {
	Hash         string `json:"hash"`
	PersistentID string `json:"persistentId"`
	Key          string `json:"key"`
}

// FormUploader 表示一个表单上传的对象
type FormUploader struct {
	// Deprecated
	Client *client.Client
	// Deprecated
	Cfg     *Config
	storage *apis.Storage
}

// NewFormUploader 用来构建一个表单上传的对象
func NewFormUploader(cfg *Config) *FormUploader {
	return NewFormUploaderEx(cfg, nil)
}

// NewFormUploaderEx 用来构建一个表单上传的对象
func NewFormUploaderEx(cfg *Config, clt *client.Client) *FormUploader {
	if cfg == nil {
		cfg = NewConfig()
	}

	if clt == nil {
		clt = &client.DefaultClient
	}
	opts := http_client.Options{
		BasicHTTPClient:     clt.Client,
		UseInsecureProtocol: !cfg.UseHTTPS,
	}
	if region := cfg.GetRegion(); region != nil {
		opts.Regions = region
	}

	return &FormUploader{
		Client:  clt,
		Cfg:     cfg,
		storage: apis.NewStorage(&opts),
	}
}

// PutFile 用来以表单方式上传一个文件，和 Put 不同的只是一个通过提供文件路径来访问文件内容，一个通过 io.Reader 来访问。
//
// ctx       是请求的上下文。
// ret       是上传成功后返回的数据。如果 uptoken 中没有设置 callbackUrl 或 returnBody，那么返回的数据结构是 PutRet 结构。
// uptoken   是由业务服务器颁发的上传凭证。
// key       是要上传的文件访问路径。比如："foo/bar.jpg"。注意我们建议 key 不要以 '/' 开头。另外，key 为空字符串是合法的。
// localFile 是要上传的文件的本地路径。
// extra     是上传的一些可选项，可以指定为nil。详细见 PutExtra 结构的描述。
func (p *FormUploader) PutFile(
	ctx context.Context, ret interface{}, uptoken, key, localFile string, extra *PutExtra) (err error) {
	return p.putFile(ctx, ret, uptoken, key, true, localFile, extra)
}

// PutFileWithoutKey 用来以表单方式上传一个文件。不指定文件上传后保存的key的情况下，文件命名方式首先看看
// uptoken 中是否设置了 saveKey，如果设置了 saveKey，那么按 saveKey 要求的规则生成 key，否则自动以文件的 hash 做 key。
// 和 Put 不同的只是一个通过提供文件路径来访问文件内容，一个通过 io.Reader 来访问。
//
// ctx       是请求的上下文。
// ret       是上传成功后返回的数据。如果 uptoken 中没有设置 CallbackUrl 或 ReturnBody，那么返回的数据结构是 PutRet 结构。
// uptoken   是由业务服务器颁发的上传凭证。
// localFile 是要上传的文件的本地路径。
// extra     是上传的一些可选项。可以指定为nil。详细见 PutExtra 结构的描述。
func (p *FormUploader) PutFileWithoutKey(
	ctx context.Context, ret interface{}, uptoken, localFile string, extra *PutExtra) (err error) {
	return p.putFile(ctx, ret, uptoken, "", false, localFile, extra)
}

func (p *FormUploader) putFile(
	ctx context.Context, ret interface{}, upToken string,
	key string, hasKey bool, localFile string, extra *PutExtra) (err error) {

	f, err := os.Open(localFile)
	if err != nil {
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return
	}
	fsize := fi.Size()

	return p.put(ctx, ret, upToken, key, hasKey, f, fsize, extra, filepath.Base(localFile))
}

// Put 用来以表单方式上传一个文件。
//
// ctx     是请求的上下文。
// ret     是上传成功后返回的数据。如果 uptoken 中没有设置 callbackUrl 或 returnBody，那么返回的数据结构是 PutRet 结构。
// uptoken 是由业务服务器颁发的上传凭证。
// key     是要上传的文件访问路径。比如："foo/bar.jpg"。注意我们建议 key 不要以 '/' 开头。另外，key 为空字符串是合法的。
// data    是文件内容的访问接口（io.Reader）。
// fsize   是要上传的文件大小。
// extra   是上传的一些可选项。可以指定为nil。详细见 PutExtra 结构的描述。
func (p *FormUploader) Put(
	ctx context.Context, ret interface{}, uptoken, key string, data io.Reader, size int64, extra *PutExtra) (err error) {
	err = p.put(ctx, ret, uptoken, key, true, data, size, extra, path.Base(key))
	return
}

// PutWithoutKey 用来以表单方式上传一个文件。不指定文件上传后保存的key的情况下，文件命名方式首先看看 uptoken 中是否设置了 saveKey，
// 如果设置了 saveKey，那么按 saveKey 要求的规则生成 key，否则自动以文件的 hash 做 key。
//
// ctx     是请求的上下文。
// ret     是上传成功后返回的数据。如果 uptoken 中没有设置 CallbackUrl 或 ReturnBody，那么返回的数据结构是 PutRet 结构。
// uptoken 是由业务服务器颁发的上传凭证。
// data    是文件内容的访问接口（io.Reader）。
// fsize   是要上传的文件大小。
// extra   是上传的一些可选项。详细见 PutExtra 结构的描述。
func (p *FormUploader) PutWithoutKey(
	ctx context.Context, ret interface{}, uptoken string, data io.Reader, size int64, extra *PutExtra) (err error) {
	err = p.put(ctx, ret, uptoken, "", false, data, size, extra, "")
	return err
}

func (p *FormUploader) put(
	ctx context.Context, ret interface{}, upToken string,
	key string, hasKey bool, data io.Reader, size int64, extra *PutExtra, fileName string) error {

	if extra == nil {
		extra = &PutExtra{}
	}
	extra.init()

	seekableData, ok := data.(io.ReadSeeker)
	if !ok {
		dataBytes, rErr := internal_io.ReadAll(data)
		if rErr != nil {
			return rErr
		}
		if size <= 0 {
			size = int64(len(dataBytes))
		}
		seekableData = bytes.NewReader(dataBytes)
	}

	return p.putSeekableData(ctx, ret, upToken, key, hasKey, seekableData, size, extra, fileName)
}

func (p *FormUploader) putSeekableData(ctx context.Context, ret interface{}, upToken string,
	key string, hasKey bool, data io.ReadSeeker, dataSize int64, extra *PutExtra, fileName string) error {
	if fileName == "" {
		fileName = "Untitled"
	}
	var fileReader io.Reader = data
	if extra.OnProgress != nil {
		fileReader = &readerWithProgress{reader: data, fsize: dataSize, onProgress: extra.OnProgress}
	}

	request := apis.PostObjectRequest{
		ObjectName:  makeKeyForUploading(key, hasKey),
		UploadToken: uptoken.NewParser(upToken),
		File: http_client.MultipartFormBinaryData{
			Data:        internal_io.MakeReadSeekCloserFromLimitedReader(fileReader, dataSize),
			Name:        fileName,
			ContentType: extra.MimeType,
		},
		CustomData:   makeCustomData(extra.Params),
		ResponseBody: ret,
	}
	if crc32, ok, err := crc32FromReader(data); err != nil {
		return err
	} else if ok {
		request.Crc32 = int64(crc32)
	}
	_, err := p.storage.PostObject(ctx, &request, makeApiOptionsFromUpHost(extra.UpHost))
	return err
}

// Deprecated
func (p *FormUploader) UpHost(ak, bucket string) (upHost string, err error) {
	return getUpHost(p.Cfg, 0, 0, ak, bucket)
}

type readerWithProgress struct {
	reader     io.Reader
	uploaded   int64
	fsize      int64
	onProgress func(fsize, uploaded int64)
}

func (p *readerWithProgress) Read(b []byte) (n int, err error) {
	if p.uploaded > 0 {
		p.onProgress(p.fsize, p.uploaded)
	}

	n, err = p.reader.Read(b)
	p.uploaded += int64(n)
	if p.fsize > 0 && p.uploaded > p.fsize {
		p.uploaded = p.fsize
	}
	return
}

func (p *readerWithProgress) Seek(offset int64, whence int) (int64, error) {
	if seeker, ok := p.reader.(io.Seeker); ok {
		pos, err := seeker.Seek(offset, whence)
		if err != nil {
			return pos, err
		}
		p.uploaded = pos
		p.onProgress(p.fsize, p.uploaded)
		return pos, nil
	}
	return 0, errors.New("resource not support seek")
}

func makeCustomData(params map[string]string) map[string]string {
	customData := make(map[string]string, len(params))
	for k, v := range params {
		if (strings.HasPrefix(k, "x:") || strings.HasPrefix(k, "x-qn-meta-")) && v != "" {
			customData[k] = v
		}
	}
	return customData
}

func crc32FromReader(r io.Reader) (uint32, bool, error) {
	if readSeeker, ok := r.(io.ReadSeeker); ok {
		_, err := readSeeker.Seek(0, io.SeekStart)
		if err != nil {
			return 0, false, err
		}
		hasher := crc32.NewIEEE()
		if _, err = io.Copy(hasher, readSeeker); err != nil {
			return 0, false, err
		}
		if _, err = readSeeker.Seek(0, io.SeekStart); err != nil {
			return 0, false, err
		}
		return hasher.Sum32(), true, nil
	}
	return 0, false, nil
}
