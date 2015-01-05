package netx

import (
	"crypto/cipher"
	"io"
	"log"
	"net"
	_ "reflect"
	_ "strings"
	_ "unsafe"
)

type conn struct {
	net.Conn
	r cipher.StreamReader
	w cipher.StreamWriter
}

func NewConn(c net.Conn) net.Conn {
	return &conn{c, cipher.StreamReader{NewCipher(true), c}, cipher.StreamWriter{NewCipher(false), c, nil}}
}

func (c *conn) Read(b []byte) (n int, err error) {
	return c.r.Read(b)
}

func (c *conn) Write(b []byte) (n int, err error) {
	return c.w.Write(b)
}

func Pipe(dst net.Conn, src net.Conn) {
	defer src.Close()
	defer dst.Close()

	_, err := io.Copy(dst, src)
	if err != nil && err != io.EOF {
		log.Println("pipe err:", err)
	}

	// buf := make([]byte, 4<<10)

	// for {
	// 	n, rErr := src.Read(buf[:])
	// 	if n > 0 {
	// 		if _, wErr := dst.Write(buf[:n]); wErr != nil {
	// 			log.Println("pipe write err:", wErr)
	// 			break
	// 		}
	// 	}

	// 	if rErr != nil {
	// 		if rErr != io.EOF && !strings.HasSuffix(rErr.Error(), "use of closed network connection") {
	// 			log.Println("pipe read err:", rErr)
	// 			log.Println(src.RemoteAddr(), "->", dst.RemoteAddr())
	// 		}
	// 		break
	// 	}
	// }
}
