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

package daemon_client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"net"
	"nocalhost/internal/nhctl/daemon_common"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/syncthing/daemon"
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/internal/nhctl/utils"
	"os/exec"
	"time"
)

type DaemonClient struct {
	isSudo                 bool
	daemonServerListenPort int
}

func runDaemonServer(isSudoUser bool) error {
	// todo, run in background
	nhctlPath, err := exec.LookPath(utils.GetNhctlBinName())
	if err != nil {
		return errors.Wrap(err, "")
	}
	daemonArgs := []string{nhctlPath, "daemon", "start"}
	if isSudoUser {
		daemonArgs = append(daemonArgs, "--sudo", "true")
	}

	return daemon.RunSubProcess(daemonArgs, nil, false)
}

func waitForTCPPortToBeReady(port int, timeout time.Duration) error {
	ctx, _ := context.WithTimeout(context.TODO(), timeout)
	for {
		select {
		case <-ctx.Done():
			return errors.New(fmt.Sprintf("Wait for port %d to be ready timeout", port))
		default:
			<-time.Tick(1 * time.Second)
			b := ports.IsTCP4PortAvailable("0.0.0.0", port)
			if !b {
				return nil
			}
		}
	}
}

func NewDaemonClient(isSudoUser bool) (*DaemonClient, error) {
	var err error
	client := &DaemonClient{
		isSudo: isSudoUser,
	}
	if isSudoUser {
		client.daemonServerListenPort = daemon_common.SudoDaemonPort
	} else {
		client.daemonServerListenPort = daemon_common.DefaultDaemonPort
	}
	if ports.IsTCP4PortAvailable("0.0.0.0", client.daemonServerListenPort) {
		if err = runDaemonServer(isSudoUser); err != nil {
			return nil, err
		}
		if err = waitForTCPPortToBeReady(client.daemonServerListenPort, 10*time.Second); err != nil {
			return nil, err
		}
	}
	return client, nil
}

func (d *DaemonClient) SendStopDaemonServerCommand() error {
	cmd := &command.BaseCommand{CommandType: command.StopDaemonServer}
	bys, err := json.Marshal(cmd)
	if err != nil {
		return errors.Wrap(err, "")
	}
	return d.sendDataToDaemonServer(bys)
}

func (d *DaemonClient) SendPortForwardCommand(nhSvc *model.NocalHostResource, localPort, remotePort int, cmdType command.DaemonCommandType) error {

	startPFCmd := &command.PortForwardCommand{
		CommandType: cmdType,
		NameSpace:   nhSvc.NameSpace,
		AppName:     nhSvc.Application,
		Service:     nhSvc.Service,
		PodName:     nhSvc.PodName,
		LocalPort:   localPort,
		RemotePort:  remotePort,
	}

	bys, err := json.Marshal(startPFCmd)
	if err != nil {
		return errors.Wrap(err, "")
	}
	return d.sendDataToDaemonServer(bys)
}

func (d *DaemonClient) sendDataToDaemonServer(data []byte) error {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", d.daemonServerListenPort))
	if err != nil {
		return errors.Wrap(err, "")
	}
	defer conn.Close()
	_, err = conn.Write(data)
	return errors.Wrap(err, "")
}
