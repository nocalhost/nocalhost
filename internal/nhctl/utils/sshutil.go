package utils

import (
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"nocalhost/pkg/nhctl/log"
	"time"
)

var DefaultRoot = &Certificate{
	Username: "root",
	Password: "root",
}

type Certificate struct {
	Username string
	Password string
}

func Reverse(cert *Certificate, sshEndpoint, remoteEndpoint, localEndpoint string) (err error) {
	sshConfig := &ssh.ClientConfig{
		User:            cert.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(cert.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Second * 60,
	}
	sshConn, err := ssh.Dial("tcp", sshEndpoint, sshConfig)
	if err != nil {
		log.Error("fail to create ssh tunnel")
		return err
	}
	defer sshConn.Close()

	// Listen on remote server port
	listener, err := sshConn.Listen("tcp", remoteEndpoint)
	if err != nil {
		log.Error("fail to listen remote endpoint")
		return err
	}
	defer listener.Close()

	log.Infof("Forwarding to %s <- %s", localEndpoint, remoteEndpoint)

	// handle incoming connections on reverse forwarded tunnel
	for {
		// Open a (local) connection to localEndpoint whose content will be forwarded so serverEndpoint
		localConn, err := net.Dial("tcp", localEndpoint)
		if err != nil {
			log.Error("dial local service error: %s", err)
			return err
		}

		remoteConn, err := listener.Accept()
		if err != nil {
			log.Error("error: %s", err)
			return err
		}
		copy(remoteConn, localConn)
	}
}

func copy(remoteConn net.Conn, localConn net.Conn) {
	stop := make(chan struct{})
	// remote -> local
	go func() {
		_, err := io.Copy(localConn, remoteConn)
		if err != nil {
			log.Error("error occurs while copy local -> remote: %s", err)
		}
		stop <- struct{}{}
	}()

	// local -> remote
	go func() {
		_, err := io.Copy(remoteConn, localConn)
		if err != nil {
			log.Error("error while copy remote -> local: %s", err)
		}
		stop <- struct{}{}
	}()

	<-stop
}
