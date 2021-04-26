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

package daemon_client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"net"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/daemon_common"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/syncthing/daemon"
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"time"
)

type DaemonClient struct {
	isSudo                 bool
	daemonServerListenPort int
}

func StartDaemonServer(isSudoUser bool) error {
	nhctlPath, err := utils.GetNhctlPath()
	if err != nil {
		return err
	}
	daemonArgs := []string{nhctlPath, "daemon", "start"}
	if isSudoUser {
		daemonArgs = append(daemonArgs, "--sudo", "true")
	}
	return daemon.RunSubProcess(daemonArgs, nil, false)
}

func waitForTCPPortToBeReady(port int, timeout time.Duration) error {
	return waitTCPPort(false, port, timeout)
}

func waitTCPPort(available bool, port int, timeout time.Duration) error {
	ctx, _ := context.WithTimeout(context.TODO(), timeout)
	for {
		select {
		case <-ctx.Done():
			return errors.New(fmt.Sprintf("Wait for port %d to be ready timeout", port))
		default:
			<-time.Tick(1 * time.Second)
			b := ports.IsTCP4PortAvailable("0.0.0.0", port)
			if b == available {
				return nil
			}
		}
	}
}

func waitForTCPPortToBeDown(port int, timeout time.Duration) error {
	return waitTCPPort(true, port, timeout)
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

	if err = startDaemonServer(isSudoUser, client.daemonServerListenPort); err != nil {
		return nil, err
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

	if info.Version != daemon_common.Version || info.CommitId != daemon_common.CommitId {
		log.Log("Upgrading daemon server")
		// todo only use stop for v0.4.0
		utils.Should(client.SendStopDaemonServerCommand())
		utils.Should(waitForTCPPortToBeDown(client.daemonServerListenPort, 10*time.Second))
		if err = startDaemonServer(isSudoUser, client.daemonServerListenPort); err != nil {
			return nil, err
		}
		//utils.Should(client.SendRestartDaemonServerCommand())
	}
	return client, nil
}

func startDaemonServer(isSudoUser bool, port int) error {
	if ports.IsTCP4PortAvailable("0.0.0.0", port) {
		if err := StartDaemonServer(isSudoUser); err != nil {
			return err
		}
		log.Log("Waiting daemon to start")
		if err := waitForTCPPortToBeReady(port, 10*time.Second); err != nil {
			return err
		}
		log.Log("Daemon started")
	}
	return nil
}

func (d *DaemonClient) SendGetDaemonServerInfoCommand() ([]byte, error) {
	cmd := &command.BaseCommand{CommandType: command.GetDaemonServerInfo}
	bys, err := json.Marshal(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return d.sendAndWaitForResponse(bys)
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
	bys, err = d.sendAndWaitForResponse(bys)
	if err != nil {
		return err
	}
	log.Infof("%s", string(bys))
	return nil
}

func (d *DaemonClient) SendGetApplicationMetaCommand(ns, appName, kubeConfig string) (*appmeta.ApplicationMeta, error) {
	gamCmd := &command.GetApplicationMetaCommand{
		CommandType: command.GetApplicationMeta,
		NameSpace:   ns,
		AppName:     appName,
		KubeConfig:  kubeConfig,
	}

	bys, err := json.Marshal(gamCmd)
	if bys, err = d.sendAndWaitForResponse(bys); err != nil {
		return nil, err
	} else {
		meta := &appmeta.ApplicationMeta{}
		err := json.Unmarshal(bys, meta)
		return meta, err
	}
}

func (d *DaemonClient) SendGetApplicationMetasCommand(ns, kubeConfig string) ([]*appmeta.ApplicationMeta, error) {
	gamCmd := &command.GetApplicationMetasCommand{
		CommandType: command.GetApplicationMetas,
		NameSpace:   ns,
		KubeConfig:  kubeConfig,
	}

	bys, err := json.Marshal(gamCmd)
	if bys, err = d.sendAndWaitForResponse(bys); err != nil {
		return nil, err
	} else {
		var meta []*appmeta.ApplicationMeta
		err := json.Unmarshal(bys, &meta)
		return meta, err
	}
}

func (d *DaemonClient) SendStartPortForwardCommand(
	nhSvc *model.NocalHostResource, localPort, remotePort int, role string,
) error {

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
	if bys, err = d.sendAndWaitForResponse(bys); err != nil {
		return err
	} else {
		log.Infof("Response: %s", string(bys))
		return nil
	}
}

// SendStopPortForwardCommand send port forward to daemon
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
	if bys, err = d.sendAndWaitForResponse(bys); err != nil {
		return err
	} else {
		log.Infof("Response: %s", string(bys))
		return nil
	}
}

// sendDataToDaemonServer send data only to daemon
func (d *DaemonClient) sendDataToDaemonServer(data []byte) error {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", d.daemonServerListenPort))
	if err != nil {
		return errors.Wrap(err, "")
	}
	defer conn.Close()
	_, err = conn.Write(data)
	return errors.Wrap(err, "")
}

// sendAndWaitForResponse send data to daemon and wait for response
func (d *DaemonClient) sendAndWaitForResponse(data []byte) ([]byte, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", d.daemonServerListenPort))
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	defer conn.Close()
	if _, err = conn.Write(data); err != nil {
		return nil, errors.Wrap(err, "")
	}
	cw, ok := conn.(interface{ CloseWrite() error })
	if !ok {
		return nil, errors.Wrap(err, "Error to close write to daemon server ")
	}
	cw.CloseWrite()

	return ioutil.ReadAll(conn)
}
