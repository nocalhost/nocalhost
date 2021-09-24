/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package utils

import (
	"context"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"nocalhost/pkg/nhctl/log"
	"time"
)

var DefaultRoot = &Account{
	Username: "root",
	Password: "root",
}

type Account struct {
	Username string
	Password string
}

func Reverse(account *Account, sshEndpoint, remoteEndpoint, localEndpoint string) (err error) {
	sshConfig := &ssh.ClientConfig{
		User:            account.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(account.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Second * 60,
	}
	sshConn, err := ssh.Dial("tcp", sshEndpoint, sshConfig)
	if err != nil {
		log.Error("fail to create ssh connection")
		return err
	}
	defer sshConn.Close()

	listener, err := sshConn.Listen("tcp", remoteEndpoint)
	if err != nil {
		log.Error("fail to listen remote endpoint")
		return err
	}
	defer listener.Close()

	log.Infof("Forwarding to %s <- %s", localEndpoint, remoteEndpoint)

	for {
		remoteConn, err := listener.Accept()
		if err != nil {
			log.Error("error: %s", err)
			return err
		}
		localConn, err := net.Dial("tcp", localEndpoint)
		if err != nil {
			log.Errorf("dial local service error, err: %v", err)
			return err
		}
		go copyStream(remoteConn, localConn)
	}
}

func copyStream(remoteConn net.Conn, localConn net.Conn) {
	cancel, cancelFunc := context.WithCancel(context.Background())
	go func() {
		if _, err := io.Copy(localConn, remoteConn); err != nil {
			log.Error("error occurs while copy local -> remote: %s", err)
		}
		cancelFunc()
	}()
	go func() {
		if _, err := io.Copy(remoteConn, localConn); err != nil {
			log.Error("error occurs while copy remote -> local: %s", err)
		}
		cancelFunc()
	}()
	<-cancel.Done()
}
