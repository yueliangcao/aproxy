package netx

import (
	//"fmt"
	"crypto/cipher"
	"net"
)

type Conn struct {
	net.Conn
	R cipher.StreamReader
	W cipher.StreamWriter
}

func NewConn(conn net.Conn) *Conn {
	return &Conn{conn, cipher.StreamReader{NewCipher(true), conn}, cipher.StreamWriter{NewCipher(false), conn, nil}}
}

func (conn *Conn) Read(b []byte) (n int, err error) {
	//return conn.Conn.Read(b)
	return conn.R.Read(b)
}

func (conn *Conn) Write(b []byte) (n int, err error) {
	//return conn.Conn.Write(b)
	return conn.W.Write(b)
}

func Pipe(dst net.Conn, src net.Conn) {
	defer func() {
		src.Close()
		dst.Close()
	}()

	var buf [4096]byte
	for {
		n, err := src.Read(buf[:])
		if err != nil {
			//fmt.Printf("pipe src %s read err: %s \n", src.RemoteAddr(), err.Error())
			return
		}

		n, err = dst.Write(buf[:n])
		if err != nil {
			//fmt.Printf("pipe dst %s write err: %s \n", dst.RemoteAddr(), err.Error())
			return
		}
	}
}
