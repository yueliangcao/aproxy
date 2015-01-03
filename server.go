package main

import (
	"./config"
	"./netx"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"runtime"
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

func getRequest(conn net.Conn) (extra []byte, host string, err error) {
	const (
		INX_TYPE   = 0 // address type index
		INX_IP0    = 1 // ip addres start index
		INX_DM_LEN = 1 // domain address length index
		INX_DM0    = 2 // domain address start index

		TYPE_IPV4 = 1 // type is ipv4 address
		TYPE_DM   = 3 // type is domain address
		TYPE_IPV6 = 4 // type is ipv6 address

		LEN_IPV4    = 1 + net.IPv4len + 2 // 1addrType + ipv4 + 2port
		LEN_IPV6    = 1 + net.IPv6len + 2 // 1addrType + ipv6 + 2port
		LEN_DM_BASE = 1 + 1 + 2           // 1addrType + 1addrLen + 2port, plus addrLen
	)
	// +------+----------+----------+
	// | ATYP | DST.ADDR | DST.PORT |
	// +------+----------+----------+
	// |  1   | Variable |    2     |
	// +------+----------+----------+
	// 260 = 1 + 257(1addrLen + 256) + 2
	buf := make([]byte, 260)
	var n int
	// read till we get possible domain length field
	if n, err = io.ReadAtLeast(conn, buf, INX_DM_LEN+1); err != nil {
		return
	}

	reqLen := -1
	switch buf[INX_TYPE] {
	case TYPE_IPV4:
		reqLen = LEN_IPV4
	case TYPE_IPV6:
		reqLen = LEN_IPV6
	case TYPE_DM:
		reqLen = int(buf[INX_DM_LEN]) + LEN_DM_BASE
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

	switch buf[INX_TYPE] {
	case TYPE_IPV4:
		host = net.IP(buf[INX_IP0 : INX_IP0+net.IPv4len]).String()
	case TYPE_IPV6:
		host = net.IP(buf[INX_IP0 : INX_IP0+net.IPv6len]).String()
	case TYPE_DM:
		host = string(buf[INX_DM0 : INX_DM0+buf[INX_DM_LEN]])
	}
	port := binary.BigEndian.Uint16(buf[reqLen-2 : reqLen])
	host = net.JoinHostPort(host, strconv.Itoa(int(port)))
	extra = buf[reqLen:n]

	return
}

func handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		log.Println("connect:", conn.RemoteAddr(), "closed")
	}()

	extra, host, err := getRequest(conn)
	if err != nil {
		log.Println("get request err:", err)
		return
	}

	remote, err := net.Dial("tcp", host)
	if err != nil {
		log.Println("remote dial err:", err)
		return
	}
	log.Println("remote dial addr:", host)

	if extra != nil {
		if _, err := remote.Write(extra); err != nil {
			log.Println("remote write extra err:", err)
			return
		}
	}

	done := make(chan bool)

	go func() {
		netx.Pipe(remote, conn)
		done <- true
	}()

	netx.Pipe(conn, remote)

	<-done
}

func main() {
	n := runtime.NumCPU()
	log.Println("cpu num:", n)
	//runtime.GOMAXPROCS(n)

	cfg, err := config.Parse("server.cfg")
	if err != nil {
		log.Panicln("config parse err:", err)
		return
	}

	lnAddr := cfg["ln_addr"].(string)

	log.Println("server start, listen to:", lnAddr)

	ln, err := net.Listen("tcp", lnAddr)
	if err != nil {
		log.Panicln("listen err:", err)
		return
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept err", err)
			return
		}
		log.Println("new accept conn:", conn.RemoteAddr())
		go handleConnection(netx.NewConn(conn.(*net.TCPConn)))
	}

	log.Println("server end")
}
