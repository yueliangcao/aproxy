package netx

import (
	"crypto/cipher"
	"io"
	"log"
	"net"
	_ "reflect"
	"strings"
	_ "unsafe"
)

type conn struct {
	*net.TCPConn
	r cipher.StreamReader
	w cipher.StreamWriter
}

func NewConn(c *net.TCPConn) net.Conn {
	return &conn{c, cipher.StreamReader{NewCipher(true), c}, cipher.StreamWriter{NewCipher(false), c, nil}}
}

func (c *conn) Read(b []byte) (n int, err error) {
	return c.r.Read(b)
}

func (c *conn) Write(b []byte) (n int, err error) {
	return c.w.Write(b)
}

type TCPConnCloser interface {
	CloseRead() error
	CloseWrite() error
}

func Pipe(dst net.Conn, src net.Conn) {
	defer func() {
		src.Close()
		dst.Close()
	}()

	buf := make([]byte, 4<<10)

	for {
		n, rErr := src.Read(buf[:])
		if n > 0 {
			if _, wErr := dst.Write(buf[:n]); wErr != nil {
				log.Println("pipe write err:", wErr)
				break
			}
		}

		if rErr != nil {
			if rErr != io.EOF && !strings.HasSuffix(rErr.Error(), "use of closed network connection") {
				log.Println("pipe read err:", rErr)
				log.Println(src.RemoteAddr(), "->", dst.RemoteAddr())
			}
			break
		}
	}
}
