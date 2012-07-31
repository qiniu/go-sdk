package main

import (
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"qbox/api/conf"
	"qbox/api/rs"
	"qbox/oauth"
	"strings"
)

type Qbox struct {
	qboxUser  map[string]map[string]string
	Transport map[string]*oauth.Transport
}

func NewQbox() *Qbox {
	var config = &oauth.Config{
		ClientId:     conf.CLIENT_ID,
		ClientSecret: conf.CLIENT_SECRET,
		Scope:        "Scope",
		AuthURL:      conf.AUTHORIZATION_ENDPOINT,
		TokenURL:     conf.TOKEN_ENDPOINT,
		RedirectURL:  conf.REDIRECT_URI,
	}
	transport := &oauth.Transport{
		Config:    config,
		Transport: http.DefaultTransport, // it is default
	}

	return &Qbox{
		qboxUser: map[string]map[string]string{
			"test": map[string]string{
				"user":     "", //填入用户名
				"password": "", //填入密码
			},
		},
		Transport: map[string]*oauth.Transport{
			"test": transport,
		},
	}
}

func (q *Qbox) checkTransport(app string) bool {
	_, ok := q.Transport[app]

	return ok
}

func (q *Qbox) getQboxUser(app string) (user, password string) {
	up, ok := q.qboxUser[app]
	if ok {
		user = up["user"]
		password = up["password"]
	}

	return
}

func (q *Qbox) loginQbox(app string) (b bool) {
	u, p := q.getQboxUser(app)
	_, code, err := q.Transport[app].ExchangeByPassword(u, p)
	if code != 200 {
		fmt.Println("LoginByPassword failed:", code, "-", err)
	} else {
		b = true
	}

	return
}

func (q *Qbox) tryLoginQbox(app string) (b bool) {
	if q.Transport[app].Token == nil {
		return
	}

	b = true

	refreshToken := q.Transport[app].Token.RefreshToken
	if refreshToken == "" {
		fmt.Println("Please login first to execute this command!")
		b = false
	}

	_, code, err := q.Transport[app].ExchangeByRefreshToken(refreshToken)
	if code != 200 {
		fmt.Println("LoginByRefreshToken failed:", code, "-", err)
		b = false
	}

	return
}

func (q *Qbox) CheckQboxUser(app string) (b bool) {
	if q.checkTransport(app) == false {
		return
	}

	if b = q.tryLoginQbox(app); !b {
		b = q.loginQbox(app)
	}

	return
}

func (q *Qbox) PutAuth(app string, expires int, callBackURL string) (data rs.GetRet) {

	if q.checkTransport(app) == false {
		return
	}

	service := rs.New(q.Transport[app])

	data, code, err := service.PutAuth(expires, callBackURL)
	if err != nil {
		fmt.Println("PutAuth failed:", code, "-", err)
		return
	}

	return
}

func (q *Qbox) Put(app, table, key, filePath string) (b bool) {
	if q.checkTransport(app) == false {
		return
	}

	service := rs.New(q.Transport[app])
	fileExt := strings.ToLower(path.Ext(filePath))
	mimeType := mime.TypeByExtension(fileExt)
	entryURI := table + ":" + key

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("'open file failed:", err)
		return
	}

	attr, _ := os.Stat(filePath)
	fileSize := attr.Size()

	_, code, err := service.Put(entryURI, mimeType, file, fileSize)
	if err != nil {
		fmt.Println("Put failed:", code, "-", err)
		return
	}

	b = true

	return
}

func (q *Qbox) Publish(app, uri, table string) (b bool) {
	if q.checkTransport(app) == false {
		return
	}

	service := rs.New(q.Transport[app])
	u, _ := url.Parse(uri)
	domain := u.Host
	code, err := service.Publish(domain, table)
	if err != nil {
		fmt.Println("Publish failed:", code, "-", err)
		return
	}

	b = true

	return
}

func (q *Qbox) Delete(app, table, key string) (b bool) {
	if q.checkTransport(app) == false {
		return
	}

	service := rs.New(q.Transport[app])
	entryURI := table + ":" + key
	code, err := service.Delete(entryURI)
	if err != nil {
		fmt.Println("Delete failed:", code, "-", err)
		return
	}

	b = true

	return

}

func (q *Qbox) Drop(app, table string) (b bool) {
	if q.checkTransport(app) == false {
		return
	}

	service := rs.New(q.Transport[app])
	code, err := service.Drop(table)
	if err != nil {
		fmt.Println("Drop failed:", code, "-", err)
		return
	}

	b = true

	return

}
