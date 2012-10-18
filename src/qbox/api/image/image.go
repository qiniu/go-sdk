package image

import (
	"io"
	"net/http"
	. "qbox/api"
	"qbox/rpc"
	"qbox/auth/digest"
)

type Service struct {
	*Config
	Conn rpc.Client
}

// If c == nil, only can use Info(), Exif(), View()
func New(c *Config, t http.RoundTripper) *Service {
	if t == nil {
		t = http.DefaultTransport
	}
	if c != nil {
		t = digest.NewTransport(c.Access_key, c.Secret_key, t)
	}
	client := &http.Client{Transport: t}
	return &Service{c, rpc.Client{client}}
}


type ImageInfo struct {
	MimeType string `json:"format"`
	Width int `json:"width"`
	Height int `json:"height"`
	ColorModel string `json:"colorModel"`
}

func (s *Service) Info(url string) (ret ImageInfo, code int, err error) {
	url1 := url + "?imageInfo"
	code, err = s.Conn.Call(&ret, url1)
	return
}


type ImageExif struct {
	Value string `json:"val"`
	Type int `json:"type"`
}

func (s *Service) Exif(url string) (ret map[string]ImageExif , code int, err error) {
	url1 := url + "?exif"
	ret = make(map[string]ImageExif)
	code, err = s.Conn.Call(&ret, url1)
	return
}


// For more detail about the parameters, look at http://docs.qiniutek.com
// 
// Mode:<mode> = "1" or "2" 
// Width:<width>
// Height:<height>
// Quality:<quality> range in [1,100]
// Format:<format> = "jpg" or "gif" or "png" or "tif", etc.
// Sharpen:<sharpen>
// Watermark:<watermark> = "0" or "1"

func (s *Service) View(w io.Writer, url string, params map[string]string) (code int, err error) {

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
	code, err = s.Conn.Call(w, url1)
	return
}




// For more detail about the parameters, look at http://docs.qiniutek.com
//
// Thumbnail:<ImageSizeGeometry>
// Gravity:<GravityType> = "NorthWest", "North", "NorthEast", "West",
//	"Center", "East", "SouthWest", "South", "SouthEast"
// Crop:<ImageSizeAndOffsetGeometry>
// Quality:<ImageQuality>
// Rotate:<RotateDegree>
// Format:<DestinationImageFormat> = "jpg", "gif", "png", "tif", etc.
// Orient:<Auto-Orient> = "Auto" or ""
// SaveAs:<EntryURI>


type ImageHash struct {
	Hash string `json:"hash"`
}


func (s *Service) Mogr(ret interface{}, url string, params map[string]string) (code int, err error) {
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
	if entryURI, ok := params["SaveAs"]; ok && entryURI != "" {
		url1 += "/save-as/" + EncodeURI(entryURI)
	}

	code, err = s.Conn.Call(ret, url1)
	return
}


