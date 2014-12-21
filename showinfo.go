package main

import (
	"./config"
	"./netx"
	_ "encoding/binary"
	_ "errors"
	"fmt"
	_ "io"
	"net"
	_ "strconv"
)

var (
	svrAddr string
)

func main() {
	cfg, err := config.Parse("client.cfg")
	if err != nil {
		fmt.Println("config parse err: " + err.Error())
		return
	}

	svrAddr = cfg["svr_addr"].(string)

	fmt.Printf("showinfo start, connection to: %s \n", svrAddr)

	conn, err := net.Dial("tcp", svrAddr)
	if err != nil {
		fmt.Println("remote dial err: " + err.Error())
		return
	}
	conn = netx.NewConn(conn)

	if _, err = conn.Write([]byte{1}); err != nil {
		fmt.Print("conn write err: " + err.Error())
	}

	var buf [4096]byte
	for {
		n, err := conn.Read(buf[:])
		if err != nil {
			fmt.Printf("conn read err: %s \n", err.Error())
			return
		}

		fmt.Print(buf[:n])
	}

	fmt.Println("showinfo end")
}
