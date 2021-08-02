/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package ports

import (
	"fmt"
	"github.com/pkg/errors"
	"math/rand"
	"net"
	"time"

	"nocalhost/pkg/nhctl/log"
)

// GetRandomAvailablePort returns a random port that's available
func GetRandomAvailablePort() (int, error) {
	rand.Seed(time.Now().UnixNano())
	min := 30000
	max := 40000
	port := rand.Intn(max-min+1) + min
	address := &net.TCPAddr{
		IP:   []byte("0.0.0.0"),
		Port: port,
	}
	listener, err := net.ListenTCP("tcp", address)
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

// GetAvailablePort returns a random port that's available
func GetAvailablePort() (int, error) {
	address, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", "0.0.0.0"))
	if err != nil {
		return 0, errors.Wrap(err, "")
	}

	listener, err := net.ListenTCP("tcp", address)
	if err != nil {
		return 0, errors.Wrap(err, "")
	}

	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil

}

// IsPortAvailable returns true if the port is already taken
func IsPortAvailable(iface string, port int) bool {
	address := fmt.Sprintf("%s:%d", iface, port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Debugf("port %s is taken: %s", address, err)
		return false
	}

	defer listener.Close()
	return true
}

func IsTCP4PortAvailable(iface string, port int) bool {
	address := fmt.Sprintf("%s:%d", iface, port)
	listener, err := net.Listen("tcp4", address)
	if err != nil {
		//log.Info(err.Error())
		return false
	}

	defer listener.Close()
	return true
}
