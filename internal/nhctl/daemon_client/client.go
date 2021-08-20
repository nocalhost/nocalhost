/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package daemon_client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"net"
	"nocalhost/internal/nhctl/daemon_common"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	utils2 "nocalhost/pkg/nhctl/utils"
	"os"
	"runtime/debug"
	"time"
)

type DaemonClient struct {
	isSudo                 bool
	daemonServerListenPort int
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
// By doing this, we can make sure newer DaemonServer has the same permission with older one.
// DaemonServer needs to copy nhctl.exe from ClientPath to a tmpDir in Windows OS.
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

// the reason why return a interface is applicationMeta needs to using this client,
// otherwise it will cause cycle import
func (d *DaemonClient) SendGetApplicationMetaCommand(ns, appName, kubeConfigContent string) (interface{}, error) {
	gamCmd := &command.GetApplicationMetaCommand{
		CommandType: command.GetApplicationMeta,
		ClientStack: string(debug.Stack()),

		NameSpace:         ns,
		AppName:           appName,
		KubeConfigContent: kubeConfigContent,
	}

	bys, err := json.Marshal(gamCmd)

	var meta interface{}
	if err = d.sendAndWaitForResponse(bys, &meta); err != nil {
		return nil, err
	}
	return meta, nil
}

// the reason why return a interface array is applicationMeta needs to using this client,
// otherwise it will cause cycle import
func (d *DaemonClient) SendGetApplicationMetasCommand(ns, kubeConfig string) ([]interface{}, error) {
	gamCmd := &command.GetApplicationMetasCommand{
		CommandType: command.GetApplicationMetas,
		ClientStack: string(debug.Stack()),

		NameSpace:         ns,
		KubeConfigContent: kubeConfig,
	}

	bys, err := json.Marshal(gamCmd)

	var meta []interface{}
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

// SendGetAllInfoCommand send get resource info request to daemon
func (d *DaemonClient) SendUpdateApplicationMetaCommand(
	kubeconfig,
	ns,
	secretName string,
	secret *v1.Secret,
) (bool, error) {
	cmd := &command.UpdateApplicationMetaCommand{
		CommandType: command.UpdateApplicationMeta,
		ClientStack: string(debug.Stack()),

		KubeConfig: kubeconfig,
		Namespace:  ns,
		Secret:     secret,
		SecretName: secretName,
	}

	bys, err := json.Marshal(cmd)
	if err != nil {
		return false, errors.Wrap(err, "")
	}

	var result interface{}
	if err := d.sendAndWaitForResponse(bys, &result); err != nil {
		return false, err
	}
	return true, nil
}

// sendDataToDaemonServer send data only to daemon
func (d *DaemonClient) sendDataToDaemonServer(data []byte) error {
	baseCmd := command.BaseCommand{}
	err := json.Unmarshal(data, &baseCmd)
	if err == nil {
		log.Logf("Sending %v command", baseCmd.CommandType)
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", d.daemonServerListenPort), time.Second*30)
	if err != nil {
		log.WrapAndLogE(err)
		return errors.Wrap(err, "")
	}
	defer conn.Close()
	_, err = conn.Write(data)
	log.WrapAndLogE(err)
	return errors.Wrap(err, "")
}

// sendAndWaitForResponse send data to daemon and wait for response
func (d *DaemonClient) sendAndWaitForResponse(req []byte, resp interface{}) error {
	baseCmd := command.BaseCommand{}
	err := json.Unmarshal(req, &baseCmd)
	if err == nil {
		log.Logf("Sending %v command", baseCmd.CommandType)
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", d.daemonServerListenPort), time.Second*30)
	if err != nil {
		log.WrapAndLogE(err)
		return utils2.WrapErr(err)
	}
	defer conn.Close()
	if _, err = conn.Write(req); err != nil {
		log.WrapAndLogE(err)
		return utils2.WrapErr(err)
	}
	cw, ok := conn.(interface{ CloseWrite() error })
	if !ok {
		err = errors.New("Error to close write to daemon server")
		log.LogE(err)
		return err
	}
	log.WrapAndLogE(cw.CloseWrite())

	respBytes, err := ioutil.ReadAll(conn)
	if err != nil {
		log.WrapAndLogE(err)
		return errors.Wrap(err, "")
	}

	response := command.BaseResponse{}
	if err := json.Unmarshal(respBytes, &response); err != nil {
		log.WrapAndLogE(err)
		return errors.Wrap(err, "")
	}

	switch response.Status {
	case command.SUCCESS:

		// do nothing
	case command.PREVIEW_VERSION:

		// may from elder version
		if err := json.Unmarshal(respBytes, &resp); err == nil {
			return nil
		}else {
			log.WrapAndLogE(err)
		}
	default:
		err = errors.New(
			fmt.Sprintf("Error occur from daemon, status [%d], msg [%s].", response.Status, response.Msg ),
		)
		log.LogE(err)
		return err
	}

	if resp == nil {
		return nil
	}

	if len(response.Data) == 0 {
		return nil
	}
	if err := json.Unmarshal(response.Data, resp); err != nil {
		log.WrapAndLogE(err)
		return errors.Wrap(err, "")
	}

	return nil
}
