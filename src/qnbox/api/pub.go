package api

import (
	"strconv"
)

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
