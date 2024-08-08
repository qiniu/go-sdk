package cdn

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/qiniu/go-sdk/v7/auth"
)

// Fusion CDN服务域名
var (
	FusionHost = "http://fusion.qiniuapi.com"
)

// CdnManager 提供了文件和目录刷新，文件预取，获取域名带宽和流量数据，获取域名日志列表等功能
type CdnManager struct {
	mac *auth.Credentials
}

// NewCdnManager 用来构建一个新的 CdnManager
func NewCdnManager(mac *auth.Credentials) *CdnManager {
	if mac == nil {
		mac = auth.Default()
	}
	return &CdnManager{mac: mac}
}

// TrafficReq 为批量查询带宽/流量的API请求内容
//
//	StartDate 	开始日期，格式例如：2016-07-01
//	EndDate 	结束日期，格式例如：2016-07-03
//	Granularity	取值粒度，取值可选值：5min/hour/day
//	Domains 	域名列表，彼此用 ; 连接
//	DataType        计量数据类型, 可选 'bandwidth'(静态cdn带宽,默认)..., 参考 [DataType]
type TrafficReq struct {
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
	Granularity string `json:"granularity"`
	Domains     string `json:"domains"`
	DataType    string `json:"type,omitempty"`
}

// TrafficResp 为带宽/流量查询响应内容
type TrafficResp struct {
	Code  int                    `json:"code"`
	Error string                 `json:"error"`
	Time  []string               `json:"time,omitempty"`
	Data  map[string]TrafficData `json:"data,omitempty"`
}

// TrafficData 为带宽/流量数据
type TrafficData struct {
	DomainChina   []int `json:"china"`
	DomainOversea []int `json:"oversea"`
}

type options struct {
	dataType DataType
}

func _WithDataType(dataType DataType) Option {
	return OptionFunc(func(opt interface{}) {
		opt.(*options).dataType = dataType
	})
}

type BandwidthOption Option

func WithBandwidthDataType(dataType DataType) BandwidthOption {
	if DataTypeBandwidth <= dataType && dataType <= DataType302mBandwidth {
		return _WithDataType(dataType)
	}
	panic("cdn: invalid DataType for GetBandwidthData: " + dataType.String())
}

// GetBandwidthData 方法用来获取域名访问带宽数据
//
//	StartDate	string		必须	开始日期，例如：2016-07-01
//	EndDate		string		必须	结束日期，例如：2016-07-03
//	Granularity	string		必须	粒度，取值：5min ／ hour ／day
//	Domains		[]string	必须	域名列表
//	Opts                            非必须   可选项

func (m *CdnManager) GetBandwidthData(startDate, endDate, granularity string,
	domainList []string, opts ...BandwidthOption) (bandwidthData TrafficResp, err error) {
	var options options
	for _, opt := range opts {
		opt.Apply(&options)
	}
	domains := strings.Join(domainList, ";")
	reqBody := TrafficReq{
		StartDate:   startDate,
		EndDate:     endDate,
		Granularity: granularity,
		Domains:     domains,
	}
	if options.dataType.Valid() {
		reqBody.DataType = options.dataType.String()
	}

	resData, reqErr := postRequest(m.mac, "/v2/tune/bandwidth", reqBody)
	if reqErr != nil {
		err = reqErr
		return
	}
	umErr := json.Unmarshal(resData, &bandwidthData)
	if umErr != nil {
		err = umErr
		return
	}
	return
}

type FluxOption Option

func WithFluxDataType(dataType DataType) FluxOption {
	if DataTypeFlow <= dataType && dataType <= DataType302mFlow {
		return _WithDataType(dataType)
	}
	panic("cdn: invalid DataType for GetFluxData: " + dataType.String())
}

// GetFluxData 方法用来获取域名访问流量数据
//
//	StartDate	string		必须	开始日期，例如：2016-07-01
//	EndDate		string		必须	结束日期，例如：2016-07-03
//	Granularity	string		必须	粒度，取值：5min ／ hour ／day
//	Domains		[]string	必须	域名列表
//	Opts                            非必须   可选项
func (m *CdnManager) GetFluxData(startDate, endDate, granularity string,
	domainList []string, opts ...FluxOption) (fluxData TrafficResp, err error) {
	var options options
	for _, opt := range opts {
		opt.Apply(&options)
	}
	domains := strings.Join(domainList, ";")
	reqBody := TrafficReq{
		StartDate:   startDate,
		EndDate:     endDate,
		Granularity: granularity,
		Domains:     domains,
	}
	if options.dataType.Valid() {
		reqBody.DataType = options.dataType.String()
	}

	resData, reqErr := postRequest(m.mac, "/v2/tune/flux", reqBody)
	if reqErr != nil {
		err = reqErr
		return
	}

	umErr := json.Unmarshal(resData, &fluxData)
	if umErr != nil {
		err = umErr
		return
	}

	return
}

// RefreshReq 为缓存刷新请求内容
type RefreshReq struct {
	Urls []string `json:"urls"`
	Dirs []string `json:"dirs"`
}

// RefreshResp 缓存刷新响应内容
type RefreshResp struct {
	Code          int      `json:"code"`
	Error         string   `json:"error"`
	RequestID     string   `json:"requestId,omitempty"`
	InvalidUrls   []string `json:"invalidUrls,omitempty"`
	InvalidDirs   []string `json:"invalidDirs,omitempty"`
	URLQuotaDay   int      `json:"urlQuotaDay,omitempty"`
	URLSurplusDay int      `json:"urlSurplusDay,omitempty"`
	DirQuotaDay   int      `json:"dirQuotaDay,omitempty"`
	DirSurplusDay int      `json:"dirSurplusDay,omitempty"`
}

// RefreshUrlsAndDirs 方法用来刷新文件或目录
// urls	要刷新的单个url列表，单次方法调用总数不超过100条；单个url，即一个具体的url，
// 例如：http://bar.foo.com/index.html
// dirs	要刷新的目录url列表，单次方法调用总数不超过10条；目录dir，即表示一个目录级的url，
// 例如：http://bar.foo.com/dir/，
func (m *CdnManager) RefreshUrlsAndDirs(urls, dirs []string) (result RefreshResp, err error) {
	if len(urls) > 100 {
		err = errors.New("urls count exceeds the limit of 100")
		return
	}
	if len(dirs) > 10 {
		err = errors.New("dirs count exceeds the limit of 10")
		return
	}

	reqBody := RefreshReq{
		Urls: urls,
		Dirs: dirs,
	}

	resData, reqErr := postRequest(m.mac, "/v2/tune/refresh", reqBody)
	if reqErr != nil {
		err = reqErr
		return
	}
	umErr := json.Unmarshal(resData, &result)
	if umErr != nil {
		err = reqErr
		return
	}

	return
}

// RefreshUrls 刷新文件
func (m *CdnManager) RefreshUrls(urls []string) (result RefreshResp, err error) {
	return m.RefreshUrlsAndDirs(urls, nil)
}

// RefreshDirs 刷新目录
func (m *CdnManager) RefreshDirs(dirs []string) (result RefreshResp, err error) {
	return m.RefreshUrlsAndDirs(nil, dirs)
}

// PrefetchReq 文件预取请求内容
type PrefetchReq struct {
	Urls []string `json:"urls"`
}

// PrefetchResp 文件预取响应内容
type PrefetchResp struct {
	Code        int      `json:"code"`
	Error       string   `json:"error"`
	RequestID   string   `json:"requestId,omitempty"`
	InvalidUrls []string `json:"invalidUrls,omitempty"`
	QuotaDay    int      `json:"quotaDay,omitempty"`
	SurplusDay  int      `json:"surplusDay,omitempty"`
}

// PrefetchUrls 预取文件链接，每次最多不可以超过100条
func (m *CdnManager) PrefetchUrls(urls []string) (result PrefetchResp, err error) {
	if len(urls) > 100 {
		err = errors.New("urls count exceeds the limit of 100")
		return
	}

	reqBody := PrefetchReq{
		Urls: urls,
	}

	resData, reqErr := postRequest(m.mac, "/v2/tune/prefetch", reqBody)
	if reqErr != nil {
		err = reqErr
		return
	}

	umErr := json.Unmarshal(resData, &result)
	if umErr != nil {
		err = umErr
		return
	}

	return
}

// ListLogRequest 日志下载请求内容
type ListLogRequest struct {
	Day     string `json:"day"`
	Domains string `json:"domains"`
}

// ListLogResult 日志下载相应内容
type ListLogResult struct {
	Code  int                        `json:"code"`
	Error string                     `json:"error"`
	Data  map[string][]LogDomainInfo `json:"data"`
}

// LogDomainInfo 日志下载信息
type LogDomainInfo struct {
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	ModifiedTime int64  `json:"mtime"`
	URL          string `json:"url"`
}

// GetCdnLogList 获取CDN域名访问日志的下载链接
func (m *CdnManager) GetCdnLogList(day string, domains []string) (
	listLogResult ListLogResult, err error) {
	//new log query request
	logReq := ListLogRequest{
		Day:     day,
		Domains: strings.Join(domains, ";"),
	}

	resData, reqErr := postRequest(m.mac, "/v2/tune/log/list", logReq)
	if reqErr != nil {
		err = fmt.Errorf("get response error, %s", reqErr)
		return
	}

	if decodeErr := json.Unmarshal(resData, &listLogResult); decodeErr != nil {
		err = fmt.Errorf("get response error, %s", decodeErr)
		return
	}

	if listLogResult.Error != "" {
		err = fmt.Errorf("get log list error, %d %s", listLogResult.Code, listLogResult.Error)
		return
	}

	return
}

// RequestWithBody 带body对api发出请求并且返回response body
func postRequest(mac *auth.Credentials, path string, body interface{}) (resData []byte,
	err error) {
	urlStr := fmt.Sprintf("%s%s", FusionHost, path)
	reqData, _ := json.Marshal(body)
	req, reqErr := http.NewRequest("POST", urlStr, bytes.NewReader(reqData))
	if reqErr != nil {
		err = reqErr
		return
	}

	accessToken, signErr := mac.SignRequest(req)
	if signErr != nil {
		err = signErr
		return
	}

	req.Header.Add("Authorization", "QBox "+accessToken)
	req.Header.Add("Content-Type", "application/json")

	resp, respErr := http.DefaultClient.Do(req)
	if respErr != nil {
		err = respErr
		return
	}
	defer resp.Body.Close()

	resData, ioErr := ioutil.ReadAll(resp.Body)
	if ioErr != nil {
		err = ioErr
		return
	}

	return
}

// CertListReq 获取ssl证书列表请求内容
type CertListReq struct {
	Marker string `json:"marker"`
	Limit  int    `json:"limit"`
}

// CertListResp 获取ssl证书列表响应内容
type CertListResp struct {
	Marker string `json:"marker"`
	Certs  []struct {
		CertID     string   `json:"certid"`
		Name       string   `json:"name"`
		CommonName string   `json:"common_name"`
		DNSNames   []string `json:"dnsnames"`
		NotBefore  int      `json:"not_before"`
		NotAfter   int      `json:"not_after"`
		CreateTime int      `json:"create_time"`
	} `json:"certs"`
}

// GetCertList 获取ssl证书列表
func (m *CdnManager) GetCertList(marker string, limit int) (certList CertListResp, err error) {
	reqParams := fmt.Sprintf("marker=%s&limit=%d", marker, limit)
	urlStr := fmt.Sprintf("%s/sslcert?%s", FusionHost, reqParams)
	req, reqErr := http.NewRequest("GET", urlStr, nil)
	if reqErr != nil {
		err = reqErr
		return
	}
	accessToken, signErr := m.mac.SignRequest(req)
	if signErr != nil {
		err = signErr
		return
	}
	req.Header.Add("Authorization", "QBox "+accessToken)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, respErr := http.DefaultClient.Do(req)
	if respErr != nil {
		err = respErr
		return
	}
	defer resp.Body.Close()
	resData, ioErr := ioutil.ReadAll(resp.Body)
	if ioErr != nil {
		err = ioErr
		return
	}
	umErr := json.Unmarshal(resData, &certList)
	if umErr != nil {
		err = umErr
		return
	}

	return
}

// CertDetailResp 获取单个ssl证书响应内容
type CertDetailResp struct {
	Name       string   `json:"name"`
	CommonName string   `json:"common_name"`
	DNSNames   []string `json:"dnsnames"`
	NotBefore  int      `json:"not_before"`
	NotAfter   int      `json:"not_after"`
	Pri        string   `json:"pri"`
	Ca         string   `json:"ca"`
	CreateTime int      `json:"create_time"`
}

// RealCertDetailResp 当前的api返回与官方文档有差异
type RealCertDetailResp struct {
	CertID           string   `json:"certid"`
	Name             string   `json:"name"`
	UID              int      `json:"uid"`
	CommonName       string   `json:"common_name"`
	DNSNames         []string `json:"dnsnames"`
	CreateTime       int      `json:"create_time"`
	NotBefore        int      `json:"not_before"`
	NotAfter         int      `json:"not_after"`
	OrderID          string   `json:"orderid"`
	ProductShortName string   `json:"product_short_name"`
	ProductType      string   `json:"product_type"`
	CertType         string   `json:"cert_type"`
	Encrypt          string   `json:"encrypt"`
	EncryptParameter string   `json:"encryptParameter"`
	Enable           bool     `json:"enable"`
	ChildOrderID     string   `json:"child_order_id"`
	State            string   `json:"state"`
	AutoRenew        bool     `json:"auto_renew"`
	Renewable        bool     `json:"renewable"`
	CA               string   `json:"ca"`
}

// GetCertDetail 获取单个ssl证书的详细信息
func (m *CdnManager) GetCertDetail(certID string) (certDetail CertDetailResp, err error) {
	urlStr := fmt.Sprintf("%s/sslcert/%s", FusionHost, certID)
	req, reqErr := http.NewRequest("GET", urlStr, nil)
	if reqErr != nil {
		err = reqErr
		return
	}
	accessToken, signErr := m.mac.SignRequest(req)
	if signErr != nil {
		err = signErr
		return
	}
	req.Header.Add("Authorization", "QBox "+accessToken)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, respErr := http.DefaultClient.Do(req)
	if respErr != nil {
		err = respErr
		return
	}
	defer resp.Body.Close()
	resData, ioErr := ioutil.ReadAll(resp.Body)
	if ioErr != nil {
		err = ioErr
		return
	}
	var resJson = struct {
		Code  int
		Error string
		Cert  RealCertDetailResp
	}{}
	umErr := json.Unmarshal(resData, &resJson)
	certDetail.Ca = resJson.Cert.CA
	certDetail.CommonName = resJson.Cert.Name
	certDetail.DNSNames = resJson.Cert.DNSNames
	certDetail.Name = resJson.Cert.Name
	certDetail.NotAfter = resJson.Cert.NotAfter
	certDetail.NotBefore = resJson.Cert.NotBefore
	certDetail.CreateTime = resJson.Cert.CreateTime
	if umErr != nil {
		err = umErr
		return
	}

	return
}

// UploadCertReq 上传ssl证书请求内容
type UploadCertReq struct {
	Name       string `json:"name"`
	CommonName string `json:"common_name"`
	Pri        string `json:"pri"`
	Ca         string `json:"ca"`
}

// UploadCertResp 上传ssl证书响应内容
type UploadCertResp struct {
	CertID string `json:"certID"`
}

// UploadCert 上传ssl证书
func (m *CdnManager) UploadCert(name, commonName, pri, ca string) (resp UploadCertResp, err error) {
	reqBody := UploadCertReq{
		Name:       name,
		CommonName: commonName,
		Pri:        pri,
		Ca:         ca,
	}
	urlStr := fmt.Sprintf("%s/sslcert", FusionHost)
	reqData, _ := json.Marshal(reqBody)
	req, reqErr := http.NewRequest("POST", urlStr, bytes.NewReader(reqData))
	if reqErr != nil {
		err = reqErr
		return
	}
	accessToken, signErr := m.mac.SignRequest(req)
	if signErr != nil {
		err = signErr
		return
	}
	req.Header.Add("Authorization", "QBox "+accessToken)
	req.Header.Add("Content-Type", "application/json")
	httpResp, respErr := http.DefaultClient.Do(req)
	if respErr != nil {
		err = respErr
		return
	}
	defer httpResp.Body.Close()
	resData, ioErr := ioutil.ReadAll(httpResp.Body)
	if ioErr != nil {
		err = ioErr
		return
	}
	umErr := json.Unmarshal(resData, &resp)
	if umErr != nil {
		err = umErr
		return
	}
	return
}

// DeleteCert 删除ssl证书
func (m *CdnManager) DeleteCert(certID string) (err error) {
	urlStr := fmt.Sprintf("%s/sslcert/%s", FusionHost, certID)
	req, reqErr := http.NewRequest("DELETE", urlStr, nil)
	if reqErr != nil {
		err = reqErr
		return
	}
	accessToken, signErr := m.mac.SignRequest(req)
	if signErr != nil {
		err = signErr
		return
	}
	req.Header.Add("Authorization", "QBox "+accessToken)
	req.Header.Add("Content-Type", "application/json")
	resp, respErr := http.DefaultClient.Do(req)
	if respErr != nil {
		err = respErr
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		return
	}
	return
}
