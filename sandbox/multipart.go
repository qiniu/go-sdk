package sandbox

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"path/filepath"
)

// multipartFileWriter 封装 multipart 文件上传的 writer。
type multipartFileWriter struct {
	w *multipart.Writer
}

// newMultipartWriter 创建一个写入到 w 的 multipartFileWriter。
func newMultipartWriter(w io.Writer) *multipartFileWriter {
	return &multipartFileWriter{w: multipart.NewWriter(w)}
}

// contentType 返回 multipart 的 Content-Type 头。
func (m *multipartFileWriter) contentType() string {
	return m.w.FormDataContentType()
}

// writeFile 将文件数据写入 multipart body。
func (m *multipartFileWriter) writeFile(fieldName, fileName string, data []byte) error {
	part, err := m.w.CreateFormFile(fieldName, filepath.Base(fileName))
	if err != nil {
		return err
	}
	_, err = part.Write(data)
	return err
}

// writeFileFullPath 将文件数据写入 multipart body，part filename 使用完整路径。
// 用于批量上传场景，服务端从 part filename 获取文件路径。
// 不使用 CreateFormFile，因为它会对 filename 执行 filepath.Base()。
func (m *multipartFileWriter) writeFileFullPath(fieldName, fullPath string, data []byte) error {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fullPath))
	h.Set("Content-Type", "application/octet-stream")
	part, err := m.w.CreatePart(h)
	if err != nil {
		return err
	}
	_, err = part.Write(data)
	return err
}

// close 关闭 multipart writer。
func (m *multipartFileWriter) close() error {
	return m.w.Close()
}
