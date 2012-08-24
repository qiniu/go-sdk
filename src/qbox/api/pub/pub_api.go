package pub

import (
	"strconv"
	"net/http"
	"qbox/utils/rpc"
	. "qbox/api/conf"
)

type Service struct {
	Conn rpc.Client
}

func New(t http.RoundTripper) *Service {
	return &Service{rpc.Client{&http.Client{Transport: t}}}
}

type BucketInfo struct {
	Source string	`json:"source" bson:"source"`
	Host string `json:"host" bson:"host"`
	Expires int		`json:"expires" bson:"expires"`
	Protected int `json:"protected" bson:"protected"`
	Separator string `json:"separator" bson:"separator"`
	Styles map[string]string `json:"styles" bson:"styles"`
}

func (pub *Service) Image(bucketName string, srcSiteUrls []string, srcHost string, expires int) (code int, err error) {
	url := PU_HOST + "/image/" + bucketName
	for _, srcSiteUrl := range srcSiteUrls {
		url += "/from/" + rpc.EncodeURI(srcSiteUrl)
	}
	if expires != 0 {
		url += "/expires/" + strconv.Itoa(expires)
	}
	if srcHost != "" {
		url += "/host/" + rpc.EncodeURI(srcHost)
	}
	return pub.Conn.Call(nil, url)
}

func (pub *Service) Unimage(bucketName string) (code int, err error) {
	return pub.Conn.Call(nil, PU_HOST + "/unimage/" + bucketName)
}

func (pub *Service) Info(bucketName string) (info BucketInfo, code int, err error) {
	code, err = pub.Conn.Call(&info, PU_HOST+"/info/"+bucketName)
	return
}

func (pub *Service) AccessMode(bucketName string, mode int) (code int, err error) {
	return pub.Conn.Call(nil, PU_HOST+"/accessMode/"+bucketName+"/mode/"+strconv.Itoa(mode))
}

func (pub *Service) Separator(bucketName string, sep string) (code int, err error) {
	return pub.Conn.Call(nil, PU_HOST+"/separator/"+bucketName+"/sep/"+rpc.EncodeURI(sep))
}

func (pub *Service) Style(bucketName string, name string, style string) (code int, err error) {
	return pub.Conn.Call(nil, PU_HOST+"/style/"+bucketName+"/name/"+rpc.EncodeURI(name)+"/style/"+rpc.EncodeURI(style))
}

func (pub *Service) Unstyle(bucketName string, name string) (code int, err error) {
	return pub.Conn.Call(nil, PU_HOST+"/unstyle/"+bucketName+"/name/"+rpc.EncodeURI(name))
}

// ----------------------------------------------------------
