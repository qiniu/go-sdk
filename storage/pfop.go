package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/client"
	"github.com/qiniu/go-sdk/v7/internal/clientv2"
	"github.com/qiniu/go-sdk/v7/media/apis"
	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/region"
)

// OperationManager 提供了数据处理相关的方法
type OperationManager struct {
	Client    *client.Client
	Mac       *auth.Credentials
	Cfg       *Config
	apiClient *apis.Media
}

// NewOperationManager 用来构建一个新的数据处理对象
func NewOperationManager(mac *auth.Credentials, cfg *Config) *OperationManager {
	return NewOperationManagerEx(mac, cfg, nil)
}

// NewOperationManager 用来构建一个新的数据处理对象
func NewOperationManagerEx(mac *auth.Credentials, cfg *Config, clt *client.Client) *OperationManager {
	if cfg == nil {
		cfg = NewConfig()
	}
	if mac == nil {
		mac = auth.Default()
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
		Credentials:         mac,
		Interceptors:        []clientv2.Interceptor{},
		UseInsecureProtocol: !cfg.UseHTTPS,
	}
	if region := cfg.GetRegion(); region != nil {
		opts.Regions = region
	}

	return &OperationManager{
		Client:    clt,
		Mac:       mac,
		Cfg:       cfg,
		apiClient: apis.NewMedia(&opts),
	}
}

// PfopRet 为数据处理请求的回复内容
type PfopRet struct {
	PersistentID string `json:"persistentId,omitempty"`
}

// PrefopRet 为数据处理请求的状态查询回复内容
type PrefopRet struct {
	ID          string    `json:"id"`
	Code        int       `json:"code"`
	Desc        string    `json:"desc"`
	Type        int64     `json:"type,omitempty"`
	InputBucket string    `json:"inputBucket,omitempty"`
	InputKey    string    `json:"inputKey,omitempty"`
	Pipeline    string    `json:"pipeline,omitempty"`
	TaskFrom    string    `json:"taskFrom,omitempty"`
	Reqid       string    `json:"reqid,omitempty"`
	CreatedAt   time.Time `json:"creationDate,omitempty"`
	Items       []FopResult
}

func (r *PrefopRet) String() string {
	strData := fmt.Sprintf("Id: %s\r\nCode: %d\r\nDesc: %s\r\n", r.ID, r.Code, r.Desc)
	if r.InputBucket != "" {
		strData += fmt.Sprintf("InputBucket: %s\n", r.InputBucket)
	}
	if r.InputKey != "" {
		strData += fmt.Sprintf("InputKey: %s\n", r.InputKey)
	}
	if r.Pipeline != "" {
		strData += fmt.Sprintf("Pipeline: %s\n", r.Pipeline)
	}
	if r.TaskFrom != "" {
		strData += fmt.Sprintf("TaskFrom: %s\n", r.TaskFrom)
	}
	if r.Type != 0 {
		strData += fmt.Sprintf("Type: %d\n", r.Type)
	}
	if r.Reqid != "" {
		strData += fmt.Sprintf("Reqid: %s\n", r.Reqid)
	}
	if !r.CreatedAt.IsZero() {
		strData += fmt.Sprintf("CreatedAt: %s\n", r.CreatedAt)
	}

	strData = fmt.Sprintln(strData)
	for _, item := range r.Items {
		strData += fmt.Sprintf("\tCmd:\t%s\r\n\tCode:\t%d\r\n\tDesc:\t%s\r\n", item.Cmd, item.Code, item.Desc)
		if item.Error != "" {
			strData += fmt.Sprintf("\tError:\t%s\r\n", item.Error)
		} else {
			if item.Hash != "" {
				strData += fmt.Sprintf("\tHash:\t%s\r\n", item.Hash)
			}
			if item.Key != "" {
				strData += fmt.Sprintf("\tKey:\t%s\r\n", item.Key)
			}
			if item.Keys != nil {
				if len(item.Keys) > 0 {
					strData += "\tKeys: {\r\n"
					for _, key := range item.Keys {
						strData += fmt.Sprintf("\t\t%s\r\n", key)
					}
					strData += "\t}\r\n"
				}
			}
		}
		strData += "\r\n"
	}
	return strData
}

type PfopRequest struct {
	BucketName         string // 空间名称
	ObjectName         string // 对象名称
	Fops               string // 数据处理命令列表，以 `;` 分隔，可以指定多个数据处理命令，与 `workflowTemplateID` 二选一
	NotifyUrl          string // 处理结果通知接收 URL
	Force              int64  // 强制执行数据处理，设为 `1`，则可强制执行数据处理并覆盖原结果
	Type               int64  // 任务类型，支持 `0` 表示普通任务，`1` 表示闲时任务
	Pipeline           string // 对列名，仅适用于普通任务
	WorkflowTemplateID string // 工作流模板 ID
}

// FopResult 云处理操作列表，包含每个云处理操作的状态信息
type FopResult struct {
	Cmd   string   `json:"cmd"`
	Code  int      `json:"code"`
	Desc  string   `json:"desc"`
	Error string   `json:"error,omitempty"`
	Hash  string   `json:"hash,omitempty"`
	Key   string   `json:"key,omitempty"`
	Keys  []string `json:"keys,omitempty"`
}

// Pfop 持久化数据处理
//
//	bucket		资源空间
//	key   		源资源名
//	fops		云处理操作列表，
//	notifyURL	处理结果通知接收URL
//	pipeline	多媒体处理队列名称
//	force		强制执行数据处理
func (m *OperationManager) Pfop(bucket, key, fops, pipeline, notifyURL string,
	force bool) (string, error) {
	var forceNumber int64
	if force {
		forceNumber = 1
	}
	response, err := m.apiClient.Pfop(context.Background(), &apis.PfopRequest{
		BucketName: bucket,
		ObjectName: key,
		Fops:       fops,
		NotifyUrl:  notifyURL,
		Force:      forceNumber,
		Pipeline:   pipeline,
	}, m.makeRequestOptions())
	if err != nil {
		return "", err
	}
	return response.PersistentId, nil
}

// Pfop 持久化数据处理 v2
func (m *OperationManager) PfopV2(ctx context.Context, pfopRequest *PfopRequest) (*PfopRet, error) {
	response, err := m.apiClient.Pfop(context.Background(), &apis.PfopRequest{
		BucketName:         pfopRequest.BucketName,
		ObjectName:         pfopRequest.ObjectName,
		Fops:               pfopRequest.Fops,
		NotifyUrl:          pfopRequest.NotifyUrl,
		Force:              pfopRequest.Force,
		Type:               pfopRequest.Type,
		Pipeline:           pfopRequest.Pipeline,
		WorkflowTemplateId: pfopRequest.WorkflowTemplateID,
	}, m.makeRequestOptions())
	if err != nil {
		return nil, err
	}
	return &PfopRet{PersistentID: response.PersistentId}, nil
}

// Prefop 持久化处理状态查询
func (m *OperationManager) Prefop(persistentID string) (PrefopRet, error) {
	response, err := m.apiClient.Prefop(context.Background(), &apis.PrefopRequest{
		PersistentId: persistentID,
	}, m.makeRequestOptions())
	if err != nil {
		return PrefopRet{}, err
	}
	createdAt, err := time.Parse(time.RFC3339, response.CreatedAt)
	if err != nil {
		return PrefopRet{}, err
	}
	ret := PrefopRet{
		ID:          response.PersistentId,
		Code:        int(response.Code),
		Desc:        response.Description,
		Type:        response.Type,
		InputBucket: response.BucketName,
		InputKey:    response.ObjectName,
		Pipeline:    response.Pipeline,
		TaskFrom:    response.TaskFrom,
		Reqid:       response.RequestId,
		CreatedAt:   createdAt,
		Items:       make([]FopResult, 0, len(response.Items)),
	}
	for _, item := range response.Items {
		ret.Items = append(ret.Items, FopResult{
			Cmd:   item.Command,
			Code:  int(item.Code),
			Desc:  item.Description,
			Error: item.Error,
			Hash:  item.Hash,
			Key:   item.ObjectName,
		})
	}
	return ret, nil
}

// Deprecated
func (m *OperationManager) ApiHost(bucket string) (apiHost string, err error) {
	var zone *Zone
	if m.Cfg.Zone != nil {
		zone = m.Cfg.Zone
	} else {
		if v, zoneErr := GetZone(m.Mac.AccessKey, bucket); zoneErr != nil {
			err = zoneErr
			return
		} else {
			zone = v
		}
	}

	if zone == nil || len(zone.ApiHost) == 0 {
		return "", errors.New("can't get region api host with bucket:" + bucket)
	}

	apiHost = endpoint(m.Cfg.UseHTTPS, zone.ApiHost)
	return
}

// Deprecated
func (m *OperationManager) PrefopApiHost(persistentID string) (apiHost string) {
	apiHost = "api.qiniu.com"
	if m.Cfg.Zone != nil {
		apiHost = m.Cfg.Zone.ApiHost
	}
	apiHost = endpoint(m.Cfg.UseHTTPS, apiHost)
	return
}

func (m *OperationManager) makeRequestOptions() *apis.Options {
	return &apis.Options{OverwrittenBucketHosts: getUcEndpointProvider(m.Cfg.UseHTTPS, nil)}
}
