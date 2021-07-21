/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
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
		localConn, err := net.Dial("tcp", localEndpoint)
		if err != nil {
			log.Errorf("dial local service error, err: %v", err)
			return err
		}

		remoteConn, err := listener.Accept()
		if err != nil {
			log.Error("error: %s", err)
			return err
		}
		copyStream(remoteConn, localConn)
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
