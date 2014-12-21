package main

import (
	"./config"
	"./logx"
	"./netx"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
)

var (
	errAddrType      = errors.New("socks addr type not supported")
	errVer           = errors.New("socks version not supported")
	errMethod        = errors.New("socks only support 1 method now")
	errAuthExtraData = errors.New("socks authentication get extra data")
	errReqExtraData  = errors.New("socks request get extra data")
	errCmd           = errors.New("socks command not supported")
)

//var info *net.Conn
var logger *logx.Logger

func getRequest(conn net.Conn) (extra []byte, host string, err error) {
	const (
		idType  = 0 // address type index
		idIP0   = 1 // ip addres start index
		idDmLen = 1 // domain address length index
		idDm0   = 2 // domain address start index

		typeIPv4 = 1 // type is ipv4 address
		typeDm   = 3 // type is domain address
		typeIPv6 = 4 // type is ipv6 address

		lenIPv4   = 1 + net.IPv4len + 2 // 1addrType + ipv4 + 2port
		lenIPv6   = 1 + net.IPv6len + 2 // 1addrType + ipv6 + 2port
		lenDmBase = 1 + 1 + 2           // 1addrType + 1addrLen + 2port, plus addrLen
	)
	// refer to getRequest in server.go for why set buffer size to 263
	buf := make([]byte, 263)
	var n int
	// read till we get possible domain length field
	if n, err = io.ReadAtLeast(conn, buf, idDmLen+1); err != nil {
		return
	}

	reqLen := -1
	switch buf[idType] {
	case typeIPv4:
		reqLen = lenIPv4
	case typeIPv6:
		reqLen = lenIPv6
	case typeDm:
		reqLen = int(buf[idDmLen]) + lenDmBase
	default:
		err = errAddrType
		return
	}

	if n == reqLen {
		// common case, do nothing
	} else if n < reqLen { // rare case
		if n, err = io.ReadFull(conn, buf[n:reqLen]); err != nil {
			return
		}
	}

	switch buf[idType] {
	case typeIPv4:
		host = net.IP(buf[idIP0 : idIP0+net.IPv4len]).String()
	case typeIPv6:
		host = net.IP(buf[idIP0 : idIP0+net.IPv6len]).String()
	case typeDm:
		host = string(buf[idDm0 : idDm0+buf[idDmLen]])
	}
	port := binary.BigEndian.Uint16(buf[reqLen-2 : reqLen])
	host = net.JoinHostPort(host, strconv.Itoa(int(port)))
	extra = buf[reqLen:n]

	return
}

func handleConnection(conn net.Conn) (err error) {
	isCloseConn := true
	defer func() {
		if isCloseConn {
			conn.Close()
		}
	}()

	//根据连接后的第一次请求的第一个byte，判断连接类型
	b := [1]byte{}
	if _, err := conn.Read(b[:]); err != nil {
		fmt.Println("read type err: " + err.Error())
		return err
	}

	fmt.Printf("conn type :%d \n", b[0])
	if b[0] == 1 {
		//info = conn
		logger = logx.New(conn, "", logx.LstdFlags)
		isCloseConn = false
		return nil
	}

	extra, host, err := getRequest(conn)
	if err != nil {
		fmt.Println("get request err: " + err.Error())
		return err
	}

	remote, err := net.Dial("tcp", host)
	if err != nil {
		fmt.Println("remote dial err: " + err.Error())
		return err
	}
	fmt.Println("remote dial addr :" + host)

	if extra != nil {
		if _, err := remote.Write(extra); err != nil {
			fmt.Println("remote write extra err: " + err.Error())
			return err
		}
	}

	go netx.Pipe(remote, conn)
	netx.Pipe(conn, remote)

	return err
}

func main() {
	cfg, err := config.Parse("server.cfg")
	if err != nil {
		fmt.Println("config parse err: " + err.Error())
		return
	}

	lnAddr := cfg["ln_addr"].(string)

	fmt.Println("server start, listen to: " + lnAddr)

	ln, err := net.Listen("tcp", lnAddr)
	if err != nil {
		fmt.Println("listen err: " + err.Error())
		return
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("accept err: " + err.Error())
			return
		}
		fmt.Printf("new accept conn: %s \n", conn.RemoteAddr())
		logger.Printf("new accept conn: %s \n", conn.RemoteAddr())
		go handleConnection(netx.NewConn(conn))
	}

	fmt.Println("server end")
}
