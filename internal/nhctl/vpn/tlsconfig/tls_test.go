/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package tlsconfig

import (
	"crypto/tls"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"nocalhost/internal/nhctl/vpn/util"
	"testing"
)

func init() {
	util.InitLogger(true)
}

func TestName(t *testing.T) {
	listen, _ := net.Listen("tcp", ":9090")
	listener := tls.NewListener(listen, Server)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Errorln(err)
			}
			go func(conn net.Conn) {
				bytes := make([]byte, 1024)
				all, err2 := conn.Read(bytes)
				if err2 != nil {
					log.Errorln(err2)
					return
				}
				defer conn.Close()
				fmt.Println(string(bytes[:all]))
				io.WriteString(conn, "hello client")
			}(conn)
		}
	}()
	dial, err := net.Dial("tcp", ":9090")
	if err != nil {
		log.Errorln(err)
	}

	client := tls.Client(dial, Client)
	client.Write([]byte("hi server"))
	all, err := io.ReadAll(client)
	if err != nil {
		log.Errorln(err)
	}
	fmt.Println(string(all))
}
