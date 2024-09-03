package storage

import (
	"context"
	"io"
	"strings"
	"time"

	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
	"github.com/qiniu/go-sdk/v7/storagev2/apis"
	"github.com/qiniu/go-sdk/v7/storagev2/apis/resumable_upload_v2_complete_multipart_upload"
	"github.com/qiniu/go-sdk/v7/storagev2/region"
	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"
)

type resumeUploaderAPIs struct {
	cfg     *Config
	storage *apis.Storage
}

// BlkputRet 表示分片上传每个片上传完毕的返回值
type BlkputRet struct {
	Ctx        string `json:"ctx"`
	Checksum   string `json:"checksum"`
	Crc32      uint32 `json:"crc32"`
	Offset     uint32 `json:"offset"`
	Host       string `json:"host"`
	ExpiredAt  int64  `json:"expired_at"`
	chunkSize  int
	fileOffset int64
	blkIdx     int
}

func (p *resumeUploaderAPIs) mkBlk(
	ctx context.Context, upToken string, upEndpoints region.EndpointsProvider,
	ret *BlkputRet, blockSize int64, body io.Reader, size int64,
) error {
	response, err := p.storage.ResumableUploadV1MakeBlock(
		ctx,
		&apis.ResumableUploadV1MakeBlockRequest{
			BlockSize: blockSize,
			UpToken:   uptoken.NewParser(upToken),
			Body:      internal_io.MakeReadSeekCloserFromReader(body),
		},
		makeApiOptionsFromUpEndpoints(upEndpoints),
	)
	if err != nil {
		return err
	}
	*ret = BlkputRet{
		Ctx:       response.Ctx,
		Checksum:  response.Checksum,
		Crc32:     uint32(response.Crc32),
		Offset:    uint32(response.Offset),
		Host:      response.Host,
		ExpiredAt: response.ExpiredAt,
	}
	return nil
}

func (p *resumeUploaderAPIs) bput(
	ctx context.Context, upToken string, upEndpoints region.EndpointsProvider,
	ret *BlkputRet, body io.Reader, size int64,
) error {
	response, err := p.storage.ResumableUploadV1Bput(
		ctx,
		&apis.ResumableUploadV1BputRequest{
			Ctx:         ret.Ctx,
			ChunkOffset: int64(ret.Offset),
			UpToken:     uptoken.NewParser(upToken),
			Body:        internal_io.MakeReadSeekCloserFromReader(body),
		},
		makeApiOptionsFromUpEndpoints(upEndpoints),
	)
	if err != nil {
		return err
	}
	*ret = BlkputRet{
		Ctx:       response.Ctx,
		Checksum:  response.Checksum,
		Crc32:     uint32(response.Crc32),
		Offset:    uint32(response.Offset),
		Host:      response.Host,
		ExpiredAt: response.ExpiredAt,
	}
	return nil
}

// RputExtra 表示分片上传额外可以指定的参数
type RputExtra struct {
	Recorder Recorder // 可选。上传进度记录

	// 可选。
	// 用户自定义参数：key 以"x:"开头，而且 value 不能为空 eg: key为x:qqq
	// 自定义 meta：key 以"x-qn-meta-"开头，而且 value 不能为空 eg: key为x-qn-meta-aaa
	Params             map[string]string
	UpHost             string
	MimeType           string                                        // 可选。
	ChunkSize          int                                           // 可选。每次上传的Chunk大小
	TryTimes           int                                           // 可选。尝试次数
	HostFreezeDuration time.Duration                                 // 可选。主备域名冻结时间（默认：600s），当一个域名请求失败（单个域名会被重试 TryTimes 次），会被冻结一段时间，使用备用域名进行重试，在冻结时间内，域名不能被使用，当一个操作中所有域名竣备冻结操作不在进行重试，返回最后一次操作的错误。
	Progresses         []BlkputRet                                   // 可选。上传进度
	Notify             func(blkIdx int, blkSize int, ret *BlkputRet) // 可选。进度提示（注意多个block是并行传输的）
	NotifyErr          func(blkIdx int, blkSize int, err error)
}

func (extra *RputExtra) init() {
	if extra.ChunkSize == 0 {
		extra.ChunkSize = settings.ChunkSize
	}
	if extra.TryTimes == 0 {
		extra.TryTimes = settings.TryTimes
	}
	if extra.HostFreezeDuration <= 0 {
		extra.HostFreezeDuration = 10 * 60 * time.Second
	}
	if extra.Notify == nil {
		extra.Notify = func(blkIdx, blkSize int, ret *BlkputRet) {}
	}
	if extra.NotifyErr == nil {
		extra.NotifyErr = func(blkIdx, blkSize int, err error) {}
	}
}

func (p *resumeUploaderAPIs) mkfile(
	ctx context.Context, upToken string, upEndpoints region.EndpointsProvider,
	ret interface{}, key string, hasKey bool, fsize int64, extra *RputExtra,
) error {
	if extra == nil {
		extra = &RputExtra{}
	}
	ctxs := make([]string, len(extra.Progresses))
	for i, progress := range extra.Progresses {
		ctxs[i] = progress.Ctx
	}
	_, err := p.storage.ResumableUploadV1MakeFile(
		ctx,
		&apis.ResumableUploadV1MakeFileRequest{
			Size:         fsize,
			ObjectName:   makeKeyForUploading(key, hasKey),
			MimeType:     extra.MimeType,
			CustomData:   makeCustomData(extra.Params),
			UpToken:      uptoken.NewParser(upToken),
			Body:         internal_io.MakeReadSeekCloserFromReader(strings.NewReader(strings.Join(ctxs, ","))),
			ResponseBody: ret,
		},
		makeApiOptionsFromUpEndpoints(upEndpoints),
	)
	return err
}

// InitPartsRet 表示分片上传 v2 初始化完毕的返回值
type InitPartsRet struct {
	UploadID string `json:"uploadId"`
	ExpireAt int64  `json:"expireAt"`
}

func (p *resumeUploaderAPIs) initParts(
	ctx context.Context, upToken string, upEndpoints region.EndpointsProvider,
	bucket, key string, hasKey bool, ret *InitPartsRet,
) error {
	response, err := p.storage.ResumableUploadV2InitiateMultipartUpload(
		ctx,
		&apis.ResumableUploadV2InitiateMultipartUploadRequest{
			BucketName: bucket,
			ObjectName: makeKeyForUploading(key, hasKey),
			UpToken:    uptoken.NewParser(upToken),
		},
		makeApiOptionsFromUpEndpoints(upEndpoints),
	)
	if err != nil {
		return err
	}
	*ret = InitPartsRet{
		UploadID: response.UploadId,
		ExpireAt: response.ExpiredAt,
	}
	return nil
}

// UploadPartsRet 表示分片上传 v2 每个片上传完毕的返回值
type UploadPartsRet struct {
	Etag string `json:"etag"`
	MD5  string `json:"md5"`
}

func (p *resumeUploaderAPIs) uploadParts(
	ctx context.Context, upToken string, upEndpoints region.EndpointsProvider,
	bucket, key string, hasKey bool, uploadId string, partNumber int64, partMD5 string, ret *UploadPartsRet, body io.Reader, size int64,
) error {
	response, err := p.storage.ResumableUploadV2UploadPart(
		ctx,
		&apis.ResumableUploadV2UploadPartRequest{
			BucketName: bucket,
			ObjectName: makeKeyForUploading(key, hasKey),
			UploadId:   uploadId,
			PartNumber: partNumber,
			Md5:        partMD5,
			UpToken:    uptoken.NewParser(upToken),
			Body:       internal_io.MakeReadSeekCloserFromLimitedReader(body, size),
		},
		makeApiOptionsFromUpEndpoints(upEndpoints),
	)
	if err != nil {
		return err
	}
	*ret = UploadPartsRet{
		Etag: response.Etag,
		MD5:  response.Md5,
	}
	return nil
}

type UploadPartInfo struct {
	Etag       string `json:"etag"`
	PartNumber int64  `json:"partNumber"`
	partSize   int
	fileOffset int64
}

// RputV2Extra 表示分片上传 v2 额外可以指定的参数
type RputV2Extra struct {
	Recorder           Recorder          // 可选。上传进度记录
	Metadata           map[string]string // 可选。用户自定义文件 metadata 信息
	CustomVars         map[string]string // 可选。用户自定义参数，以"x:"开头，而且值不能为空，否则忽略
	UpHost             string
	MimeType           string                                      // 可选。
	PartSize           int64                                       // 可选。每次上传的块大小
	TryTimes           int                                         // 可选。尝试次数
	HostFreezeDuration time.Duration                               // 可选。主备域名冻结时间（默认：600s），当一个域名请求失败（单个域名会被重试 TryTimes 次），会被冻结一段时间，使用备用域名进行重试，在冻结时间内，域名不能被使用，当一个操作中所有域名竣备冻结操作不在进行重试，返回最后一次操作的错误。
	Progresses         []UploadPartInfo                            // 上传进度
	Notify             func(partNumber int64, ret *UploadPartsRet) // 可选。进度提示（注意多个block是并行传输的）
	NotifyErr          func(partNumber int64, err error)
}

func (extra *RputV2Extra) init() {
	if extra.PartSize == 0 {
		extra.PartSize = settings.PartSize
	}
	if extra.TryTimes == 0 {
		extra.TryTimes = settings.TryTimes
	}
	if extra.HostFreezeDuration <= 0 {
		extra.HostFreezeDuration = 10 * 60 * time.Second
	}
	if extra.Notify == nil {
		extra.Notify = func(partNumber int64, ret *UploadPartsRet) {}
	}
	if extra.NotifyErr == nil {
		extra.NotifyErr = func(partNumber int64, err error) {}
	}
}

func (p *resumeUploaderAPIs) completeParts(
	ctx context.Context, upToken string, upEndpoints region.EndpointsProvider, ret interface{},
	bucket, key string, hasKey bool, uploadId string, extra *RputV2Extra,
) error {
	parts := make([]resumable_upload_v2_complete_multipart_upload.PartInfo, 0, len(extra.Progresses))
	for i := range extra.Progresses {
		parts = append(parts, resumable_upload_v2_complete_multipart_upload.PartInfo{
			PartNumber: extra.Progresses[i].PartNumber,
			Etag:       extra.Progresses[i].Etag,
		})
	}
	customVars := make(map[string]string, len(extra.CustomVars))
	for k, v := range extra.CustomVars {
		if strings.HasPrefix(k, "x:") && v != "" {
			customVars[k] = v
		}
	}
	_, err := p.storage.ResumableUploadV2CompleteMultipartUpload(
		ctx,
		&apis.ResumableUploadV2CompleteMultipartUploadRequest{
			BucketName:   bucket,
			ObjectName:   makeKeyForUploading(key, hasKey),
			UploadId:     uploadId,
			UpToken:      uptoken.NewParser(upToken),
			Parts:        parts,
			MimeType:     extra.MimeType,
			Metadata:     extra.Metadata,
			CustomVars:   customVars,
			ResponseBody: ret,
		},
		makeApiOptionsFromUpEndpoints(upEndpoints),
	)
	return err
}

func (p *resumeUploaderAPIs) upHost(ak, bucket string) (upHost string, err error) {
	return getUpHost(p.cfg, 0, 0, ak, bucket)
}

func makeEndpointsFromUpHost(upHost string) region.EndpointsProvider {
	if upHost != "" {
		return &region.Endpoints{Preferred: []string{upHost}}
	}
	return nil
}

func makeApiOptionsFromUpEndpoints(upEndpoints region.EndpointsProvider) *apis.Options {
	if upEndpoints != nil {
		return &apis.Options{
			OverwrittenEndpoints: upEndpoints,
		}
	}
	return nil
}

func makeKeyForUploading(key string, hasKey bool) *string {
	if hasKey {
		return &key
	} else {
		return nil
	}
}

type blkputRets []BlkputRet

func (rets blkputRets) Len() int {
	return len(rets)
}

func (rets blkputRets) Less(i, j int) bool {
	return rets[i].blkIdx < rets[j].blkIdx
}

func (rets blkputRets) Swap(i, j int) {
	rets[i], rets[j] = rets[j], rets[i]
}

type uploadPartInfos []UploadPartInfo

func (infos uploadPartInfos) Len() int {
	return len(infos)
}

func (infos uploadPartInfos) Less(i, j int) bool {
	return infos[i].PartNumber < infos[j].PartNumber
}

func (infos uploadPartInfos) Swap(i, j int) {
	infos[i], infos[j] = infos[j], infos[i]
}
