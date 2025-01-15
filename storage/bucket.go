package storage

// TODO:
// BucketManager 每个接口的基本逻辑都是设置Mac信息， 获取请求地址， 发送HTTP请求。
// 后期可以调整抽象出Request struct, APIOperation struct， 这样不用每个接口都要写
// 重复的逻辑

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/qiniu/go-sdk/v7/internal/clientv2"
	"github.com/qiniu/go-sdk/v7/storagev2/apis"
	"github.com/qiniu/go-sdk/v7/storagev2/apis/batch_ops"
	"github.com/qiniu/go-sdk/v7/storagev2/apis/get_bucket_domains_v3"
	"github.com/qiniu/go-sdk/v7/storagev2/backoff"
	"github.com/qiniu/go-sdk/v7/storagev2/chooser"
	"github.com/qiniu/go-sdk/v7/storagev2/downloader"
	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/region"
	"github.com/qiniu/go-sdk/v7/storagev2/resolver"
	"github.com/qiniu/go-sdk/v7/storagev2/retrier"

	"github.com/qiniu/go-sdk/v7/auth"
	clientv1 "github.com/qiniu/go-sdk/v7/client"
)

// 资源管理相关的默认域名
const (
	DefaultRsHost  = "rs.qiniu.com"
	DefaultRsfHost = "rsf.qiniu.com"
	DefaultAPIHost = "api.qiniu.com"
	DefaultPubHost = "pu.qbox.me:10200"
)

// FileInfo 文件基本信息
type FileInfo struct {

	// 资源内容的大小，单位：字节。
	Fsize int64 `json:"fsize"`

	// 文件的HASH值，使用hash值算法计算。
	Hash string `json:"hash"`

	// 资源的 MIME 类型。
	MimeType string `json:"mimeType"`

	/**
	 * 资源的存储类型
	 * 0 表示标准存储
	 * 1 表示低频存储
	 * 2 表示归档存储
	 * 3 表示深度归档存储
	 * 4 表示归档存储直读
	 */
	Type int `json:"type"`

	// 上传时间，单位：100纳秒，其值去掉低七位即为Unix时间戳。
	PutTime int64 `json:"putTime"`

	/**
	 * 归档/深度归档存储文件的解冻状态，归档/深度归档文件冻结时，不返回该字段。
	 * 1 表示解冻中
	 * 2 表示解冻完成
	 */
	RestoreStatus int `json:"restoreStatus"`

	/**
	 * 文件的存储状态，即禁用状态和启用状态间的的互相转换，请参考：文件状态。
	 * 0 表示启用
	 * 1 表示禁用
	 */
	Status int `json:"status"`

	/**
	 * 文件的 md5 值
	 */
	Md5 string `json:"md5"`

	/**
	 * 文件上传时设置的endUser
	 */
	EndUser string `json:"endUser"`

	/**
	 * 自定义元数据，含 meta_key 参数 及对应 metaValue 值；仅当自定义元数据后返回该字段。
	 */
	MetaData map[string]string `json:"x-qn-meta"`

	/**
	 * 文件过期删除日期，int64 类型，Unix 时间戳格式，具体文件过期日期计算参考 生命周期管理。
	 * 文件在设置过期时间后才会返回该字段（通过生命周期规则设置文件过期时间，仅对该功能发布后满足规则条件新上传文件返回该字段；
	 * 历史文件想要返回该字段需要在功能发布后可通过 修改文件过期删除时间 API 或者 修改文件生命周期 API 指定过期时间；对于已
	 * 经设置过过期时间的历史文件，到期都会正常过期删除，只是服务端没有该字段返回)
	 *
	 * 例如：值为1568736000的时间，表示文件会在2019/9/18当天内删除。
	 */
	Expiration int64 `json:"expiration"`

	/**
	 * 文件生命周期中转为低频存储的日期，int64 类型，Unix 时间戳格式 ，具体日期计算参考 生命周期管理。
	 * 文件在设置转低频后才会返回该字段（通过生命周期规则设置文件转低频，仅对该功能发布后满足规则条件新上传文件返回该字段；
	 * 历史文件想要返回该字段需要在功能发布后可通过 修改文件生命周期 API 指定转低频时间；对于已经设置过转低频时间的历史文
	 * 件，到期都会正常执行，只是服务端没有该字段返回)
	 *
	 * 例如：值为1568736000的时间，表示文件会在2019/9/18当天转为低频存储类型。
	 */
	TransitionToIA int64 `json:"transitionToIA"`

	/**
	 * 文件生命周期中转为归档直读存储的日期，int64 类型，Unix 时间戳格式 ，具体日期计算参考 生命周期管理。
	 * 文件在设置转归档直读后才会返回该字段（通过生命周期规则设置文件转归档直读，仅对该功能发布后满足规则条件新上传文件返回该字段；
	 * 历史文件想要返回该字段需要在功能发布后可通过 修改文件生命周期 API 指定转归档直读时间；对于已经设置过转归档直读时间的历史文
	 * 件，到期都会正常执行，只是服务端没有该字段返回)
	 *
	 * 例如：值为1568736000的时间，表示文件会在2019/9/18当天转为归档直读存储类型。
	 */
	TransitionToArchiveIR int64 `json:"transitionToArchiveIR"`

	/**
	 * 文件生命周期中转为归档存储的日期，int64 类型，Unix 时间戳格式 ，具体日期计算参考 生命周期管理。
	 * 文件在设置转归档后才会返回该字段（通过生命周期规则设置文件转归档，仅对该功能发布后满足规则条件新上传文件返回该字段；
	 * 历史文件想要返回该字段需要在功能发布后可通过 修改文件生命周期 API 指定转归档时间；对于已经设置过转归档时间的历史文
	 * 件，到期都会正常执行，只是服务端没有该字段返回)
	 *
	 * 例如：值为1568736000的时间，表示文件会在2019/9/18当天转为归档存储类型。
	 */
	TransitionToArchive int64 `json:"transitionToARCHIVE"`

	/**
	 * 文件生命周期中转为深度归档存储的日期，int64 类型，Unix 时间戳格式 ，具体日期计算参考 生命周期管理。
	 * 文件在设置转深度归档后才会返回该字段（通过生命周期规则设置文件转深度归档，仅对该功能发布后满足规则条件新上传文件返回该字段；
	 * 历史文件想要返回该字段需要在功能发布后可通过 修改文件生命周期 API 指定转深度归档时间；对于已经设置过转深度归档时间的历史文
	 * 件，到期都会正常执行，只是服务端没有该字段返回)
	 *
	 * 例如：值为1568736000的时间，表示文件会在2019/9/18当天转为深度归档存储类型。
	 */
	TransitionToDeepArchive int64 `json:"transitionToDeepArchive"`

	// 分拣分片信息，可能为空
	Parts []int64 `json:"parts"`
}

func (f *FileInfo) String() string {
	str := ""
	str += fmt.Sprintf("Hash:                      %s\n", f.Hash)
	str += fmt.Sprintf("Fsize:                     %d\n", f.Fsize)
	str += fmt.Sprintf("PutTime:                   %d\n", f.PutTime)
	str += fmt.Sprintf("MimeType:                  %s\n", f.MimeType)
	str += fmt.Sprintf("Type:                      %d\n", f.Type)
	str += fmt.Sprintf("RestoreStatus:             %d\n", f.RestoreStatus)
	str += fmt.Sprintf("Status:                    %d\n", f.Status)
	str += fmt.Sprintf("Md5:                       %s\n", f.Md5)
	str += fmt.Sprintf("EndUser:                   %s\n", f.EndUser)
	str += fmt.Sprintf("Expiration:                %d\n", f.Expiration)
	str += fmt.Sprintf("TransitionToIA:            %d\n", f.TransitionToIA)
	str += fmt.Sprintf("TransitionToArchiveIR:     %d\n", f.TransitionToArchiveIR)
	str += fmt.Sprintf("TransitionToArchive:       %d\n", f.TransitionToArchive)
	str += fmt.Sprintf("TransitionToDeepArchive:   %d\n", f.TransitionToDeepArchive)
	str += fmt.Sprintf("MetaData:                  %s\n", f.MetaData)
	return str
}

// FetchRet 资源抓取的返回值
type FetchRet struct {
	Hash     string `json:"hash"`
	Fsize    int64  `json:"fsize"`
	MimeType string `json:"mimeType"`
	Key      string `json:"key"`
}

func (r *FetchRet) String() string {
	str := ""
	str += fmt.Sprintf("Key:      %s\n", r.Key)
	str += fmt.Sprintf("Hash:     %s\n", r.Hash)
	str += fmt.Sprintf("Fsize:    %d\n", r.Fsize)
	str += fmt.Sprintf("MimeType: %s\n", r.MimeType)
	return str
}

// BatchOpRet 为批量执行操作的返回值
// 批量操作支持 stat，copy，delete，move，chgm，chtype，deleteAfterDays几个操作
// 其中 stat 为获取文件的基本信息，如果文件存在则返回基本信息，如果文件不存在返回 error 。
// 其他的操作，如果成功，则返回 code，不成功会同时返回 error 信息，可以根据 error 信息来判断问题所在。
type BatchOpRet struct {
	Code int `json:"code,omitempty"`
	Data struct {
		// 资源内容的大小，单位：字节。
		Fsize int64 `json:"fsize"`

		// 文件的HASH值，使用hash值算法计算。
		Hash string `json:"hash"`

		// 资源的 MIME 类型。
		MimeType string `json:"mimeType"`

		/**
		 * 资源的存储类型
		 * 0 表示标准存储
		 * 1 表示低频存储
		 * 2 表示归档存储
		 * 3 表示深度归档存储
		 * 4 表示归档直读存储
		 */
		Type int `json:"type"`

		// 上传时间，单位：100纳秒，其值去掉低七位即为Unix时间戳。
		PutTime int64 `json:"putTime"`

		/**
		 * 归档/深度归档存储文件的解冻状态，归档/深度归档文件冻结时，不返回该字段。
		 * 1 表示解冻中
		 * 2 表示解冻完成
		 */
		RestoreStatus *int `json:"restoreStatus"`

		/**
		 * 文件的存储状态，即禁用状态和启用状态间的的互相转换，请参考：文件状态。
		 * 0 表示启用
		 * 1 表示禁用
		 */
		Status *int `json:"status"`

		/**
		 * 文件的 md5 值
		 */
		Md5 string `json:"md5"`

		/**
		 * 文件上传时设置的endUser
		 */
		EndUser string `json:"endUser"`

		/**
		 * 文件过期删除日期，Unix 时间戳格式，具体文件过期日期计算参考 生命周期管理。
		 * 文件在设置过期时间后才会返回该字段（通过生命周期规则设置文件过期时间，仅对该功能发布后满足规则条件新上传文件返回该字段；
		 * 历史文件想要返回该字段需要在功能发布后可通过 修改文件过期删除时间 API 或者 修改文件生命周期 API 指定过期时间；对于已
		 * 经设置过过期时间的历史文件，到期都会正常过期删除，只是服务端没有该字段返回)
		 *
		 * 例如：值为1568736000的时间，表示文件会在2019/9/18当天内删除。
		 */
		Expiration *int64 `json:"expiration"`

		/**
		 * 文件生命周期中转为低频存储的日期，Unix 时间戳格式 ，具体日期计算参考 生命周期管理。
		 * 文件在设置转低频后才会返回该字段（通过生命周期规则设置文件转低频，仅对该功能发布后满足规则条件新上传文件返回该字段；
		 * 历史文件想要返回该字段需要在功能发布后可通过 修改文件生命周期 API 指定转低频时间；对于已经设置过转低频时间的历史文
		 * 件，到期都会正常执行，只是服务端没有该字段返回)
		 *
		 * 例如：值为1568736000的时间，表示文件会在2019/9/18当天转为低频存储类型。
		 */
		TransitionToIA *int64 `json:"transitionToIA"`

		/**
		 * 文件生命周期中转为归档直读存储的日期，int64 类型，Unix 时间戳格式 ，具体日期计算参考 生命周期管理。
		 * 文件在设置转归档直读后才会返回该字段（通过生命周期规则设置文件转归档直读，仅对该功能发布后满足规则条件新上传文件返回该字段；
		 * 历史文件想要返回该字段需要在功能发布后可通过 修改文件生命周期 API 指定转归档直读时间；对于已经设置过转归档直读时间的历史文
		 * 件，到期都会正常执行，只是服务端没有该字段返回)
		 *
		 * 例如：值为1568736000的时间，表示文件会在2019/9/18当天转为归档直读存储类型。
		 */
		TransitionToArchiveIR int64 `json:"transitionToArchiveIR"`

		/**
		 * 文件生命周期中转为归档存储的日期，Unix 时间戳格式 ，具体日期计算参考 生命周期管理。
		 * 文件在设置转归档后才会返回该字段（通过生命周期规则设置文件转归档，仅对该功能发布后满足规则条件新上传文件返回该字段；
		 * 历史文件想要返回该字段需要在功能发布后可通过 修改文件生命周期 API 指定转归档时间；对于已经设置过转归档时间的历史文
		 * 件，到期都会正常执行，只是服务端没有该字段返回)
		 *
		 * 例如：值为1568736000的时间，表示文件会在2019/9/18当天转为归档存储类型。
		 */
		TransitionToArchive *int64 `json:"transitionToARCHIVE"`

		/**
		 * 文件生命周期中转为深度归档存储的日期，Unix 时间戳格式 ，具体日期计算参考 生命周期管理。
		 * 文件在设置转深度归档后才会返回该字段（通过生命周期规则设置文件转深度归档，仅对该功能发布后满足规则条件新上传文件返回该字段；
		 * 历史文件想要返回该字段需要在功能发布后可通过 修改文件生命周期 API 指定转深度归档时间；对于已经设置过转深度归档时间的历史文
		 * 件，到期都会正常执行，只是服务端没有该字段返回)
		 *
		 * 例如：值为1568736000的时间，表示文件会在2019/9/18当天转为深度归档存储类型。
		 */
		TransitionToDeepArchive *int64 `json:"transitionToDeepArchive"`

		Error string `json:"error"`
	} `json:"data,omitempty"`
}

type BucketManagerOptions struct {
	// 单域名重试次数，当前只有 uc 相关的服务有多域名
	RetryMax int

	// 主备域名冻结时间（默认：600s），当一个域名请求失败（单个域名会被重试 TryTimes 次），会被冻结一段时间，使用备用域名进行重试，在冻结时间内，域名不能被使用，当一个操作中所有域名竣备冻结操作不在进行重试，返回最后一次操作的错误。
	HostFreezeDuration time.Duration

	// 域名解析器
	Resolver resolver.Resolver

	// 域名选择器
	Chooser chooser.Chooser

	// 退避器
	Backoff backoff.Backoff

	// 重试器
	Retrier retrier.Retrier

	// 签名前回调函数
	BeforeSign func(*http.Request)

	// 签名后回调函数
	AfterSign func(*http.Request)

	// 签名失败回调函数
	SignError func(*http.Request, error)

	// 域名解析前回调函数
	BeforeResolve func(*http.Request)

	// 域名解析后回调函数
	AfterResolve func(*http.Request, []net.IP)

	// 域名解析错误回调函数
	ResolveError func(*http.Request, error)

	// 退避前回调函数
	BeforeBackoff func(*http.Request, *retrier.RetrierOptions, time.Duration)

	// 退避后回调函数
	AfterBackoff func(*http.Request, *retrier.RetrierOptions, time.Duration)

	// 请求前回调函数
	BeforeRequest func(*http.Request, *retrier.RetrierOptions)

	// 请求后回调函数
	AfterResponse func(*http.Response, *retrier.RetrierOptions, error)
}

// BucketManager 提供了对资源进行管理的操作
type BucketManager struct {
	Client          *clientv1.Client
	Mac             *auth.Credentials
	Cfg             *Config
	options         BucketManagerOptions
	apiClient       *apis.Storage
	downloadManager *downloader.DownloadManager
}

// NewBucketManager 用来构建一个新的资源管理对象
func NewBucketManager(mac *auth.Credentials, cfg *Config) *BucketManager {
	return NewBucketManagerEx(mac, cfg, &clientv1.DefaultClient)
}

// NewBucketManagerEx 用来构建一个新的资源管理对象
func NewBucketManagerEx(mac *auth.Credentials, cfg *Config, clt *clientv1.Client) *BucketManager {
	return NewBucketManagerExWithOptions(mac, cfg, clt, BucketManagerOptions{})
}

func NewBucketManagerExWithOptions(mac *auth.Credentials, cfg *Config, clt *clientv1.Client, options BucketManagerOptions) *BucketManager {
	if cfg == nil {
		cfg = NewConfig()
	}
	if mac == nil {
		mac = auth.Default()
	}

	if clt == nil {
		clt = &clientv1.DefaultClient
	}
	if cfg.CentralRsHost == "" {
		cfg.CentralRsHost = DefaultRsHost
	}

	bucketQuery, _ := region.NewBucketRegionsQuery(getUcEndpoint(cfg.UseHTTPS, nil), &region.BucketRegionsQueryOptions{
		UseInsecureProtocol: !cfg.UseHTTPS,
		Client:              clt.Client,
	})

	opts := http_client.Options{
		BasicHTTPClient:     clt.Client,
		BucketQuery:         bucketQuery,
		Credentials:         mac,
		Interceptors:        []clientv2.Interceptor{},
		UseInsecureProtocol: !cfg.UseHTTPS,
		Resolver:            options.Resolver,
		Chooser:             options.Chooser,
		HostRetryConfig:     &clientv2.RetryConfig{RetryMax: options.RetryMax, Retrier: options.Retrier},
		HostsRetryConfig:    &clientv2.RetryConfig{Retrier: options.Retrier},
		HostFreezeDuration:  options.HostFreezeDuration,
		BeforeSign:          options.BeforeSign,
		AfterSign:           options.AfterSign,
		SignError:           options.SignError,
		BeforeResolve:       options.BeforeResolve,
		AfterResolve:        options.AfterResolve,
		ResolveError:        options.ResolveError,
		BeforeBackoff:       options.BeforeBackoff,
		AfterBackoff:        options.AfterBackoff,
		BeforeRequest:       options.BeforeRequest,
		AfterResponse:       options.AfterResponse,
	}
	if region := cfg.GetRegion(); region != nil {
		opts.Regions = region
	}

	return &BucketManager{
		Client:          clt,
		Mac:             mac,
		Cfg:             cfg,
		options:         options,
		apiClient:       apis.NewStorage(&opts),
		downloadManager: downloader.NewDownloadManager(&downloader.DownloadManagerOptions{Options: opts}),
	}
}

// UpdateObjectStatus 用来修改文件状态, 禁用和启用文件的可访问性

// 请求包：
//
// POST /chstatus/<EncodedEntry>/status/<status>
// status：0表示启用，1表示禁用
// 返回包(JSON)：
//
// 200 OK
// 当<EncodedEntryURI>解析失败，返回400 Bad Request {"error":"invalid argument"}
// 当<EncodedEntryURI>不符合UTF-8编码，返回400 Bad Request {"error":"key must be utf8 encoding"}
// 当文件不存在时，返回612 status code 612 {"error":"no such file or directory"}
// 当文件当前状态和设置的状态已经一致，返回400 {"error":"already enabled"}或400 {"error":"already disabled"}
func (m *BucketManager) UpdateObjectStatus(bucketName string, key string, enable bool) error {
	var status int64
	if enable {
		status = 0
	} else {
		status = 1
	}
	_, err := m.apiClient.ModifyObjectStatus(
		context.Background(),
		&apis.ModifyObjectStatusRequest{
			Entry:  bucketName + ":" + key,
			Status: status,
		},
		m.makeRequestOptions(),
	)
	return err
}

// CreateBucket 创建一个七牛存储空间
func (m *BucketManager) CreateBucket(bucketName string, regionID RegionID) error {
	_, err := m.apiClient.CreateBucket(
		context.Background(),
		&apis.CreateBucketRequest{
			Bucket: bucketName,
			Region: string(regionID),
		},
		m.makeRequestOptions(),
	)
	return err
}

// Buckets 用来获取空间列表，如果指定了 shared 参数则额外包含仅被授予了读权限的空间，否则额外包含被授予了读写权限的授权空间
func (m *BucketManager) Buckets(shared bool) ([]string, error) {
	var sharedMode string
	if shared {
		sharedMode = "rd"
	} else {
		sharedMode = "rw"
	}
	response, err := m.apiClient.GetBuckets(
		context.Background(),
		&apis.GetBucketsRequest{
			Shared: sharedMode,
		},
		m.makeRequestOptions(),
	)
	if err != nil {
		return nil, err
	}
	return response.BucketNames, nil
}

// BucketsV4 获取该用户的指定区域内的空间信息，注意该 API 以分页形式返回 Bucket 列表
func (m *BucketManager) BucketsV4(input *BucketV4Input) (output BucketsV4Output, err error) {
	if input == nil {
		input = &BucketV4Input{}
	}
	response, err := m.apiClient.GetBucketsV4(
		context.Background(),
		&apis.GetBucketsV4Request{
			Region: input.Region,
			Limit:  int64(input.Limit),
			Marker: input.Marker,
		},
		m.makeRequestOptions(),
	)
	if err != nil {
		return
	}
	output.IsTruncated = response.IsTruncated
	output.NextMarker = response.NextMarker
	output.Buckets = make([]BucketV4Output, 0, len(response.Buckets))
	for _, bucket := range response.Buckets {
		ctime, err := time.Parse(time.RFC3339, bucket.CreatedTime)
		if err != nil {
			return output, err
		}
		output.Buckets = append(output.Buckets, BucketV4Output{
			Name:    bucket.Name,
			Region:  bucket.Region,
			Private: bucket.Private,
			Ctime:   ctime,
		})
	}
	return
}

// DropBucket 删除七牛存储空间
func (m *BucketManager) DropBucket(bucketName string) error {
	_, err := m.apiClient.DeleteBucket(
		context.Background(),
		&apis.DeleteBucketRequest{
			Bucket: bucketName,
		},
		m.makeRequestOptions(),
	)
	return err
}

// Stat 用来获取一个文件的基本信息
func (m *BucketManager) Stat(bucket, key string) (FileInfo, error) {
	return m.StatWithOpts(bucket, key, nil)
}

type StatOpts struct {
	NeedParts bool
}

// StatWithParts 用来获取一个文件的基本信息以及分片信息
func (m *BucketManager) StatWithOpts(bucket, key string, opt *StatOpts) (FileInfo, error) {
	if opt == nil {
		opt = &StatOpts{}
	}
	response, err := m.apiClient.StatObject(
		context.Background(),
		&apis.StatObjectRequest{
			Entry:     bucket + ":" + key,
			NeedParts: opt.NeedParts,
		},
		m.makeRequestOptions(),
	)
	if err != nil {
		return FileInfo{}, err
	}
	return FileInfo{
		Fsize:                   response.Size,
		Hash:                    response.Hash,
		MimeType:                response.MimeType,
		Type:                    int(response.Type),
		PutTime:                 response.PutTime,
		RestoreStatus:           int(response.RestoringStatus),
		Status:                  int(response.Status),
		Md5:                     response.Md5,
		EndUser:                 response.EndUser,
		MetaData:                response.Metadata,
		Expiration:              response.ExpirationTime,
		TransitionToIA:          response.TransitionToIaTime,
		TransitionToArchive:     response.TransitionToArchiveTime,
		TransitionToDeepArchive: response.TransitionToDeepArchiveTime,
		Parts:                   response.Parts,
	}, nil
}

// Delete 用来删除空间中的一个文件
func (m *BucketManager) Delete(bucket, key string) error {
	_, err := m.apiClient.DeleteObject(
		context.Background(),
		&apis.DeleteObjectRequest{
			Entry: bucket + ":" + key,
		},
		m.makeRequestOptions(),
	)
	return err
}

// Copy 用来创建已有空间中的文件的一个新的副本
func (m *BucketManager) Copy(srcBucket, srcKey, destBucket, destKey string, force bool) error {
	_, err := m.apiClient.CopyObject(
		context.Background(),
		&apis.CopyObjectRequest{
			SrcEntry:  srcBucket + ":" + srcKey,
			DestEntry: destBucket + ":" + destKey,
			IsForce:   force,
		},
		m.makeRequestOptions(),
	)
	return err
}

// Move 用来将空间中的一个文件移动到新的空间或者重命名
func (m *BucketManager) Move(srcBucket, srcKey, destBucket, destKey string, force bool) error {
	_, err := m.apiClient.MoveObject(
		context.Background(),
		&apis.MoveObjectRequest{
			SrcEntry:  srcBucket + ":" + srcKey,
			DestEntry: destBucket + ":" + destKey,
			IsForce:   force,
		},
		m.makeRequestOptions(),
	)
	return err
}

// ChangeMime 用来更新文件的MimeType
func (m *BucketManager) ChangeMime(bucket, key, newMime string) error {
	return m.ChangeMimeAndMeta(bucket, key, newMime, nil)
}

// ChangeMeta
//
//	@Description: 用来修改或者增加 metas，metas 只包含需要修改的，未涉及的保存不变
//	@receiver m BucketManager
//	@param bucket 对象所属空间
//	@param key 对象保存的 key
//	@param newMime 新的 Mime
//	@param metas 需要修改的 metas，只包含需要更改的 metas，可增加；
//				 服务接口中 key 必须包含 x-qn-meta- 前缀，SDK 会对 metas 参数中的 key 进行检测;
//				 - key 如果包含了 x-qn-meta- 前缀，则直接使用 key；
//				 - key 如果不包含了 x-qn-meta- 前缀，则内部会为 key 拼接 x-qn-meta- 前缀。
//	@return err 错误信息
func (m *BucketManager) ChangeMeta(bucket, key string, metas map[string]string) (err error) {
	return m.ChangeMimeAndMeta(bucket, key, "", metas)
}

// ChangeMimeAndMeta
//
//	@Description: 用用来更新文件的 MimeType 以及修改或者增加 metas，metas 只包含需要修改的
//	@receiver m BucketManager
//	@param bucket 对象所属空间
//	@param key 对象保存的 key
//	@param newMime 新的 Mime
//	@param metas 需要修改的 metas，只包含需要更改的 metas，可增加；
//				 服务接口中 key 必须包含 x-qn-meta- 前缀，SDK 会对 metas 参数中的 key 进行检测;
//				 - key 如果包含了 x-qn-meta- 前缀，则直接使用 key；
//				 - key 如果不包含了 x-qn-meta- 前缀，则内部会为 key 拼接 x-qn-meta- 前缀。
//	@return err 错误信息
func (m *BucketManager) ChangeMimeAndMeta(bucket, key, newMime string, metas map[string]string) error {
	metaData := make(map[string]string, len(metas))
	for k, v := range normalizeMeta(metas) {
		metaData[k] = v
	}
	_, err := m.apiClient.ModifyObjectMetadata(
		context.Background(),
		&apis.ModifyObjectMetadataRequest{
			Entry:    bucket + ":" + key,
			MimeType: newMime,
			MetaData: metaData,
		},
		m.makeRequestOptions(),
	)
	return err
}

// ChangeType 用来更新文件的存储类型，0 表示普通存储，1 表示低频存储，2 表示归档存储，3 表示深度归档存储，4 表示归档直读存储
func (m *BucketManager) ChangeType(bucket, key string, fileType int) error {
	_, err := m.apiClient.SetObjectFileType(
		context.Background(),
		&apis.SetObjectFileTypeRequest{
			Entry: bucket + ":" + key,
			Type:  int64(fileType),
		},
		m.makeRequestOptions(),
	)
	return err
}

// RestoreAr 解冻归档存储类型的文件，可设置解冻有效期1～7天, 完成解冻任务通常需要1～5分钟
func (m *BucketManager) RestoreAr(bucket, key string, freezeAfterDays int) error {
	_, err := m.apiClient.RestoreArchivedObject(
		context.Background(),
		&apis.RestoreArchivedObjectRequest{
			Entry:           bucket + ":" + key,
			FreezeAfterDays: int64(freezeAfterDays),
		},
		m.makeRequestOptions(),
	)
	return err
}

// DeleteAfterDays 用来更新文件生命周期，如果 days 设置为0，则表示取消文件的定期删除功能，永久存储
func (m *BucketManager) DeleteAfterDays(bucket, key string, days int) error {
	_, err := m.apiClient.DeleteObjectAfterDays(
		context.Background(),
		&apis.DeleteObjectAfterDaysRequest{
			Entry:           bucket + ":" + key,
			DeleteAfterDays: int64(days),
		},
		m.makeRequestOptions(),
	)
	return err
}

// Batch 接口提供了资源管理的批量操作，支持 stat，copy，move，delete，chgm，chtype，deleteAfterDays几个接口
// 没有 bucket 参数，会从 operations 中解析出 bucket
// @param	operations	操作对象列表，操作对象所属的 bucket 可能会不同，但是必须属于同一个区域
func (m *BucketManager) Batch(operations []string) ([]BatchOpRet, error) {
	if len(operations) == 0 {
		return nil, errors.New("operations is empty")
	}

	bucket := ""
	for _, operation := range operations {
		paths := strings.Split(operation, "/")
		if len(paths) < 3 {
			continue
		}

		// 按当前模式，第 3 个 entry 是 bucket 和 key 键值对
		if b, _, err := decodedEntry(paths[2]); err != nil {
			continue
		} else {
			bucket = b
			break
		}
	}
	if len(bucket) == 0 {
		return nil, errors.New("can't get one bucket from operations")
	}

	return m.BatchWithContext(context.Background(), bucket, operations)
}

func convertToBatchOptRet(op batch_ops.OperationResponse) BatchOpRet {
	var ret BatchOpRet
	ret.Code = int(op.Code)
	opData := op.Data
	ret.Data.Fsize = opData.Size
	ret.Data.Hash = opData.Hash
	ret.Data.MimeType = opData.MimeType
	ret.Data.Type = int(opData.Type)
	ret.Data.PutTime = opData.PutTime
	if restoringStatus := int(opData.RestoringStatus); restoringStatus != 0 {
		ret.Data.RestoreStatus = &restoringStatus
	}
	if status := int(opData.Status); status != 0 {
		ret.Data.Status = &status
	}
	ret.Data.Md5 = opData.Md5
	ret.Data.EndUser = opData.EndUser
	if expirationTime := opData.ExpirationTime; expirationTime != 0 {
		ret.Data.Expiration = &expirationTime
	}
	if transitionToIA := opData.TransitionToIaTime; transitionToIA != 0 {
		ret.Data.TransitionToIA = &transitionToIA
	}
	if transitionToArchiveTime := opData.TransitionToArchiveTime; transitionToArchiveTime != 0 {
		ret.Data.TransitionToArchive = &transitionToArchiveTime
	}
	if transitionToDeepArchiveTime := opData.TransitionToDeepArchiveTime; transitionToDeepArchiveTime != 0 {
		ret.Data.TransitionToDeepArchive = &transitionToDeepArchiveTime
	}
	ret.Data.Error = opData.Error
	return ret
}

// BatchWithContext 接口提供了资源管理的批量操作，支持 stat，copy，move，delete，chgm，chtype，deleteAfterDays几个接口
// @param	ctx		context.Context
// @param	bucket	operations 列表中任意一个操作对象所属的 bucket
// @param	operations	操作对象列表，操作对象所属的 bucket 可能会不同，但是必须属于同一个区域
func (m *BucketManager) BatchWithContext(ctx context.Context, bucket string, operations []string) ([]BatchOpRet, error) {
	if len(operations) > 1000 {
		return nil, errors.New("batch operation count exceeds the limit of 1000")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	opts := m.makeRequestOptions()
	opts.OverwrittenBucketName = bucket
	response, err := m.apiClient.BatchOps(
		ctx,
		&apis.BatchOpsRequest{
			Operations: operations,
		},
		opts,
	)
	if err != nil {
		return nil, err
	}
	rets := make([]BatchOpRet, 0, len(response.OperationResponses))
	for _, op := range response.OperationResponses {
		rets = append(rets, convertToBatchOptRet(op))
	}
	return rets, nil
}

// Fetch 根据提供的远程资源链接来抓取一个文件到空间并已指定文件名保存
func (m *BucketManager) Fetch(resURL, bucket, key string) (FetchRet, error) {
	response, err := m.apiClient.FetchObject(
		context.Background(),
		&apis.FetchObjectRequest{
			FromUrl: resURL,
			ToEntry: bucket + ":" + key,
		},
		m.makeRequestOptions(),
	)
	if err != nil {
		return FetchRet{}, err
	}
	return FetchRet{
		Hash:     response.Hash,
		Fsize:    response.Size,
		MimeType: response.MimeType,
		Key:      response.ObjectName,
	}, nil
}

func (m *BucketManager) RsReqHost(bucket string) (reqHost string, err error) {
	var reqErr error

	if m.Cfg.RsHost == "" {
		reqHost, reqErr = m.RsHost(bucket)
		if reqErr != nil {
			err = reqErr
			return
		}
	} else {
		reqHost = m.Cfg.RsHost
	}
	if !strings.HasPrefix(reqHost, "http") {
		reqHost = endpoint(m.Cfg.UseHTTPS, reqHost)
	}
	return
}

func (m *BucketManager) ApiReqHost(bucket string) (reqHost string, err error) {
	var reqErr error

	if m.Cfg.ApiHost == "" {
		reqHost, reqErr = m.ApiHost(bucket)
		if reqErr != nil {
			err = reqErr
			return
		}
	} else {
		reqHost = m.Cfg.ApiHost
	}
	if !strings.HasPrefix(reqHost, "http") {
		reqHost = endpoint(m.Cfg.UseHTTPS, reqHost)
	}
	return
}

func (m *BucketManager) RsfReqHost(bucket string) (reqHost string, err error) {
	var reqErr error

	if m.Cfg.RsfHost == "" {
		reqHost, reqErr = m.RsfHost(bucket)
		if reqErr != nil {
			err = reqErr
			return
		}
	} else {
		reqHost = m.Cfg.RsfHost
	}
	if !strings.HasPrefix(reqHost, "http") {
		reqHost = endpoint(m.Cfg.UseHTTPS, reqHost)
	}
	return
}

func (m *BucketManager) IoReqHost(bucket string) (reqHost string, err error) {
	var reqErr error

	if m.Cfg.IoHost == "" {
		reqHost, reqErr = m.IovipHost(bucket)
		if reqErr != nil {
			err = reqErr
			return
		}
	} else {
		reqHost = m.Cfg.IoHost
	}
	if !strings.HasPrefix(reqHost, "http") {
		reqHost = endpoint(m.Cfg.UseHTTPS, reqHost)
	}
	return
}

// FetchWithoutKey 根据提供的远程资源链接来抓取一个文件到空间并以文件的内容hash作为文件名
func (m *BucketManager) FetchWithoutKey(resURL, bucket string) (fetchRet FetchRet, err error) {
	response, err := m.apiClient.FetchObject(
		context.Background(),
		&apis.FetchObjectRequest{
			FromUrl: resURL,
			ToEntry: bucket,
		},
		m.makeRequestOptions(),
	)
	if err != nil {
		return FetchRet{}, err
	}
	return FetchRet{
		Hash:     response.Hash,
		Fsize:    response.Size,
		MimeType: response.MimeType,
		Key:      response.ObjectName,
	}, nil
}

// DomainInfo 是绑定在存储空间上的域名的具体信息
type DomainInfo struct {
	Domain string `json:"domain"`

	// 存储空间名字
	Tbl string `json:"tbl"`

	// 用户UID
	Owner   int  `json:"uid"`
	Refresh bool `json:"refresh"`
	Ctime   int  `json:"ctime"`
	Utime   int  `json:"utime"`
}

// ListBucketDomains 返回绑定在存储空间中的域名信息
func (m *BucketManager) ListBucketDomains(bucket string) (info []DomainInfo, err error) {
	toDomainInfo := func(info *get_bucket_domains_v3.DomainInfo) DomainInfo {
		return DomainInfo{
			Domain:  info.Domain,
			Tbl:     info.Bucket,
			Owner:   int(info.OwnerId),
			Refresh: info.AutoRefresh,
			Ctime:   int(info.CreatedTime),
			Utime:   int(info.UpdatedTime),
		}
	}
	response, err := m.apiClient.GetBucketDomainsV3(
		context.Background(),
		&apis.GetBucketDomainsV3Request{
			BucketName: bucket,
		},
		m.makeRequestOptions(),
	)
	if err != nil {
		return nil, err
	}
	domainInfos := make([]DomainInfo, 0, len(response.DomainInfos))
	for _, domainInfo := range response.DomainInfos {
		domainInfos = append(domainInfos, toDomainInfo(&domainInfo))
	}
	return domainInfos, nil
}

// Prefetch 用来同步镜像空间的资源和镜像源资源内容
func (m *BucketManager) Prefetch(bucket, key string) error {
	_, err := m.apiClient.PrefetchObject(
		context.Background(),
		&apis.PrefetchObjectRequest{
			Entry: bucket + ":" + key,
		},
		m.makeRequestOptions(),
	)
	return err
}

// SetImage 用来设置空间镜像源
func (m *BucketManager) SetImage(siteURL, bucket string) error {
	_, err := m.apiClient.SetBucketImage(
		context.Background(),
		&apis.SetBucketImageRequest{
			Bucket: bucket,
			Url:    siteURL,
		},
		m.makeRequestOptions(),
	)
	return err
}

// SetImageWithHost 用来设置空间镜像源，额外添加回源Host头部
func (m *BucketManager) SetImageWithHost(siteURL, bucket, host string) error {
	_, err := m.apiClient.SetBucketImage(
		context.Background(),
		&apis.SetBucketImageRequest{
			Bucket: bucket,
			Url:    siteURL,
			Host:   host,
		},
		m.makeRequestOptions(),
	)
	return err
}

// UnsetImage 用来取消空间镜像源设置
func (m *BucketManager) UnsetImage(bucket string) error {
	_, err := m.apiClient.UnsetBucketImage(
		context.Background(),
		&apis.UnsetBucketImageRequest{
			Bucket: bucket,
		},
		m.makeRequestOptions(),
	)
	return err
}

type AsyncFetchParam struct {
	Url              string `json:"url"`
	Host             string `json:"host,omitempty"`
	Bucket           string `json:"bucket"`
	Key              string `json:"key,omitempty"`
	Md5              string `json:"md5,omitempty"`
	Etag             string `json:"etag,omitempty"`
	CallbackURL      string `json:"callbackurl,omitempty"`
	CallbackBody     string `json:"callbackbody,omitempty"`
	CallbackBodyType string `json:"callbackbodytype,omitempty"`
	FileType         int    `json:"file_type,omitempty"`
}

type AsyncFetchRet struct {
	Id   string `json:"id"`
	Wait int    `json:"wait"`
}

func (m *BucketManager) AsyncFetch(param AsyncFetchParam) (ret AsyncFetchRet, err error) {
	response, err := m.apiClient.AsyncFetchObject(
		context.Background(),
		&apis.AsyncFetchObjectRequest{
			Url:              param.Url,
			Bucket:           param.Bucket,
			Host:             param.Host,
			Key:              param.Key,
			Md5:              param.Md5,
			Etag:             param.Etag,
			CallbackUrl:      param.CallbackURL,
			CallbackBody:     param.CallbackBody,
			CallbackBodyType: param.CallbackBodyType,
			FileType:         int64(param.FileType),
		},
		m.makeRequestOptions(),
	)
	if err != nil {
		return AsyncFetchRet{}, err
	}
	return AsyncFetchRet{
		Id:   response.Id,
		Wait: int(response.QueuedTasksCount),
	}, nil
}

func (m *BucketManager) RsHost(bucket string) (rsHost string, err error) {
	zone, err := m.Zone(bucket)
	if err != nil {
		return
	}

	if len(zone.RsHost) == 0 {
		return "", errors.New("can't get region rs host with bucket:" + bucket)
	}

	rsHost = zone.GetRsHost(m.Cfg.UseHTTPS)
	return
}

func (m *BucketManager) RsfHost(bucket string) (rsfHost string, err error) {
	zone, err := m.Zone(bucket)
	if err != nil {
		return
	}

	if len(zone.RsfHost) == 0 {
		return "", errors.New("can't get region rsf host with bucket:" + bucket)
	}

	rsfHost = zone.GetRsfHost(m.Cfg.UseHTTPS)
	return
}

func (m *BucketManager) IovipHost(bucket string) (iovipHost string, err error) {
	zone, err := m.Zone(bucket)
	if err != nil {
		return
	}

	if len(zone.IovipHost) == 0 {
		return "", errors.New("can't get region io host with bucket:" + bucket)
	}

	iovipHost = zone.GetIoHost(m.Cfg.UseHTTPS)
	return
}

func (m *BucketManager) ApiHost(bucket string) (apiHost string, err error) {
	zone, err := m.Zone(bucket)
	if err != nil {
		return
	}

	if len(zone.ApiHost) == 0 {
		return "", errors.New("can't get region api host with bucket:" + bucket)
	}

	apiHost = zone.GetApiHost(m.Cfg.UseHTTPS)
	return
}

func (m *BucketManager) Zone(bucket string) (z *Zone, err error) {

	z = m.Cfg.GetRegion()
	if z != nil {
		return
	}

	z, err = GetZone(m.Mac.AccessKey, bucket)
	return
}

func (m *BucketManager) makeRequestOptions() *apis.Options {
	return &apis.Options{OverwrittenBucketHosts: getUcEndpointProvider(m.Cfg.UseHTTPS, nil)}
}

// 构建op的方法，导出的方法支持在Batch操作中使用

// URIStat 构建 stat 接口的请求命令
func URIStat(bucket, key string) string {
	return fmt.Sprintf("/stat/%s", EncodedEntry(bucket, key))
}

// URIDelete 构建 delete 接口的请求命令
func URIDelete(bucket, key string) string {
	return fmt.Sprintf("/delete/%s", EncodedEntry(bucket, key))
}

// URICopy 构建 copy 接口的请求命令
func URICopy(srcBucket, srcKey, destBucket, destKey string, force bool) string {
	return fmt.Sprintf("/copy/%s/%s/force/%v", EncodedEntry(srcBucket, srcKey),
		EncodedEntry(destBucket, destKey), force)
}

// URIMove 构建 move 接口的请求命令
func URIMove(srcBucket, srcKey, destBucket, destKey string, force bool) string {
	return fmt.Sprintf("/move/%s/%s/force/%v", EncodedEntry(srcBucket, srcKey),
		EncodedEntry(destBucket, destKey), force)
}

// URIDeleteAfterDays 构建 deleteAfterDays 接口的请求命令
func URIDeleteAfterDays(bucket, key string, days int) string {
	return fmt.Sprintf("/deleteAfterDays/%s/%d", EncodedEntry(bucket, key), days)
}

// URIChangeMime 构建 chgm 接口的请求命令
func URIChangeMime(bucket, key, newMime string) string {
	return fmt.Sprintf("/chgm/%s/mime/%s", EncodedEntry(bucket, key),
		base64.URLEncoding.EncodeToString([]byte(newMime)))
}

// URIChangeMeta
//
//	@Description: 构建 chgm 接口的请求命令，修改 meta
//	@param bucket 空间
//	@param key 文件保存的 key
//	@param metas 需要修改的 metas，只包含需要更改的 metas，可增加
//				 服务接口中 key 必须包含 x-qn-meta- 前缀，SDK 会对 metas 中的 key 进行检测
//				 - key 如果包含了 x-qn-meta- 前缀，则直接使用 key
//				 - key 如果不包含了 x-qn-meta- 前缀，则内部会为 key 拼接 x-qn-meta- 前缀
//	@return string URI
func URIChangeMeta(bucket, key string, changeMetas map[string]string) string {
	return URIChangeMimeAndMeta(bucket, key, "", changeMetas)
}

// URIChangeMimeAndMeta
//
//	@Description: 构建 chgm 接口的请求命令
//	@param bucket 空间
//	@param key 文件保存的 key
//	@param newMime 新的 mime
//	@param metas 需要修改的 metas，只包含需要更改的 metas，可增加
//				 服务接口中 key 必须包含 x-qn-meta- 前缀，SDK 会对 metas 中的 key 进行检测
//				 - key 如果包含了 x-qn-meta- 前缀，则直接使用 key
//				 - key 如果不包含了 x-qn-meta- 前缀，则内部会为 key 拼接 x-qn-meta- 前缀
//	@return string URI
func URIChangeMimeAndMeta(bucket, key, newMime string, metas map[string]string) string {
	uri := fmt.Sprintf("/chgm/%s", EncodedEntry(bucket, key))
	if len(newMime) > 0 {
		uri = fmt.Sprintf("%s/mime/%s", uri, base64.URLEncoding.EncodeToString([]byte(newMime)))
	}
	if len(metas) > 0 {
		for k, v := range normalizeMeta(metas) {
			uri = fmt.Sprintf("%s/%s/%s", uri, k, base64.URLEncoding.EncodeToString([]byte(v)))
		}
	}
	return uri
}

func normalizeMeta(metas map[string]string) map[string]string {
	newMetas := make(map[string]string, len(metas))
	for k, v := range metas {
		if !strings.HasPrefix(k, "x-qn-meta-") {
			k = "x-qn-meta-" + k
		}
		newMetas[k] = v
	}
	return newMetas
}

// URIChangeType 构建 chtype 接口的请求命令
func URIChangeType(bucket, key string, fileType int) string {
	return fmt.Sprintf("/chtype/%s/type/%d", EncodedEntry(bucket, key), fileType)
}

// URIRestoreAr 构建 restoreAr 接口的请求命令
func URIRestoreAr(bucket, key string, afterDay int) string {
	return fmt.Sprintf("/restoreAr/%s/freezeAfterDays/%d", EncodedEntry(bucket, key), afterDay)
}

// EncodedEntry 生成URL Safe Base64编码的 Entry
func EncodedEntry(bucket, key string) string {
	entry := fmt.Sprintf("%s:%s", bucket, key)
	return base64.URLEncoding.EncodeToString([]byte(entry))
}

func decodedEntry(entry string) (bucket, key string, err error) {
	value, dErr := base64.URLEncoding.DecodeString(entry)
	if dErr != nil {
		return "", "", dErr
	}
	bk := strings.Split(string(value), ":")
	if len(bk) == 0 {
		return "", "", errors.New("entry format error")
	}
	if len(bk) == 1 {
		return bk[0], "", nil
	}
	return bk[0], bk[1], nil
}

// EncodedEntryWithoutKey 生成 key 为null的情况下 URL Safe Base64编码的Entry
func EncodedEntryWithoutKey(bucket string) string {
	return base64.URLEncoding.EncodeToString([]byte(bucket))
}

// MakePublicURL 用来生成公开空间资源下载链接，注意该方法并不会对 key 进行 escape
func MakePublicURL(domain, key string) (finalUrl string) {
	domain = strings.TrimRight(domain, "/")
	srcUrl := fmt.Sprintf("%s/%s", domain, key)
	srcUri, _ := url.Parse(srcUrl)
	finalUrl = srcUri.String()
	return
}

// MakePublicURLv2 用来生成公开空间资源下载链接，并且该方法确保 key 将会被 escape
func MakePublicURLv2(domain, key string) string {
	return MakePublicURLv2WithQuery(domain, key, nil)
}

// MakePublicURLv2WithQuery 用来生成公开空间资源下载链接，并且该方法确保 key 将会被 escape，并在 URL 后追加经过编码的查询参数
func MakePublicURLv2WithQuery(domain, key string, query url.Values) string {
	var rawQuery string
	if query != nil {
		rawQuery = query.Encode()
	}
	return makePublicURLv2WithRawQuery(domain, key, rawQuery)
}

// MakePublicURLv2WithQueryString 用来生成公开空间资源下载链接，并且该方法确保 key 将会被 escape，并在 URL 后直接追加查询参数
func MakePublicURLv2WithQueryString(domain, key, query string) string {
	return makePublicURLv2WithRawQuery(domain, key, urlEncodeQuery(query))
}

func makePublicURLv2WithRawQuery(domain, key, rawQuery string) string {
	domain = strings.TrimRight(domain, "/")
	srcUrl := fmt.Sprintf("%s/%s", domain, urlEncodeQuery(key))
	if rawQuery != "" {
		srcUrl += "?" + rawQuery
	}
	srcUri, _ := url.Parse(srcUrl)
	return srcUri.String()
}

// MakePrivateURL 用来生成私有空间资源下载链接，注意该方法并不会对 key 进行 escape
func MakePrivateURL(mac *auth.Credentials, domain, key string, deadline int64) (privateURL string) {
	publicURL := MakePublicURL(domain, key)
	urlToSign := publicURL
	if strings.Contains(publicURL, "?") {
		urlToSign = fmt.Sprintf("%s&e=%d", urlToSign, deadline)
	} else {
		urlToSign = fmt.Sprintf("%s?e=%d", urlToSign, deadline)
	}
	token := mac.Sign([]byte(urlToSign))
	privateURL = fmt.Sprintf("%s&token=%s", urlToSign, token)
	return
}

// MakePrivateURLv2 用来生成私有空间资源下载链接，并且该方法确保 key 将会被 escape
func MakePrivateURLv2(mac *auth.Credentials, domain, key string, deadline int64) (privateURL string) {
	return MakePrivateURLv2WithQuery(mac, domain, key, nil, deadline)
}

// MakePrivateURLv2WithQuery 用来生成私有空间资源下载链接，并且该方法确保 key 将会被 escape，并在 URL 后追加经过编码的查询参数
func MakePrivateURLv2WithQuery(mac *auth.Credentials, domain, key string, query url.Values, deadline int64) (privateURL string) {
	var rawQuery string
	if query != nil {
		rawQuery = query.Encode()
	}
	return makePrivateURLv2WithRawQuery(mac, domain, key, rawQuery, deadline)
}

// MakePrivateURLv2WithQueryString 用来生成私有空间资源下载链接，并且该方法确保 key 将会被 escape，并在 URL 后直接追加查询参数
func MakePrivateURLv2WithQueryString(mac *auth.Credentials, domain, key, query string, deadline int64) (privateURL string) {
	return makePrivateURLv2WithRawQuery(mac, domain, key, urlEncodeQuery(query), deadline)
}

func makePrivateURLv2WithRawQuery(mac *auth.Credentials, domain, key, rawQuery string, deadline int64) (privateURL string) {
	publicURL := makePublicURLv2WithRawQuery(domain, key, rawQuery)
	urlToSign := publicURL
	if strings.Contains(publicURL, "?") {
		urlToSign = fmt.Sprintf("%s&e=%d", urlToSign, deadline)
	} else {
		urlToSign = fmt.Sprintf("%s?e=%d", urlToSign, deadline)
	}
	token := mac.Sign([]byte(urlToSign))
	privateURL = fmt.Sprintf("%s&token=%s", urlToSign, token)
	return
}

func urlEncodeQuery(str string) (ret string) {
	str = url.QueryEscape(str)
	str = strings.Replace(str, "%2F", "/", -1)
	str = strings.Replace(str, "%7C", "|", -1)
	str = strings.Replace(str, "+", "%20", -1)
	return str
}
