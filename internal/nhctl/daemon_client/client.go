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
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"runtime/debug"
	"time"
)

type DaemonClient struct {
	isSudo                 bool
	daemonServerListenPort int
}

//func StartDaemonServer(isSudoUser bool) error {
//	var (
//		nhctlPath string
//		err       error
//	)
//	if utils.IsWindows() {
//		daemonDir, err := ioutil.TempDir("", "nhctl-daemon")
//		if err != nil {
//			return errors.Wrap(err, "")
//		}
//		// cp nhctl to daemonDir
//		err = utils.CopyFile(os.Args[0], filepath.Join(daemonDir, utils.GetNhctlBinName()))
//		if err != nil {
//			return errors.Wrap(err, "")
//		}
//		nhctlPath = filepath.Join(daemonDir, utils.GetNhctlBinName())
//	} else {
//		nhctlPath, err = utils.GetNhctlPath()
//		if err != nil {
//			return err
//		}
//	}
//	daemonArgs := []string{nhctlPath, "daemon", "start"}
//	if isSudoUser {
//		daemonArgs = append(daemonArgs, "--sudo", "true")
//	}
//	return daemon.RunSubProcess(daemonArgs, nil, false)
//}

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

	if err = startDaemonServerIfNotRunning(isSudoUser, client.daemonServerListenPort); err != nil {
		return nil, err
	}

	// Check Server's version
	daemonServerInfo, err := client.SendGetDaemonServerInfoCommand()
	if err != nil {
		return nil, err
	}

	if daemonServerInfo.Upgrading {
		return nil, errors.New("DaemonServer is upgrading. Please Try it later.")
	}

	nhctlPath, _ := utils.GetNhctlPath()

	// force update earlier version
	if daemonServerInfo.NhctlPath == "" {
		log.Log("Daemon server need to stop and restart")
		utils.Should(client.SendStopDaemonServerCommand())
		utils.Should(waitForTCPPortToBeDown(client.daemonServerListenPort, 10*time.Second))
		if err = startDaemonServerIfNotRunning(isSudoUser, client.daemonServerListenPort); err != nil {
			return nil, err
		}
	} else if daemonServerInfo.Version != daemon_common.Version || daemonServerInfo.CommitId != daemon_common.CommitId {

		// if from same nhctl
		if nhctlPath == daemonServerInfo.NhctlPath || utils.IsWindows() {
			log.Logf("Daemon server [%s] need to upgrade", daemonServerInfo.NhctlPath)
			utils.Should(client.SendRestartDaemonServerCommand())
		} else {
			// else do not update the daemon
			log.Logf(
				"Current nhctl [%s] but daemon server use [%s], "+
					"nocalhost will not update the daemon automatic.",
				nhctlPath, daemonServerInfo.NhctlPath,
			)
		}
	}
	return client, nil
}

func startDaemonServerIfNotRunning(isSudoUser bool, port int) error {
	if ports.IsTCP4PortAvailable("0.0.0.0", port) {
		if err := daemon_common.StartDaemonServerBySubProcess(isSudoUser); err != nil {
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

func (d *DaemonClient) SendGetDaemonServerInfoCommand() (*daemon_common.DaemonServerInfo, error) {
	cmd := &command.BaseCommand{CommandType: command.GetDaemonServerInfo, ClientStack: string(debug.Stack())}
	bys, err := json.Marshal(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	daemonServerInfo := &daemon_common.DaemonServerInfo{}
	if err := d.sendAndWaitForResponse(bys, daemonServerInfo); err != nil {
		return nil, err
	}
	return daemonServerInfo, err
}

// SendRestartDaemonServerCommand
// This command tells DaemonServer to run a newer version(by sub progress) with nhctl binary
// in ClientPath and then stops itself.
// By doing this, we can make sure newer DaemonServer have the same permission with older one.
// DaemonServer needs to copy nhctl.exe from ClientPath to a tmpDir in windows.
func (d *DaemonClient) SendRestartDaemonServerCommand() error {
	cmd := &command.BaseCommand{
		CommandType: command.RestartDaemonServer,
		ClientStack: string(debug.Stack()),
		ClientPath:  os.Args[0],
	}
	bys, err := json.Marshal(cmd)
	if err != nil {
		return errors.Wrap(err, "")
	}
	return d.sendDataToDaemonServer(bys)
}

func (d *DaemonClient) SendStopDaemonServerCommand() error {
	cmd := &command.BaseCommand{CommandType: command.StopDaemonServer, ClientStack: string(debug.Stack())}
	bys, err := json.Marshal(cmd)
	if err != nil {
		return errors.Wrap(err, "")
	}
	return d.sendDataToDaemonServer(bys)
}

func (d *DaemonClient) SendGetDaemonServerStatusCommand() (*daemon_common.DaemonServerStatusResponse, error) {
	cmd := &command.BaseCommand{CommandType: command.GetDaemonServerStatus, ClientStack: string(debug.Stack())}
	bys, err := json.Marshal(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	status := &daemon_common.DaemonServerStatusResponse{}
	if err = d.sendAndWaitForResponse(bys, status); err != nil {
		return nil, err
	}
	return status, nil
}

func (d *DaemonClient) SendGetApplicationMetaCommand(ns, appName, kubeConfigContent string) (*appmeta.ApplicationMeta, error) {
	gamCmd := &command.GetApplicationMetaCommand{
		CommandType: command.GetApplicationMeta,
		ClientStack: string(debug.Stack()),

		NameSpace:         ns,
		AppName:           appName,
		KubeConfigContent: kubeConfigContent,
	}

	bys, err := json.Marshal(gamCmd)

	meta := &appmeta.ApplicationMeta{}
	if err = d.sendAndWaitForResponse(bys, meta); err != nil {
		return nil, err
	}
	return meta, nil
}

func (d *DaemonClient) SendGetApplicationMetasCommand(ns, kubeConfig string) ([]*appmeta.ApplicationMeta, error) {
	gamCmd := &command.GetApplicationMetasCommand{
		CommandType: command.GetApplicationMetas,
		ClientStack: string(debug.Stack()),

		NameSpace:         ns,
		KubeConfigContent: kubeConfig,
	}

	bys, err := json.Marshal(gamCmd)

	var meta []*appmeta.ApplicationMeta
	if err = d.sendAndWaitForResponse(bys, &meta); err != nil {
		return nil, err
	}
	return meta, nil
}

func (d *DaemonClient) SendStartPortForwardCommand(
	nhSvc *model.NocalHostResource, localPort, remotePort int, role string,
) error {

	startPFCmd := &command.PortForwardCommand{
		CommandType: command.StartPortForward,
		ClientStack: string(debug.Stack()),

		NameSpace:   nhSvc.NameSpace,
		AppName:     nhSvc.Application,
		Service:     nhSvc.Service,
		ServiceType: nhSvc.ServiceType,
		PodName:     nhSvc.PodName,
		LocalPort:   localPort,
		RemotePort:  remotePort,
		Role:        role,
	}

	bys, err := json.Marshal(startPFCmd)
	if err != nil {
		return errors.Wrap(err, "")
	}

	return d.sendAndWaitForResponse(bys, nil)
}

// SendStopPortForwardCommand send port forward to daemon
func (d *DaemonClient) SendStopPortForwardCommand(nhSvc *model.NocalHostResource, localPort, remotePort int) error {

	startPFCmd := &command.PortForwardCommand{
		CommandType: command.StopPortForward,
		ClientStack: string(debug.Stack()),

		NameSpace:   nhSvc.NameSpace,
		AppName:     nhSvc.Application,
		Service:     nhSvc.Service,
		ServiceType: nhSvc.ServiceType,
		PodName:     nhSvc.PodName,
		LocalPort:   localPort,
		RemotePort:  remotePort,
	}

	bys, err := json.Marshal(startPFCmd)
	if err != nil {
		return errors.Wrap(err, "")
	}

	return d.sendAndWaitForResponse(bys, nil)
}

// SendGetAllInfoCommand send get resource info request to daemon
func (d *DaemonClient) SendGetResourceInfoCommand(
	kubeconfig,
	ns,
	appName,
	resource,
	resourceName string,
	label map[string]string,
) (interface{}, error) {
	cmd := &command.GetResourceInfoCommand{
		CommandType: command.GetResourceInfo,
		ClientStack: string(debug.Stack()),

		KubeConfig:   kubeconfig,
		Namespace:    ns,
		AppName:      appName,
		Resource:     resource,
		ResourceName: resourceName,
		Label:        label,
	}

	bys, err := json.Marshal(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	var result interface{}
	if err := d.sendAndWaitForResponse(bys, &result); err != nil {
		return nil, err
	}
	return result, nil
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
func (d *DaemonClient) sendAndWaitForResponse(req []byte, resp interface{}) error {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", d.daemonServerListenPort))
	if err != nil {
		return errors.Wrap(err, "")
	}
	defer conn.Close()
	if _, err = conn.Write(req); err != nil {
		return errors.Wrap(err, "")
	}
	cw, ok := conn.(interface{ CloseWrite() error })
	if !ok {
		return errors.Wrap(err, "Error to close write to daemon server ")
	}
	cw.CloseWrite()

	respBytes, err := ioutil.ReadAll(conn)
	if err != nil {
		return errors.Wrap(err, "")
	}

	response := command.BaseResponse{}
	if err := json.Unmarshal(respBytes, &response); err != nil {
		return errors.Wrap(err, "")
	}

	switch response.Status {
	case command.SUCCESS:

		// do nothing
	case command.PREVIEW_VERSION:

		// may from elder version
		if err := json.Unmarshal(respBytes, &resp); err == nil {
			return nil
		}
	default:

		return errors.New(
			fmt.Sprintf(
				"Error occur from daemon, status [%d], msg [%s].",
				response.Status, response.Msg,
			),
		)
	}

	if resp == nil {
		return nil
	}

	if err := json.Unmarshal(response.Data, resp); err != nil {
		return errors.Wrap(err, "")
	}

	return nil
}
