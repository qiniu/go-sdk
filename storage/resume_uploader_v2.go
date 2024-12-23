package storage

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/qiniu/go-sdk/v7/client"
	"github.com/qiniu/go-sdk/v7/internal/clientv2"
	"github.com/qiniu/go-sdk/v7/storagev2/apis"
	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/region"
	"github.com/qiniu/go-sdk/v7/storagev2/uplog"
)

// ResumeUploaderV2 表示一个分片上传 v2 的对象
type ResumeUploaderV2 struct {
	Client  *client.Client
	Cfg     *Config
	storage *apis.Storage
}

// NewResumeUploaderV2 表示构建一个新的分片上传的对象
func NewResumeUploaderV2(cfg *Config) *ResumeUploaderV2 {
	return NewResumeUploaderV2Ex(cfg, nil)
}

// NewResumeUploaderV2Ex 表示构建一个新的分片上传 v2 的对象
func NewResumeUploaderV2Ex(cfg *Config, clt *client.Client) *ResumeUploaderV2 {
	if cfg == nil {
		cfg = NewConfig()
	}

	if clt == nil {
		clt = &client.DefaultClient
	}

	bucketQuery, _ := region.NewBucketRegionsQuery(getUcEndpoint(cfg.UseHTTPS, nil), &region.BucketRegionsQueryOptions{
		UseInsecureProtocol: !cfg.UseHTTPS,
		Client:              clt.Client,
	})

	opts := http_client.Options{
		BasicHTTPClient:     clt.Client,
		BucketQuery:         bucketQuery,
		UseInsecureProtocol: !cfg.UseHTTPS,
		AccelerateUploading: cfg.AccelerateUploading,
		HostRetryConfig:     &clientv2.RetryConfig{},
	}
	if region := cfg.GetRegion(); region != nil {
		opts.Regions = region
	}

	return &ResumeUploaderV2{
		Client:  clt,
		Cfg:     cfg,
		storage: apis.NewStorage(&opts),
	}
}

// Put 方法用来上传一个文件，支持断点续传和分块上传。
//
// ctx     是请求的上下文。
// ret     是上传成功后返回的数据。如果 upToken 中没有设置 CallbackUrl 或 ReturnBody，那么返回的数据结构是 PutRet 结构。
// upToken 是由业务服务器颁发的上传凭证。
// key     是要上传的文件访问路径。比如："foo/bar.jpg"。注意我们建议 key 不要以 '/' 开头。另外，key 为空字符串是合法的。
// f       是文件内容的访问接口。考虑到需要支持分块上传和断点续传，要的是 io.ReaderAt 接口，而不是 io.Reader。
// fsize   是要上传的文件大小。
// extra   是上传的一些可选项。详细见 RputV2Extra 结构的描述。
func (p *ResumeUploaderV2) Put(ctx context.Context, ret interface{}, upToken string, key string, f io.ReaderAt, fsize int64, extra *RputV2Extra) error {
	return p.rput(ctx, ret, upToken, key, true, f, fsize, nil, extra)
}

func (p *ResumeUploaderV2) PutWithoutSize(ctx context.Context, ret interface{}, upToken, key string, r io.Reader, extra *RputV2Extra) error {
	return p.rputWithoutSize(ctx, ret, upToken, key, true, r, extra)
}

// PutWithoutKey 方法用来上传一个文件，支持断点续传和分块上传。文件命名方式首先看看
// upToken 中是否设置了 saveKey，如果设置了 saveKey，那么按 saveKey 要求的规则生成 key，否则自动以文件的 hash 做 key。
//
// ctx     是请求的上下文。
// ret     是上传成功后返回的数据。如果 upToken 中没有设置 CallbackUrl 或 ReturnBody，那么返回的数据结构是 PutRet 结构。
// upToken 是由业务服务器颁发的上传凭证。
// f       是文件内容的访问接口。考虑到需要支持分块上传和断点续传，要的是 io.ReaderAt 接口，而不是 io.Reader。
// fsize   是要上传的文件大小。
// extra   是上传的一些可选项。详细见 RputV2Extra 结构的描述。
func (p *ResumeUploaderV2) PutWithoutKey(ctx context.Context, ret interface{}, upToken string, f io.ReaderAt, fsize int64, extra *RputV2Extra) error {
	return p.rput(ctx, ret, upToken, "", false, f, fsize, nil, extra)
}

// PutFile 用来上传一个文件，支持断点续传和分块上传。
// 和 Put 不同的只是一个通过提供文件路径来访问文件内容，一个通过 io.ReaderAt 来访问。
//
// ctx       是请求的上下文。
// ret       是上传成功后返回的数据。如果 upToken 中没有设置 CallbackUrl 或 ReturnBody，那么返回的数据结构是 PutRet 结构。
// upToken   是由业务服务器颁发的上传凭证。
// key       是要上传的文件访问路径。比如："foo/bar.jpg"。注意我们建议 key 不要以 '/' 开头。另外，key 为空字符串是合法的。
// localFile 是要上传的文件的本地路径。
// extra     是上传的一些可选项。详细见 RputV2Extra 结构的描述。
func (p *ResumeUploaderV2) PutFile(ctx context.Context, ret interface{}, upToken, key, localFile string, extra *RputV2Extra) error {
	return p.rputFile(ctx, ret, upToken, key, true, localFile, extra)
}

// PutFileWithoutKey 上传一个文件，支持断点续传和分块上传。文件命名方式首先看看
// upToken 中是否设置了 saveKey，如果设置了 saveKey，那么按 saveKey 要求的规则生成 key，否则自动以文件的 hash 做 key。
// 和 PutWithoutKey 不同的只是一个通过提供文件路径来访问文件内容，一个通过 io.ReaderAt 来访问。
//
// ctx       是请求的上下文。
// ret       是上传成功后返回的数据。如果 upToken 中没有设置 CallbackUrl 或 ReturnBody，那么返回的数据结构是 PutRet 结构。
// upToken   是由业务服务器颁发的上传凭证。
// localFile 是要上传的文件的本地路径。
// extra     是上传的一些可选项。详细见 RputV2Extra 结构的描述。
func (p *ResumeUploaderV2) PutFileWithoutKey(ctx context.Context, ret interface{}, upToken, localFile string, extra *RputV2Extra) error {
	return p.rputFile(ctx, ret, upToken, "", false, localFile, extra)
}

func (p *ResumeUploaderV2) rput(ctx context.Context, ret interface{}, upToken string, key string, hasKey bool, f io.ReaderAt, fsize int64, fileDetails *fileDetailsInfo, extra *RputV2Extra) (err error) {
	if extra == nil {
		extra = &RputV2Extra{}
	}
	extra.init()

	var (
		bucket, recorderKey string
		fileInfo            os.FileInfo = nil
	)

	if fileDetails != nil {
		fileInfo = fileDetails.fileInfo
	}

	if _, bucket, err = getAkBucketFromUploadToken(upToken); err != nil {
		return
	}

	recorderKey = getRecorderKey(extra.Recorder, upToken, key, "v2", extra.PartSize, fileDetails)

	return uploadByWorkers(
		newResumeUploaderV2Impl(p, bucket, key, hasKey, upToken, makeEndpointsFromUpHost(extra.UpHost), fileInfo, extra, ret, recorderKey),
		ctx, newSizedChunkReader(f, fsize, extra.PartSize))
}

func (p *ResumeUploaderV2) rputWithoutSize(ctx context.Context, ret interface{}, upToken string, key string, hasKey bool, r io.Reader, extra *RputV2Extra) (err error) {
	if extra == nil {
		extra = &RputV2Extra{}
	}
	extra.init()

	var bucket string

	if _, bucket, err = getAkBucketFromUploadToken(upToken); err != nil {
		return
	}

	return uploadByWorkers(
		newResumeUploaderV2Impl(p, bucket, key, hasKey, upToken, makeEndpointsFromUpHost(extra.UpHost), nil, extra, ret, ""),
		ctx, newUnsizedChunkReader(r, extra.PartSize))
}

func (p *ResumeUploaderV2) rputFile(ctx context.Context, ret interface{}, upToken string, key string, hasKey bool, localFile string, extra *RputV2Extra) (err error) {
	var (
		file        *os.File
		fileInfo    os.FileInfo
		fileDetails *fileDetailsInfo
	)

	if file, err = os.Open(localFile); err != nil {
		return
	}
	defer file.Close()

	if fileInfo, err = file.Stat(); err != nil {
		return
	}

	if fullPath, absErr := filepath.Abs(file.Name()); absErr == nil {
		fileDetails = &fileDetailsInfo{fileFullPath: fullPath, fileInfo: fileInfo}
	}

	return p.rput(ctx, ret, upToken, key, hasKey, file, fileInfo.Size(), fileDetails, extra)
}

// 初始化块请求
func (p *ResumeUploaderV2) InitParts(ctx context.Context, upToken, upHost, bucket, key string, hasKey bool, ret *InitPartsRet) error {
	return p.resumeUploaderAPIs().initParts(ctx, upToken, makeEndpointsFromUpHost(upHost), bucket, key, hasKey, ret)
}

// 发送块请求
func (p *ResumeUploaderV2) UploadParts(ctx context.Context, upToken, upHost, bucket, key string, hasKey bool, uploadId string, partNumber int64, partMD5 string, ret *UploadPartsRet, body io.Reader, size int) error {
	return p.resumeUploaderAPIs().uploadParts(ctx, upToken, makeEndpointsFromUpHost(upHost), bucket, key, hasKey, uploadId, partNumber, partMD5, ret, body, int64(size))
}

// 完成块请求
func (p *ResumeUploaderV2) CompleteParts(ctx context.Context, upToken, upHost string, ret interface{}, bucket, key string, hasKey bool, uploadId string, extra *RputV2Extra) (err error) {
	return p.resumeUploaderAPIs().completeParts(ctx, upToken, makeEndpointsFromUpHost(upHost), ret, bucket, key, hasKey, uploadId, extra)
}

func (p *ResumeUploaderV2) UpHost(ak, bucket string) (upHost string, err error) {
	return p.resumeUploaderAPIs().upHost(ak, bucket)
}

func (p *ResumeUploaderV2) resumeUploaderAPIs() *resumeUploaderAPIs {
	return &resumeUploaderAPIs{cfg: p.Cfg, storage: p.storage}
}

type (
	// 用于实现 resumeUploaderBase 的 V2 分片接口
	resumeUploaderV2Impl struct {
		cfg         *Config
		storage     *apis.Storage
		bucket      string
		key         string
		hasKey      bool
		uploadId    string
		expiredAt   int64
		upToken     string
		upEndpoints region.EndpointsProvider
		extra       *RputV2Extra
		fileInfo    os.FileInfo
		recorderKey string
		ret         interface{}
		lock        sync.Mutex
		bufPool     *sync.Pool
	}

	resumeUploaderV2RecoveryInfoContext struct {
		Offset     int64  `json:"o"`
		Etag       string `json:"e"`
		PartSize   int    `json:"s"`
		PartNumber int64  `json:"p"`
	}

	resumeUploaderV2RecoveryInfo struct {
		RecorderVersion string                                `json:"v"`
		Region          *Region                               `json:"r"`
		FileSize        int64                                 `json:"s"`
		ModTimeStamp    int64                                 `json:"m"`
		ExpiredAt       int64                                 `json:"e"`
		UploadId        string                                `json:"i"`
		Contexts        []resumeUploaderV2RecoveryInfoContext `json:"c"`
	}
)

func newResumeUploaderV2Impl(resumeUploader *ResumeUploaderV2, bucket, key string, hasKey bool, upToken string, upEndpoints region.EndpointsProvider, fileInfo os.FileInfo, extra *RputV2Extra, ret interface{}, recorderKey string) *resumeUploaderV2Impl {
	bucketQuery, _ := region.NewBucketRegionsQuery(getUcEndpoint(resumeUploader.Cfg.UseHTTPS, nil), &region.BucketRegionsQueryOptions{
		UseInsecureProtocol: !resumeUploader.Cfg.UseHTTPS,
		Client:              resumeUploader.Client.Client,
	})
	opts := http_client.Options{
		BasicHTTPClient:     resumeUploader.Client.Client,
		BucketQuery:         bucketQuery,
		UseInsecureProtocol: !resumeUploader.Cfg.UseHTTPS,
		AccelerateUploading: resumeUploader.Cfg.AccelerateUploading,
		HostRetryConfig:     &clientv2.RetryConfig{},
	}
	if region := resumeUploader.Cfg.GetRegion(); region != nil {
		opts.Regions = region
	}
	if extra != nil {
		if extra.TryTimes > 0 {
			opts.HostRetryConfig.RetryMax = extra.TryTimes
		}
		if extra.HostFreezeDuration > 0 {
			opts.HostFreezeDuration = extra.HostFreezeDuration
		}
	}
	return &resumeUploaderV2Impl{
		cfg:         resumeUploader.Cfg,
		bucket:      bucket,
		key:         key,
		hasKey:      hasKey,
		upToken:     upToken,
		upEndpoints: upEndpoints,
		fileInfo:    fileInfo,
		recorderKey: recorderKey,
		extra:       extra,
		ret:         ret,
		storage:     apis.NewStorage(&opts),
		bufPool: &sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, extra.PartSize))
			},
		},
	}
}

func (impl *resumeUploaderV2Impl) initUploader(ctx context.Context) (recovered []int64, recoveredSize int64, err error) {
	var ret InitPartsRet

	if impl.extra.Recorder != nil && len(impl.recorderKey) > 0 {
		if recorderData, err := impl.extra.Recorder.Get(impl.recorderKey); err == nil {
			if recovered, recoveredSize = impl.recover(ctx, recorderData); len(recovered) > 0 {
				return recovered, recoveredSize, nil
			}
			if len(recovered) == 0 {
				impl.deleteUploadRecordIfNeed(nil, true)
			}
		}
	}

	if err = impl.resumeUploaderAPIs().initParts(ctx, impl.upToken, impl.upEndpoints, impl.bucket, impl.key, impl.hasKey, &ret); err == nil {
		impl.uploadId = ret.UploadID
		impl.expiredAt = ret.ExpireAt
	}
	return nil, 0, err
}

func (impl *resumeUploaderV2Impl) uploadChunk(ctx context.Context, c chunk) error {
	var (
		apis   = impl.resumeUploaderAPIs()
		ret    UploadPartsRet
		buffer = impl.bufPool.Get().(*bytes.Buffer)
	)
	defer impl.bufPool.Put(buffer)

	partNumber := c.id + 1
	hasher := md5.New()
	buffer.Reset()
	chunkSize, err := io.Copy(hasher, io.TeeReader(io.NewSectionReader(c.reader, 0, c.size), buffer))
	if err != nil {
		impl.extra.NotifyErr(partNumber, err)
		return err
	} else if chunkSize == 0 {
		return nil
	}

	md5Value := hex.EncodeToString(hasher.Sum(nil))
	seekableData := bytes.NewReader(buffer.Bytes())
	err = apis.uploadParts(ctx, impl.upToken, impl.upEndpoints, impl.bucket, impl.key, impl.hasKey, impl.uploadId, partNumber, md5Value, &ret, seekableData, chunkSize)
	if err != nil {
		impl.extra.NotifyErr(partNumber, err)
		impl.deleteUploadRecordIfNeed(err, false)
	} else {
		impl.extra.Notify(partNumber, &ret)

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		func() {
			impl.lock.Lock()
			defer impl.lock.Unlock()
			impl.extra.Progresses = append(impl.extra.Progresses, UploadPartInfo{
				Etag: ret.Etag, PartNumber: partNumber, partSize: int(chunkSize), fileOffset: c.offset,
			})
			_ = impl.save(ctx)
		}()
	}
	return err
}

func (impl *resumeUploaderV2Impl) final(ctx context.Context) error {
	if impl.extra.Recorder != nil && len(impl.recorderKey) > 0 {
		impl.deleteUploadRecordIfNeed(nil, true)
	}

	sort.Sort(uploadPartInfos(impl.extra.Progresses))
	err := impl.resumeUploaderAPIs().completeParts(ctx, impl.upToken, impl.upEndpoints, impl.ret, impl.bucket, impl.key, impl.hasKey, impl.uploadId, impl.extra)
	impl.deleteUploadRecordIfNeed(err, false)
	return err
}

func (impl *resumeUploaderV2Impl) version() uplog.UpApiVersion {
	return uplog.UpApiVersionV2
}

func (impl *resumeUploaderV2Impl) getUpToken() string {
	return impl.upToken
}

func (impl *resumeUploaderV2Impl) getBucket() string {
	return impl.bucket
}

func (impl *resumeUploaderV2Impl) getKey() (string, bool) {
	return impl.key, impl.hasKey
}

func (impl *resumeUploaderV2Impl) deleteUploadRecordIfNeed(err error, force bool) {
	// 无效删除之前的记录
	if force || (isContextExpiredError(err) && impl.extra.Recorder != nil && len(impl.recorderKey) > 0) {
		_ = impl.extra.Recorder.Delete(impl.recorderKey)
	}
}

func (impl *resumeUploaderV2Impl) recover(ctx context.Context, recoverData []byte) (recovered []int64, recoveredSize int64) {
	var recoveryInfo resumeUploaderV2RecoveryInfo
	if err := json.Unmarshal(recoverData, &recoveryInfo); err != nil {
		return
	}
	if impl.fileInfo == nil || recoveryInfo.FileSize != impl.fileInfo.Size() ||
		recoveryInfo.RecorderVersion != uploadRecordVersion ||
		recoveryInfo.ModTimeStamp != impl.fileInfo.ModTime().UnixNano() {
		return
	}

	if isUploadContextExpired(recoveryInfo.ExpiredAt) {
		return
	}

	impl.uploadId = recoveryInfo.UploadId
	impl.expiredAt = recoveryInfo.ExpiredAt

	for _, c := range recoveryInfo.Contexts {
		impl.extra.Progresses = append(impl.extra.Progresses, UploadPartInfo{
			Etag: c.Etag, PartNumber: c.PartNumber, fileOffset: c.Offset, partSize: c.PartSize,
		})
		recoveredSize += int64(c.PartSize)
		recovered = append(recovered, int64(c.Offset))
	}

	return
}

func (impl *resumeUploaderV2Impl) save(ctx context.Context) error {
	var (
		recoveryInfo  resumeUploaderV2RecoveryInfo
		recoveredData []byte
		err           error
	)

	if impl.fileInfo == nil || impl.extra.Recorder == nil || len(impl.recorderKey) == 0 {
		return nil
	}

	recoveryInfo.RecorderVersion = uploadRecordVersion
	recoveryInfo.Region = impl.cfg.Region
	recoveryInfo.FileSize = impl.fileInfo.Size()
	recoveryInfo.ModTimeStamp = impl.fileInfo.ModTime().UnixNano()
	recoveryInfo.UploadId = impl.uploadId
	recoveryInfo.ExpiredAt = impl.expiredAt
	recoveryInfo.Contexts = make([]resumeUploaderV2RecoveryInfoContext, 0, len(impl.extra.Progresses))

	for _, progress := range impl.extra.Progresses {
		recoveryInfo.Contexts = append(recoveryInfo.Contexts, resumeUploaderV2RecoveryInfoContext{
			Offset: progress.fileOffset, Etag: progress.Etag, PartSize: progress.partSize, PartNumber: progress.PartNumber,
		})
	}

	if recoveredData, err = json.Marshal(recoveryInfo); err != nil {
		return err
	}

	return impl.extra.Recorder.Set(impl.recorderKey, recoveredData)
}

func (impl *resumeUploaderV2Impl) resumeUploaderAPIs() *resumeUploaderAPIs {
	return &resumeUploaderAPIs{cfg: impl.cfg, storage: impl.storage}
}
