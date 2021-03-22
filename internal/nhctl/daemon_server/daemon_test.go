/*
Copyright 2021 The Nocalhost Authors.
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

package daemon_server

import (
	"fmt"
	"net"
	"testing"
)

func TestStartDaemonServer(t *testing.T) {
	StartDaemon()
}

func TestStartDaemonClient(t *testing.T) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", "0.0.0.0", DefaultDaemonPort))
	if err != nil {
		panic(err)
	}
	_, err = conn.Write([]byte("Hello World"))
	if err != nil {
		panic(err)
	}
	defer conn.Close()
}
