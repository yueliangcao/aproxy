package main

import (
	"./config"
	"./netx"
	"encoding/binary"
	"errors"
	"io"
	"log"
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

const (
	SOCKS_VER_5       = 5
	SOCKS_CMD_CONNECT = 1
)

var (
	lnAddr  string
	svrAddr string
)

func getRequest(conn net.Conn) (rawaddr []byte, host string, err error) {
	const (
		INX_VER    = 0
		INX_CMD    = 1
		INX_TYPE   = 3 // address type index
		INX_IP0    = 4 // ip addres start index
		INX_DM_LEN = 4 // domain address length index
		INX_DM0    = 5 // domain address start index

		TYPE_IPV4 = 1 // type is ipv4 address
		TYPE_DM   = 3 // type is domain address
		TYPE_IPV6 = 4 // type is ipv6 address

		LEN_IPV4    = 3 + 1 + net.IPv4len + 2 // 3(ver+cmd+rsv) + 1addrType + ipv4 + 2port
		LEN_IPV6    = 3 + 1 + net.IPv6len + 2 // 3(ver+cmd+rsv) + 1addrType + ipv6 + 2port
		LEN_DM_BASE = 3 + 1 + 1 + 2           // 3(ver+cmd+rsv) + 1addrType + 1addrLen + 2port, plus addrLen
	)
	// refer to getRequest in server.go for why set buffer size to 263
	// +----+-----+-------+------+----------+----------+
	// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+
	// 263 = 1 + 1 + 1 + 1 + 257(1addrLen + 256) + 2
	buf := make([]byte, 263)
	var n int
	// read till we get possible domain length field
	if n, err = io.ReadAtLeast(conn, buf, INX_DM_LEN+1); err != nil {
		return
	}
	// check version and cmd
	if buf[INX_VER] != SOCKS_VER_5 {
		err = errVer
		return
	}
	if buf[INX_CMD] != SOCKS_CMD_CONNECT {
		err = errCmd
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
		if _, err = io.ReadFull(conn, buf[n:reqLen]); err != nil {
			return
		}
	} else {
		err = errReqExtraData
		return
	}

	rawaddr = buf[INX_TYPE:reqLen]

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

	return
}

func handShake(conn net.Conn) (err error) {
	const (
		INX_VER     = 0
		INX_NMETHOD = 1
	)
	// version identification and method selection message in theory can have
	// at most 256 methods, plus version and nmethod field in total 258 bytes
	// the current rfc defines only 3 authentication methods (plus 2 reserved),
	// so it won't be such long in practice

	// +----+----------+----------+
	// |VER | NMETHODS | METHODS  |
	// +----+----------+----------+
	// | 1  |    1     | 1 to 255 |
	// +----+----------+----------+
	// 258 = 1 + 1 + 256
	buf := make([]byte, 258)

	var n int
	// make sure we get the nmethod field
	if n, err = io.ReadAtLeast(conn, buf, INX_NMETHOD+1); err != nil {
		return
	}
	if buf[INX_VER] != SOCKS_VER_5 {
		return errVer
	}
	nmethod := int(buf[INX_NMETHOD])
	msgLen := nmethod + 2
	if n == msgLen { // handshake done, common case
		// do nothing, jump directly to send confirmation
	} else if n < msgLen { // has more methods to read, rare case
		if _, err = io.ReadFull(conn, buf[n:msgLen]); err != nil {
			return
		}
	} else { // error, should not get extra data
		return errAuthExtraData
	}
	// send confirmation: version 5, no authentication required
	_, err = conn.Write([]byte{SOCKS_VER_5, 0})
	return
}

func handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
	}()

	if err := handShake(conn); err != nil {
		log.Println("socks handshake err", err)
		return
	}

	rawaddr, _, err := getRequest(conn)
	if err != nil {
		log.Println("socks get request err", err)
		return
	}

	//send confirmation
	if _, err = conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x08, 0x43}); err != nil {
		log.Println("send connection confirmation:", err)
		return
	}

	remote, err := net.Dial("tcp", svrAddr)
	if err != nil {
		log.Println("remote dial err", err)
		return
	}
	remote = netx.NewConn(remote.(*net.TCPConn))

	if _, err := remote.Write(rawaddr); err != nil {
		log.Println("remote write err: ", err)
		return
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
	cfg, err := config.Parse("client.cfg")
	if err != nil {
		log.Panicln("config parse err", err)
		return
	}

	lnAddr = cfg["ln_addr"].(string)
	svrAddr = cfg["svr_addr"].(string)

	log.Printf("client start, listen to: %s, send to: %s\n", lnAddr, svrAddr)

	ln, err := net.Listen("tcp", lnAddr)
	if err != nil {
		log.Panicln("listen err", err)
		return
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept err", err)
			return
		}
		log.Printf("new accept conn: %s \n", conn.RemoteAddr())
		go handleConnection(conn)
	}

	log.Println("client end")
}
