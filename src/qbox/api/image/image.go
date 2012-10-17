package image

import (
	"io"
	"net/http"
	. "qbox/api"
	"qbox/rpc"
)

var (
	ims *Service
)

type Service struct {
	*Config
	Conn rpc.Client
}

func New(args... interface{}) *Service {
	var (
		t http.RoundTripper
	)
	for _,v := range args {
		switch v.(type) {
		case http.RoundTripper:
			t = v.(http.RoundTripper)
			break
		}
	}
	if t == nil {
		t = http.DefaultTransport
	}
	client := &http.Client{Transport: t}
	return &Service{nil, rpc.Client{client}}
}

func init() {
	ims = New()
}

type ImageInfo struct {
	MimeType string `json:"format"`
	Width int `json:"width"`
	Height int `json:"height"`
	ColorModel string `json:"colorModel"`
}

func Info(url string) (ret ImageInfo, code int, err error) {
	url1 := url + "?imageInfo"
	code, err = ims.Conn.Call(&ret, url1)
	return
}


type ImageExif struct {
	Value string `json:"val"`
	Type int `json:"type"`
}

func Exif(url string) (ret map[string]ImageExif , code int, err error) {
	url1 := url + "?exif"
	ret = make(map[string]ImageExif)
	code, err = ims.Conn.Call(&ret, url1)
	return
}

/*
<Mode>
    图像缩略处理的模式，分为如下几种：
    <mode> = 1，表示限定目标缩略图的宽度和高度，放大并从缩略图中央处裁剪为指定 <Width>x<Height> 大小的图片。
    <mode> = 2，指定 <Width> 和 <Height>，表示限定目标缩略图的长边，短边等比缩略自适应，将缩略图的大小限定在指定的宽高矩形内。
    <mode> = 2，指定 <Width> 但不指定 <Height>，表示限定目标缩略图的宽度，高度等比缩略自适应。
    <mode> = 2，指定 <Height> 但不指定 <Width>，表示限定目标缩略图的高度，宽度等比缩略自适应。

<Width>
    指定目标缩略图的宽度，单位：像素（px）
<Height>
    指定目标缩略图的高度，单位：像素（px）
<Quality>
    指定目标缩略图的图像质量，取值范围 1-100
<Format>
    指定目标缩略图的输出格式，取值范围：jpg, gif, png, tif 等图片格式
<Sharpen>
    指定目标缩略图的锐化指数，值为正整数，此数值越大，锐化度越高，图像细节损失越大
<HasWatermark>
    是否打水印，<HasWatermark> 可选值为 0 或者 1

*/
func View(w io.Writer, url string, params map[string]string) (code int, err error) {

	url1 := url + "?imageView/"
	if mode, ok := params["Mode"]; ok && mode != "" {
		url1 += mode
	}
	if width, ok := params["Width"]; ok && width != "" {
		url1 += "/w/" + width
	}
	if height, ok := params["Height"]; ok && height != "" {
		url1 += "/h/" + height
	}
	if quality, ok := params["Quality"]; ok && quality != "" {
		url1 += "/q/" + quality
	}
	if format, ok := params["Format"]; ok && format != "" {
		url1 += "/format/" + format
	}
	if sharpen, ok := params["Sharpen"]; ok && sharpen != "" {
		url1 += "/sharpen/" + sharpen
	}
	if wm, ok := params["Watermark"]; ok && wm != "" {
		url1 += "/watermark/" + wm
	}
	code, err = ims.Conn.Call(w, url1)
	return
}


/*

/thumbnail/<ImageSizeGeometry>
/gravity/<GravityType> =NorthWest, North, NorthEast, West, Center, East, SouthWest, South, SouthEast
/crop/<ImageSizeAndOffsetGeometry>
/quality/<ImageQuality>
/rotate/<RotateDegree>
/format/<DestinationImageFormat> =jpg, gif, png, tif, etc.
/auto-orient

*/
func Mogr(w io.Writer, url string, params map[string]string) (code int, err error) {
	url1 := url + "?imageMogr"
	if thumb, ok := params["Thumbnail"]; ok && thumb != "" {
		url1 += "/thumbnail/" + thumb
	}
	if gravity, ok := params["Gravity"]; ok && gravity != "" {
		url1 += "/gravity/" + gravity
	}
	if crop, ok := params["Crop"]; ok && crop != "" {
		url1 += "/crop/" + crop
	}
	if quality, ok := params["Quality"]; ok && quality != "" {
		url1 += "/quality/" + quality
	}
	if rotate, ok := params["Rotate"]; ok && rotate != "" {
		url1 += "/rotate/" + rotate
	}
	if format, ok := params["format"]; ok && format != "" {
		url1 += "/format/" + format
	}
	if orient, ok := params["Orient"]; ok && orient == "true" {
		url1 += "/auto-orient"
	}

	code, err = ims.Conn.Call(w, url1)
	return
}