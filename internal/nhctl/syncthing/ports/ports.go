/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ports

import (
	"fmt"
	"math/rand"
	"net"
	"nocalhost/pkg/nhctl/log"
	"time"
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
		return 0, err
	}

	listener, err := net.ListenTCP("tcp", address)
	if err != nil {
		return 0, err
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
