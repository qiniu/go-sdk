package pili

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"math"
	"net/url"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/qiniu/dyn/text"
)

// RTMPPublishURL 生成 RTMP 推流地址
func RTMPPublishURL(hub, domain, streamTitle string) string {
	return fmt.Sprintf("rtmp://%s/%s/%s", domain, hub, streamTitle)
}

// SRTPublishURL 生成 SRT 推流地址
func SRTPublishURL(hub, domain, streamTitle string) string {
	return fmt.Sprintf("srt://%s:%s?streamid=#!::h=%s/%s,m=publish,domain=%s", domain, SRTPort, hub, streamTitle, domain)
}

// RTMPPlayURL 生成 RTMP 播放地址
func RTMPPlayURL(hub, domain, streamTitle string) string {
	return fmt.Sprintf("rtmp://%s/%s/%s", domain, hub, streamTitle)
}

// HLSPlayURL 生成 HLS 播放地址
func HLSPlayURL(hub, domain, streamTitle string) string {
	return fmt.Sprintf("https://%s/%s/%s.m3u8", domain, hub, streamTitle)
}

// HDLPlayURL 生成 HDL 播放地址
func HDLPlayURL(hub, domain, streamTitle string) string {
	return fmt.Sprintf("https://%s/%s/%s.flv", domain, hub, streamTitle)
}

// SignPublishURL 推流 URL 签算（直播鉴权）
// 支持鉴权规格为
// - static: 静态鉴权
// - expiry: 限时鉴权
// - expiry_sk: 限时鉴权SK
// - dynamic: 动态鉴权
func SignPublishURL(publishURL string, args SignPublishURLArgs) (string, error) {

	u, err := url.Parse(publishURL)
	if err != nil {
		return "", ErrInvalidArgs
	}

	if args.ExpireAt <= 0 {
		args.ExpireAt = time.Now().Add(10 * time.Minute).Unix()
	}

	isSRT := u.Scheme == SRTScheme
	if isSRT {
		u, err = parseSRTURL(u)
		if err != nil {
			return "", err
		}
	}
	query := u.Query()
	token := ""
	switch args.SecurityType {
	case SecurityTypeNull, SecurityTypeNone:
		return publishURL, nil
	case SecurityTypeStatic:
		if len(args.PublishKey) == 0 {
			return "", ErrInvalidArgs
		}
		query.Set(SecurityTypeStaticKeyPart, args.PublishKey)
		u.RawQuery = query.Encode()
	case SecurityTypeExpiry:
		if len(args.PublishKey) == 0 {
			return "", ErrInvalidArgs
		}
		query.Set(SecurityTypeExpiryTsPart, strconv.FormatInt(args.ExpireAt, 10))
		query.Del(SecurityTypeTokenPart)
		u.RawQuery = query.Encode()
		h := hmac.New(sha1.New, []byte(args.PublishKey))
		h.Write([]byte(u.Path + "?" + u.RawQuery))
		token = base64.URLEncoding.EncodeToString(h.Sum(nil))
	case SecurityTypeExpirySK:
		if len(args.AccessKey) == 0 || len(args.SecretKey) == 0 {
			return "", ErrInvalidArgs
		}
		query.Set(SecurityTypeExpirySkTsPart, strconv.FormatInt(args.ExpireAt, 10))
		query.Del(SecurityTypeTokenPart)
		u.RawQuery = query.Encode()
		h := hmac.New(sha1.New, []byte(args.SecretKey))
		h.Write([]byte(u.Path + "?" + u.RawQuery))
		token = fmt.Sprintf("%s:%s", args.AccessKey, base64.URLEncoding.EncodeToString(h.Sum(nil)))
	case SecurityTypeDynamic:
		query.Set(SecurityTypeDynamicNoncePart, strconv.Itoa(args.Nonce))
		query.Del(SecurityTypeTokenPart)
		u.RawQuery = query.Encode()
		h := hmac.New(sha1.New, []byte(args.PublishKey))
		h.Write([]byte(u.Path + "?" + u.RawQuery))
		token = base64.URLEncoding.EncodeToString(h.Sum(nil))
	default:
		return "", ErrUnsupportedSecurityType
	}

	if isSRT {
		return convertSRTURL(u, token), nil
	}

	ret := u.String()
	if len(token) > 0 {
		ret += "&token=" + token
	}

	return ret, nil
}

// SignPlayURL 播放 URL 签算（时间戳防盗链）
// 支持鉴权规格为
// - tsStartMD5: 时间戳防盗链开始时间限制
// - tsExpireMD5: 时间戳防盗链结束时间限制
func SignPlayURL(playURL string, args SignPlayURLArgs) (string, error) {

	u, err := url.Parse(playURL)
	if err != nil {
		return "", ErrInvalidArgs
	}

	if args.TsBase <= 0 {
		args.TsBase = 16
	}

	query := u.Query()

	switch args.SecurityType {
	case SecurityTypeNull, SecurityTypeNone:
		return playURL, nil
	case SecurityTypeTsStartMD5, SecurityTypeTsExpireMD5:
		query.Set(args.TsPart, strconv.FormatInt(args.Timestamp, args.TsBase))
		query.Del(args.SignPart)
	default:
		return "", ErrUnsupportedSecurityType
	}

	ctx := magicCtx(u, query, args.Key)

	sign, err := text.Subst(args.Rule, ctx, text.Fmttype_Text, true)
	if err != nil {
		return "", ErrInvalidRule
	}

	query.Set(args.SignPart, fmt.Sprintf("%x", md5.Sum([]byte(sign))))
	u.RawQuery = query.Encode()
	return u.String(), nil
}

// magicCtx 生成魔法变量
// - $(key): 密钥
// - $(path): URL 中的 path 部分
// - $(streamKey): URL 中的 hub/stream 部分
// - $(streamTitle): URL 中的 stream 部分
// - $(path_<number>): URL 中的 path 部分，<number> 表示 path 层级
// - $(_<query>): URL 中的 query 字段，举例: key1=val，魔法变量中使用 $(_key1) 表示 val
func magicCtx(u *url.URL, query url.Values, key string) map[string]interface{} {
	path := u.Path
	ctx := map[string]interface{}{
		"key":         key,
		"path":        path,
		"streamKey":   parseStreamKey(path),
		"streamTitle": parseStreamTitile(path),
	}

	for k, v := range strings.Split(path, "/") {
		if k == 0 {
			continue
		}
		ctx["path_"+strconv.Itoa(k)] = v
	}

	for k, v := range query {
		if len(v) != 0 {
			q := "_" + k
			ctx[q] = v[0]
		}
	}
	return ctx
}

// convertSRTURL 转换 SRT URL
func convertSRTURL(u *url.URL, token string) string {
	query := u.Query()
	query.Set("m", "publish")
	rawQuery := strings.Replace(query.Encode(), "&", ",", -1)
	if len(token) > 0 {
		rawQuery += ",token=" + token
	}
	if !strings.Contains(u.Host, ":") {
		u.Host += ":" + SRTPort
	}

	return fmt.Sprintf("srt://%s?streamid=#!::h=%s,%s", u.Host, u.Path[1:], rawQuery)
}

// parseSRTURL 解析 SRT URL
// 相关参数放入 query 中
func parseSRTURL(u *url.URL) (*url.URL, error) {
	if !strings.HasPrefix(u.Fragment, "!::") {
		return nil, ErrInvalidArgs
	}

	query := url.Values{}
	for _, split := range strings.Split(u.Fragment, ",") {
		split2 := strings.SplitN(split, "=", 2)
		if len(split2) != 2 {
			return nil, ErrInvalidArgs
		}
		switch split2[0] {
		case "!::h":
			u.Path = "/" + split2[1]
		case "m":
			continue
		default:
			query.Set(split2[0], split2[1])
		}
	}
	u.RawQuery = query.Encode()
	u.Fragment = ""
	return u, nil
}

// url 拼接生成 API 访问 URL
func (m *Manager) url(format string, args ...interface{}) string {
	query := ""
	if len(args) > 0 {
		if q, ok := isQuery(args[len(args)-1]); ok {
			query = q.Encode()
			args = args[:len(args)-1]
		}
	}
	reqURL := m.apiHTTPScheme + m.apiHost + fmt.Sprintf(format, args...)
	if !strings.Contains(reqURL, "?") && query != "" {
		reqURL += "?" + query
	}
	return reqURL
}

func isQuery(q interface{}) (url.Values, bool) {
	switch q := q.(type) {
	case url.Values:
		return q, true
	default:
		return nil, false
	}
}

func setQuery(q url.Values, key string, v interface{}) {
	if !isZero(v) {
		q.Set(key, fmt.Sprint(v))
	}
}

// isZero reports whether v is the zero value for its type.
// FIXME IsZero was added to the reflect standard package in go 1.13
// compatible with older versions of go 1.13
func isZero(_v interface{}) bool {
	v := reflect.ValueOf(_v)
	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return math.Float64bits(v.Float()) == 0
	case reflect.Complex64, reflect.Complex128:
		c := v.Complex()
		return math.Float64bits(real(c)) == 0 && math.Float64bits(imag(c)) == 0
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if !isZero(v.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		return v.IsNil()
	case reflect.String:
		return v.Len() == 0
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !isZero(v.Field(i)) {
				return false
			}
		}
		return true
	default:
		return true
	}
}

func parseStreamKey(path string) string {
	noExt := strings.TrimSuffix(path, filepath.Ext(path))
	return strings.TrimPrefix(noExt, "/")
}

func parseStreamTitile(path string) string {
	noExt := strings.TrimSuffix(path, filepath.Ext(path))
	noFirstSlash := strings.TrimPrefix(noExt, "/")
	slash := strings.IndexByte(noFirstSlash, '/')
	return noFirstSlash[slash+1:]
}
