package uplog

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"os"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/qiniu/go-sdk/v7/conf"
	"github.com/qiniu/go-sdk/v7/internal/clientv2"
	internal_io "github.com/qiniu/go-sdk/v7/internal/io"
)

type (
	GetUpToken   func() (string, error)
	RequestUplog struct {
		LogType               LogType    `json:"log_type,omitempty"`
		StatusCode            int        `json:"status_code,omitempty"`
		RequestID             string     `json:"req_id,omitempty"`
		Host                  string     `json:"host,omitempty"`
		Port                  uint64     `json:"port,omitempty"`
		RemoteIP              string     `json:"remote_ip,omitempty"`
		Method                string     `json:"method,omitempty"`
		Path                  string     `json:"path,omitempty"`
		APIName               string     `json:"api_name,omitempty"`
		APIType               APIType    `json:"api_type,omitempty"`
		TotalElapsedTime      uint64     `json:"total_elapsed_time,omitempty"`
		DNSElapsedTime        uint64     `json:"dns_elapsed_time,omitempty"`
		ConnectElapsedTime    uint64     `json:"connect_elapsed_time,omitempty"`
		TLSConnectElapsedTime uint64     `json:"tls_connect_elapsed_time,omitempty"`
		RequestElapsedTime    uint64     `json:"request_elapsed_time,omitempty"`
		WaitElapsedTime       uint64     `json:"wait_elapsed_time,omitempty"`
		ResponseElapsedTime   uint64     `json:"response_elapsed_time,omitempty"`
		BytesTotal            int64      `json:"bytes_total,omitempty"`
		BytesSent             int64      `json:"bytes_sent,omitempty"`
		BytesReceived         int64      `json:"bytes_received,omitempty"`
		TargetBucket          string     `json:"target_bucket,omitempty"`
		TargetKey             string     `json:"target_key,omitempty"`
		ErrorType             ErrorType  `json:"error_type,omitempty"`
		ErrorDescription      string     `json:"error_description,omitempty"`
		OSName                string     `json:"os_name,omitempty"`
		OSVersion             string     `json:"os_version,omitempty"`
		OSArch                string     `json:"os_arch,omitempty"`
		SDKName               string     `json:"sdk_name,omitempty"`
		SDKVersion            string     `json:"sdk_version,omitempty"`
		HTTPClient            string     `json:"http_client,omitempty"`
		HTTPClientVersion     string     `json:"http_client_version,omitempty"`
		UpTime                int64      `json:"up_time,omitempty"`
		HTTPVersion           string     `json:"http_version,omitempty"`
		Pid                   int        `json:"pid,omitempty"`
		PerceptiveSpeed       int64      `json:"perceptive_speed,omitempty"`
		getUpToken            GetUpToken `json:"-"`
	}
)

func NewRequestUplog(apiName, targetBucket, targetKey string, getUpToken GetUpToken) (*RequestUplog, error) {
	return &RequestUplog{
		LogType:           LogTypeRequest,
		APIType:           APITypeKodo,
		OSName:            truncate(runtime.GOOS, maxFieldValueLength),
		OSVersion:         truncate(getOsVersion(), maxFieldValueLength),
		OSArch:            truncate(runtime.GOARCH, maxFieldValueLength),
		SDKName:           "go",
		SDKVersion:        truncate(conf.Version, maxFieldValueLength),
		HTTPClient:        truncate(getHttpClientName(), maxFieldValueLength),
		HTTPClientVersion: truncate(conf.Version, maxFieldValueLength),
		APIName:           truncate(apiName, maxFieldValueLength),
		Pid:               os.Getpid(),
		TargetBucket:      truncate(targetBucket, maxFieldValueLength),
		TargetKey:         truncate(targetKey, maxFieldValueLength),
		getUpToken:        getUpToken,
	}, nil
}

func (uplog *RequestUplog) Priority() clientv2.InterceptorPriority {
	return clientv2.InterceptorPriorityUplog
}

func (uplog *RequestUplog) Intercept(req *http.Request, handler clientv2.Handler) (resp *http.Response, err error) {
	if !IsUplogEnabled() {
		return handler(req)
	}

	var dnsStartTime, gotFirstResponseByteTime, connectStartTime, tlsHandshakeStartTime, wroteHeadersTime, wroteRequestTime time.Time

	uplog.Method = truncate(req.Method, maxFieldValueLength)
	uplog.Path = truncate(req.URL.Path, maxFieldValueLength)
	uplog.Host = truncate(req.URL.Hostname(), maxFieldValueLength)
	if port, err := strconv.ParseUint(req.URL.Port(), 10, 16); err == nil {
		uplog.Port = port
	}
	if req.Body != nil {
		if kl, ok := req.Body.(internal_io.KnownLength); ok {
			if detectedLength, err := kl.DetectLength(); err == nil {
				uplog.BytesTotal = detectedLength
			}
		}
		req.Body = newRequestBodyWithCounter(req.Body, &uplog.BytesSent)
	}
	if req.GetBody != nil {
		getBody := req.GetBody
		req.GetBody = func() (r io.ReadCloser, err error) {
			if r, err = getBody(); err != nil {
				return
			}
			r = newRequestBodyWithCounter(r, &uplog.BytesSent)
			return
		}
	}
	bytesSentBlockLevelTotal := getBytesSentTotalFromContext(req.Context(), true)
	bytesReceivedBlockLevelTotal := getBytesReceivedTotalFromContext(req.Context(), true)
	requestsCountBlockLevel := getRequestsCountFromContext(req.Context(), true)
	bytesSentQualityLevelTotal := getBytesSentTotalFromContext(req.Context(), false)
	bytesReceivedQualityLevelTotal := getBytesReceivedTotalFromContext(req.Context(), false)
	requestsCountQualityLevel := getRequestsCountFromContext(req.Context(), false)
	tracer := httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			if host, port, err := net.SplitHostPort(info.Conn.RemoteAddr().String()); err == nil {
				uplog.RemoteIP = host
				if uplog.Port == 0 {
					if portInt, err := strconv.ParseUint(port, 10, 16); err == nil {
						uplog.Port = portInt
					}
				}
			}
		},
		GotFirstResponseByte: func() {
			if !wroteRequestTime.IsZero() {
				uplog.WaitElapsedTime = getElapsedTime(wroteRequestTime)
			}
			gotFirstResponseByteTime = time.Now()
		},
		DNSStart: func(httptrace.DNSStartInfo) {
			dnsStartTime = time.Now()
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			if !dnsStartTime.IsZero() {
				uplog.DNSElapsedTime = getElapsedTime(dnsStartTime)
			}
		},
		ConnectStart: func(network string, addr string) {
			connectStartTime = time.Now()
		},
		ConnectDone: func(network string, addr string, err error) {
			if !connectStartTime.IsZero() {
				uplog.ConnectElapsedTime = getElapsedTime(connectStartTime)
			}
		},
		TLSHandshakeStart: func() {
			tlsHandshakeStartTime = time.Now()
		},
		TLSHandshakeDone: func(tls.ConnectionState, error) {
			if !tlsHandshakeStartTime.IsZero() {
				uplog.TLSConnectElapsedTime = getElapsedTime(tlsHandshakeStartTime)
			}
		},
		WroteHeaders: func() {
			wroteHeadersTime = time.Now()
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			if !wroteHeadersTime.IsZero() {
				uplog.RequestElapsedTime = getElapsedTime(wroteHeadersTime)
			}
			wroteRequestTime = time.Now()
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), &tracer))
	beginAt := time.Now()
	uplog.UpTime = beginAt.Unix()
	resp, err = handler(req)
	if !gotFirstResponseByteTime.IsZero() {
		uplog.ResponseElapsedTime = getElapsedTime(gotFirstResponseByteTime)
	}
	uplog.TotalElapsedTime = getElapsedTime(beginAt)
	if err != nil {
		uplog.ErrorType, uplog.ErrorDescription = uplog.detect(resp, err)
		uplog.ErrorDescription = truncate(uplog.ErrorDescription, maxFieldValueLength)
	}
	addCounter(bytesSentBlockLevelTotal, uint64(uplog.BytesSent))
	addCounter(bytesSentQualityLevelTotal, uint64(uplog.BytesSent))
	addCounter(requestsCountBlockLevel, 1)
	addCounter(requestsCountQualityLevel, 1)
	if resp != nil {
		uplog.StatusCode = resp.StatusCode
		uplog.RequestID = resp.Header.Get("x-reqid")
		if resp.Body != nil {
			if kl, ok := resp.Body.(internal_io.KnownLength); ok {
				if detectedLength, err := kl.DetectLength(); err == nil {
					uplog.BytesReceived = detectedLength
					addCounter(bytesReceivedBlockLevelTotal, uint64(detectedLength))
					addCounter(bytesReceivedQualityLevelTotal, uint64(detectedLength))
				}
			}
		}
		if resp.Request != nil {
			if resp.Request.ProtoMajor == 1 || resp.Request.ProtoMinor != 0 {
				uplog.HTTPVersion = fmt.Sprintf("%d.%d", resp.Request.ProtoMajor, resp.Request.ProtoMinor)
			} else {
				uplog.HTTPVersion = fmt.Sprintf("%d", resp.Request.ProtoMajor)
			}
		}
	}
	if uplog.TotalElapsedTime > 0 {
		if uplog.BytesSent > uplog.BytesReceived {
			uplog.PerceptiveSpeed = uplog.BytesSent * 1000 / int64(uplog.TotalElapsedTime)
		} else {
			uplog.PerceptiveSpeed = uplog.BytesReceived * 1000 / int64(uplog.TotalElapsedTime)
		}
	}
	if uplogBytes, jsonError := json.Marshal(uplog); jsonError == nil {
		uplogChan <- uplogSerializedEntry{serializedUplog: uplogBytes, getUpToken: uplog.getUpToken}
	}
	return
}

func (uplog *RequestUplog) detect(response *http.Response, err error) (errorType ErrorType, errorDescription string) {
	if err == nil {
		if response.StatusCode >= 300 {
			errorType = ErrorTypeResponseError
			errorDescription = response.Status
		}
		return
	}
	errorType = detectErrorType(err)
	errorDescription = err.Error()
	return
}

type requestBodyWithCounter struct {
	r io.ReadCloser
	c *int64
}

func newRequestBodyWithCounter(r io.ReadCloser, c *int64) *requestBodyWithCounter {
	*c = 0
	return &requestBodyWithCounter{r, c}
}

func (r *requestBodyWithCounter) Read(p []byte) (n int, err error) {
	n, err = r.r.Read(p)
	*r.c += int64(n)
	return
}

func (r *requestBodyWithCounter) Close() error {
	return r.r.Close()
}

func getElapsedTime(startTime time.Time) uint64 {
	return uint64(time.Since(startTime) / time.Millisecond)
}

func addCounter(c *uint64, n uint64) {
	if c != nil {
		atomic.AddUint64(c, n)
	}
}
