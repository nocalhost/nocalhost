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
	"nocalhost/pkg/nhctl/log"
	"os/exec"
	"time"
)

type DaemonClient struct {
	isSudo                 bool
	daemonServerListenPort int
}

func StartDaemonServer(isSudoUser bool) error {
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

func CheckIfDaemonServerRunning(isSudoUser bool) bool {
	var listenPort int
	if isSudoUser {
		listenPort = daemon_common.SudoDaemonPort
	} else {
		listenPort = daemon_common.DefaultDaemonPort
	}
	return !ports.IsTCP4PortAvailable("0.0.0.0", listenPort)
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
		if err = StartDaemonServer(isSudoUser); err != nil {
			return nil, err
		}
		if err = waitForTCPPortToBeReady(client.daemonServerListenPort, 10*time.Second); err != nil {
			return nil, err
		}
	}
	// Check Server's version
	bys, err := client.SendGetDaemonServerInfoCommand()
	if err != nil {
		return nil, err
	}

	info := &daemon_common.DaemonServerInfo{}
	err = json.Unmarshal(bys, info)
	if err != nil {
		return nil, err
	}

	if info.Version != daemon_common.Version {
		log.Log("Upgrading daemon server")
		err = client.SendRestartDaemonServerCommand()
		if err != nil {
			log.WarnE(err, "Failed to upgrade daemon server")
		}
	} else {
		//log.Log("No need to upgrade daemon server")
	}
	return client, nil
}

func (d *DaemonClient) SendGetDaemonServerInfoCommand() ([]byte, error) {
	cmd := &command.BaseCommand{CommandType: command.GetDaemonServerInfo}
	bys, err := json.Marshal(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return d.sendDataToDaemonServerAndWaitForResponse(bys)
}

func (d *DaemonClient) SendRestartDaemonServerCommand() error {
	cmd := &command.BaseCommand{CommandType: command.RestartDaemonServer}
	bys, err := json.Marshal(cmd)
	if err != nil {
		return errors.Wrap(err, "")
	}
	return d.sendDataToDaemonServer(bys)
}

func (d *DaemonClient) SendStopDaemonServerCommand() error {
	cmd := &command.BaseCommand{CommandType: command.StopDaemonServer}
	bys, err := json.Marshal(cmd)
	if err != nil {
		return errors.Wrap(err, "")
	}
	return d.sendDataToDaemonServer(bys)
}

func (d *DaemonClient) SendGetDaemonServerStatusCommand() error {
	cmd := &command.BaseCommand{CommandType: command.GetDaemonServerStatus}
	bys, err := json.Marshal(cmd)
	if err != nil {
		return errors.Wrap(err, "")
	}
	bys, err = d.sendDataToDaemonServerAndWaitForResponse(bys)
	if err != nil {
		return err
	}
	log.Infof("%s", string(bys))
	return nil
}

func (d *DaemonClient) SendStartPortForwardCommand(nhSvc *model.NocalHostResource, localPort, remotePort int, role string) error {

	startPFCmd := &command.PortForwardCommand{
		CommandType: command.StartPortForward,
		NameSpace:   nhSvc.NameSpace,
		AppName:     nhSvc.Application,
		Service:     nhSvc.Service,
		PodName:     nhSvc.PodName,
		LocalPort:   localPort,
		RemotePort:  remotePort,
		Role:        role,
	}

	bys, err := json.Marshal(startPFCmd)
	if err != nil {
		return errors.Wrap(err, "")
	}
	if bys, err = d.sendDataToDaemonServerAndWaitForResponse(bys); err != nil {
		return err
	} else {
		log.Infof("Response: %s", string(bys))
		return nil
	}
}

func (d *DaemonClient) SendStopPortForwardCommand(nhSvc *model.NocalHostResource, localPort, remotePort int) error {

	startPFCmd := &command.PortForwardCommand{
		CommandType: command.StopPortForward,
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
	if bys, err = d.sendDataToDaemonServerAndWaitForResponse(bys); err != nil {
		return err
	} else {
		log.Infof("Response: %s", string(bys))
		return nil
	}
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

func (d *DaemonClient) sendDataToDaemonServerAndWaitForResponse(data []byte) ([]byte, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", d.daemonServerListenPort))
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	defer conn.Close()
	if _, err = conn.Write(data); err != nil {
		return nil, errors.Wrap(err, "")
	}
	bys := make([]byte, 4096)
	n, err := conn.Read(bys)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return bys[0:n], nil
}
