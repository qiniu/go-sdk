package bytes

import (
	"io"
	"syscall"
)

// ---------------------------------------------------

func ReadAll(src io.Reader, length int) (b []byte, err error) {
	b = make([]byte, length)
	_, err = io.ReadFull(src, b)
	return
}

// ---------------------------------------------------

type Reader struct {
	b   []byte // see strings.Reader
	off int
}

func NewReader(val []byte) *Reader {
	return &Reader{val, 0}
}

func (r *Reader) Len() int {
	if r.off >= len(r.b) {
		return 0
	}
	return len(r.b) - r.off
}

func (r *Reader) Bytes() []byte {
	return r.b[r.off:]
}

func (r *Reader) SeekToBegin() (err error) {
	r.off = 0
	return
}

func (r *Reader) Seek(offset int64, whence int) (ret int64, err error) {
	switch whence {
	case 0:
	case 1:
		offset += int64(r.off)
	case 2:
		offset += int64(len(r.b))
	default:
		err = syscall.EINVAL
		return
	}
	if offset < 0 {
		err = syscall.EINVAL
		return
	}
	if offset >= int64(len(r.b)) {
		r.off = len(r.b)
	} else {
		r.off = int(offset)
	}
	ret = int64(r.off)
	return
}

func (r *Reader) Read(val []byte) (n int, err error) {
	n = copy(val, r.b[r.off:])
	if n == 0 && len(val) != 0 {
		err = io.EOF
		return
	}
	r.off += n
	return
}

func (r *Reader) Close() (err error) {
	return
}

// ---------------------------------------------------

type Writer struct {
	b []byte
	n int
}

func NewWriter(buff []byte) *Writer {
	return &Writer{buff, 0}
}

func (p *Writer) Write(val []byte) (n int, err error) {
	n = copy(p.b[p.n:], val)
	if n == 0 && len(val) > 0 {
		err = io.EOF
		return
	}
	p.n += n
	return
}

func (p *Writer) Len() int {
	return p.n
}

func (p *Writer) Bytes() []byte {
	return p.b[:p.n]
}

func (p *Writer) Reset() {
	p.n = 0
}

// ---------------------------------------------------

