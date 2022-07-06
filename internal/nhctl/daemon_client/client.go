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
	"io"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"net"
	"nocalhost/internal/nhctl/daemon_common"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"runtime/debug"
	"sync"
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

var (
	client     *DaemonClient
	sudoClient *DaemonClient
	lock       sync.Mutex
)

func GetDaemonClient(isSudoUser bool) (*DaemonClient, error) {
	var (
		c   *DaemonClient
		err error
	)

	if c, err = getCachedDaemonClient(isSudoUser); err == nil {
		return c, nil
	}

	if c, err = newDaemonClient(isSudoUser); err != nil {
		return nil, err
	}

	lock.Lock()
	defer lock.Unlock()
	if isSudoUser {
		if sudoClient == nil {
			sudoClient = c
		}
		return sudoClient, nil
	}
	if client == nil {
		client = c
	}
	return client, nil
}

func getCachedDaemonClient(isSudoUser bool) (*DaemonClient, error) {
	lock.Lock()
	defer lock.Unlock()
	if !isSudoUser && client != nil {
		return client, nil
	}
	if isSudoUser && sudoClient != nil {
		return sudoClient, nil
	}
	return nil, errors.New("No cached daemon client")
}

func newDaemonClient(isSudoUser bool) (*DaemonClient, error) {
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

func (d *DaemonClient) SendCheckClusterStatusCommand(kubeContent string) (*daemon_common.CheckClusterStatus, error) {
	cmd := &command.CheckClusterStatusCommand{
		CommandType:       command.CheckClusterStatus,
		ClientStack:       string(debug.Stack()),
		KubeConfigContent: kubeContent,
	}

	bys, err := json.Marshal(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	r := &daemon_common.CheckClusterStatus{}
	if err := d.sendAndWaitForResponse(bys, r); err != nil {
		return nil, err
	}
	return r, err
}

func (d *DaemonClient) SendFlushDirMappingCacheCommand(ns, nid, appName string) error {
	cmd := &command.InvalidCacheCommand{
		CommandType: command.FlushDirMappingCache,
		ClientStack: string(debug.Stack()),
		Namespace:   ns,
		Nid:         nid,
		AppName:     appName,
	}

	bys, err := json.Marshal(cmd)
	if err != nil {
		return errors.Wrap(err, "")
	}
	return d.sendAndWaitForResponse(bys, nil)
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

func (d *DaemonClient) SendAuthCheckCommand(ns, kubeConfigContent string, needChecks ...string) (bool, error) {
	acCmd := &command.AuthCheckCommand{
		CommandType: command.AuthCheck,
		ClientStack: string(debug.Stack()),

		NameSpace:         ns,
		KubeConfigContent: kubeConfigContent,
		NeedChecks:        needChecks,
	}

	bys, err := json.Marshal(acCmd)

	var nothing interface{}
	if err = d.sendAndWaitForResponse(bys, &nothing); err != nil {
		return false, err
	}
	return true, nil
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
	nhSvc *model.NocalHostResource, localPort, remotePort int, role, nid string,
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
		Nid:         nid,
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
		Nid:         nhSvc.Nid,
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
	showHidden bool,
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
		ShowHidden:   showHidden,
	}

	bys, err := json.Marshal(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	var result interface{}
	if err = d.sendAndWaitForResponse(bys, &result); err != nil {
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

// SendKubeconfigOperationCommand send add/remove kubeconfig request to daemon
func (d *DaemonClient) SendKubeconfigOperationCommand(kubeconfigBytes []byte, ns string, operation command.Operation) error {
	cmd := &command.KubeconfigOperationCommand{
		CommandType: command.KubeconfigOperation,
		ClientStack: string(debug.Stack()),

		KubeConfigBytes: kubeconfigBytes,
		Namespace:       ns,
		Operation:       operation,
	}
	bys, err := json.Marshal(cmd)
	if err != nil {
		return errors.Wrap(err, "")
	}
	return d.sendDataToDaemonServer(bys)
}

func (d *DaemonClient) SendVPNOperateCommand(
	kubeconfig,
	ns string,
	operation command.VPNOperation,
	workloads string,
	consumer func(io.Reader) error,
) error {
	cmd := &command.VPNOperateCommand{
		CommandType: command.VPNOperate,
		ClientStack: string(debug.Stack()),

		KubeConfig: kubeconfig,
		Namespace:  ns,
		Action:     operation,
		Resource:   workloads,
	}
	bys, err := json.Marshal(cmd)
	if err != nil {
		return errors.Wrap(err, "")
	}
	return d.sendAndWaitForStream(bys, consumer)
}

func (d *DaemonClient) SendSudoVPNOperateCommand(
	kubeconfig, ns string,
	operation command.VPNOperation,
	consumer func(io.Reader) error,
) error {
	cmd := &command.VPNOperateCommand{
		CommandType: command.SudoVPNOperate,
		ClientStack: string(debug.Stack()),

		KubeConfig: kubeconfig,
		Namespace:  ns,
		Action:     operation,
	}
	bys, err := json.Marshal(cmd)
	if err != nil {
		return errors.Wrap(err, "")
	}
	return d.sendAndWaitForStream(bys, consumer)
}

func (d *DaemonClient) SendVPNStatusCommand() (interface{}, error) {
	cmd := &command.VPNOperateCommand{
		CommandType: command.VPNStatus,
		ClientStack: string(debug.Stack()),
	}
	bys, err := json.Marshal(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	var result interface{}
	if err = d.sendAndWaitForResponse(bys, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (d *DaemonClient) SendSudoVPNStatusCommand() (interface{}, error) {
	cmd := &command.VPNOperateCommand{
		CommandType: command.SudoVPNStatus,
		ClientStack: string(debug.Stack()),
	}
	bys, err := json.Marshal(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	var result interface{}
	if err = d.sendAndWaitForResponse(bys, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// sendDataToDaemonServer send data only to daemon
func (d *DaemonClient) sendDataToDaemonServer(data []byte) error {
	baseCmd := command.BaseCommand{}
	err := json.Unmarshal(data, &baseCmd)
	if err != nil {
		return errors.Wrap(err, "Failed to unmarshal command")
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", d.daemonServerListenPort), time.Second*30)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("%s failed to dial to daemon", baseCmd.CommandType))
	}
	defer conn.Close()
	if err = conn.SetDeadline(time.Now().Add(time.Second * 30)); err != nil {
		log.Logf("set connection deadline, err: %v", err)
	}
	if err = conn.SetWriteDeadline(time.Now().Add(time.Second * 30)); err != nil {
		log.Logf("set connection write deadline, err: %v", err)
	}
	_, err = conn.Write(data)
	return errors.Wrap(err, fmt.Sprintf("%s failed to write to daemon", baseCmd.CommandType))
}

// sendAndWaitForResponse send data to daemon and wait for response
func (d *DaemonClient) sendAndWaitForStream(req []byte, consumer func(io.Reader) error) error {
	var conn net.Conn
	var err error
	baseCmd := command.BaseCommand{}
	err = json.Unmarshal(req, &baseCmd)
	if err != nil {
		return errors.Wrap(err, "Failed to unmarshal command")
	}
	conn, err = net.DialTimeout("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", d.daemonServerListenPort), time.Second*30)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("%s failed to dial to daemon", baseCmd.CommandType))
	}
	defer conn.Close()
	if _, err = conn.Write(req); err != nil {
		return errors.Wrap(err, fmt.Sprintf("%s failed to write to daemon", baseCmd.CommandType))
	}
	cw, ok := conn.(interface{ CloseWrite() error })
	if !ok {
		return errors.New(fmt.Sprintf("%s failed to close write to daemon server", baseCmd.CommandType))
	}
	log.WrapAndLogE(cw.CloseWrite())
	if consumer != nil {
		return consumer(conn)
	}
	return nil
}
func (d *DaemonClient) sendAndWaitForResponse(req []byte, resp interface{}) error {
	var (
		conn net.Conn
		err  error
	)
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	baseCmd := command.BaseCommand{}
	err = json.Unmarshal(req, &baseCmd)
	if err != nil {
		return errors.Wrap(err, "Failed to unmarshal command")
	}
	conn, err = net.DialTimeout("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", d.daemonServerListenPort), time.Second*30)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("%s failed to dial to daemon", baseCmd.CommandType))
	}

	if _, err = conn.Write(req); err != nil {
		return errors.Wrap(err, fmt.Sprintf("%s failed to write to daemon", baseCmd.CommandType))
	}
	cw, ok := conn.(interface{ CloseWrite() error })
	if !ok {
		return errors.New(fmt.Sprintf("%s failed to close write to daemon server", baseCmd.CommandType))
	}
	log.WrapAndLogE(cw.CloseWrite())

	errChan := make(chan error, 0)
	var respBytes []byte

	go func() {
		var err error
		respBytes, err = ioutil.ReadAll(conn)
		errChan <- err
	}()

	select {
	case err = <-errChan:
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("%s failed to get response from daemon", baseCmd.CommandType))
		}
	case <-time.After(30 * time.Second):
		return errors.New(fmt.Sprintf("%s waited response timeout after 30s", baseCmd.CommandType))

	}

	if len(respBytes) == 0 {
		return errors.New(fmt.Sprintf("Daemon responses empty data to %s", baseCmd.CommandType))
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
		} else {
			log.WrapAndLogE(err)
		}
	default:
		err = errors.New(
			fmt.Sprintf("Error occur from daemon, status [%d], msg [%s].", response.Status, response.Msg),
		)
		return err
	}

	if resp == nil {
		return nil
	}

	if len(response.Data) == 0 {
		return nil
	}
	if err := json.Unmarshal(response.Data, resp); err != nil {
		return errors.Wrap(err, fmt.Sprintf("%s failed to unmarshal response data", baseCmd.CommandType))
	}

	return nil
}
