package clientv2

import (
	"io"
	"mime/multipart"

	"github.com/qiniu/go-sdk/v7/internal/context"
	compatible_io "github.com/qiniu/go-sdk/v7/internal/io"
)

type (
	keyValuePair struct {
		key, value string
	}
	keyFilePair struct {
		key, fileName string
		stream        compatible_io.ReadSeekCloser
	}

	MultipartForm struct {
		values []keyValuePair
		files  []keyFilePair
		ctx    context.Context
		cancel context.CancelCauseFunc
		w      *io.PipeWriter
	}

	multipartFormReader struct {
		multipartWriter *multipart.Writer
		form            *MultipartForm
		r               *io.PipeReader
	}
)

func (f *MultipartForm) SetValue(key, value string) *MultipartForm {
	f.values = append(f.values, keyValuePair{key: key, value: value})
	return f
}

func (f *MultipartForm) SetFile(key, fileName string, stream compatible_io.ReadSeekCloser) *MultipartForm {
	f.files = append(f.files, keyFilePair{key: key, fileName: fileName, stream: stream})
	return f
}

func newMultipartFormReader(form *MultipartForm) *multipartFormReader {
	reader := &multipartFormReader{form: form}
	reader.r, form.w = io.Pipe()
	reader.multipartWriter = multipart.NewWriter(form.w)

	go func(multipartWriter *multipart.Writer, w *io.PipeWriter, ctx context.Context, cancel context.CancelCauseFunc) {
		defer w.Close()
		defer multipartWriter.Close()

		for _, pair := range form.values {
			select {
			case <-ctx.Done():
				return
			default:
				if err := multipartWriter.WriteField(pair.key, pair.value); err != nil {
					cancel(err)
					return
				}
			}
		}
		for _, pair := range form.files {
			select {
			case <-ctx.Done():
				return
			default:
				if err := reader.createFormFile(pair.key, pair.fileName, pair.stream); err != nil {
					cancel(err)
					return
				}
			}
		}
	}(reader.multipartWriter, form.w, form.ctx, form.cancel)

	return reader
}

func (r *multipartFormReader) Read(p []byte) (int, error) {
	select {
	case <-r.form.ctx.Done():
		return 0, context.Cause(r.form.ctx)
	default:
		return r.r.Read(p)
	}
}

func (r *multipartFormReader) Close() (err error) {
	r.form.cancel(io.ErrClosedPipe)
	err = r.r.Close()
	for _, pair := range r.form.files {
		if e := pair.stream.Close(); e != nil && err == nil {
			err = e
		}
	}
	return err
}

func (r *multipartFormReader) formDataContentType() string {
	return r.multipartWriter.FormDataContentType()
}

func (r *multipartFormReader) createFormFile(fieldName, fileName string, stream compatible_io.ReadSeekCloser) error {
	if w, err := r.multipartWriter.CreateFormFile(fieldName, fileName); err != nil {
		return err
	} else if _, err := io.Copy(w, stream); err != nil {
		return err
	}
	return nil
}

func GetMultipartFormRequestBody(info *MultipartForm) GetRequestBody {
	return func(o *RequestParams) (io.ReadCloser, error) {
		if cancel := info.cancel; cancel != nil {
			cancel(io.ErrClosedPipe)
		}
		info.ctx, info.cancel = context.WithCancelCause(context.Background())

		if w := info.w; w != nil {
			w.Close()
			info.w = nil
		}

		for _, pair := range info.files {
			if _, err := pair.stream.Seek(0, io.SeekStart); err != nil {
				return nil, err
			}
		}
		r := newMultipartFormReader(info)
		o.Header.Add("Content-Type", r.formDataContentType())
		return r, nil
	}
}
