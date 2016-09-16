package goproxy

import (
	"bufio"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
)

type connResponseWriter struct {
	dst io.Writer
}

func (w *connResponseWriter) Header() http.Header {
	panic("proxy: ConnectionResponseWriter does not implement Header()")
}

func (w *connResponseWriter) Write(data []byte) (int, error) {
	return w.dst.Write(data)
}

func (w *connResponseWriter) WriteHeader(code int) {
	panic("proxy: ConnectionResponseWriter does not implement WriteHeader(int)")
}

func (w *connResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	conn, ok := w.dst.(net.Conn)

	if !ok {
		return nil, nil, errors.New("proxy: nested io.Writer does not implement net.Conn interface")
	}

	rw := bufio.NewReadWriter(
		bufio.NewReader(io.MultiReader()),
		bufio.NewWriter(ioutil.Discard),
	)

	return conn, rw, nil
}

func NewConnResponseWriter(dst io.Writer) *connResponseWriter {
	return &connResponseWriter{
		dst: dst,
	}
}

func Error(w http.ResponseWriter, error string, code int) {

}
