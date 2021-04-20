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
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"net"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/appmeta_manager"
	"nocalhost/internal/nhctl/daemon_common"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/syncthing/daemon"
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"os/exec"
	"runtime/debug"
	"strings"
)

var (
	isSudo                      = false
	version                     = "1.0"
	pfManager                   *PortForwardManager
	tcpCtx, tcpCancelFunc       = context.WithCancel(context.Background()) // For stopping listening tcp port
	daemonCtx, daemonCancelFunc = context.WithCancel(context.Background()) // For exiting current daemon server
)

func init() {
	pfManager = NewPortForwardManager()
}

func daemonListenPort() int {
	if isSudo {
		return daemon_common.SudoDaemonPort
	}
	return daemon_common.DefaultDaemonPort
}

func StartDaemon(isSudoUser bool, v string) error {
	version = v
	if isSudoUser && !utils.IsSudoUser() {
		return errors.New("Failed to start daemon server with sudo")
	}
	isSudo = isSudoUser // Mark daemon server if it is run as sudo
	ports.IsPortAvailable("0.0.0.0", daemonListenPort())
	address := fmt.Sprintf("%s:%d", "0.0.0.0", daemonListenPort())
	listener, err := net.Listen("tcp4", address)
	if err != nil {
		return errors.Wrap(err, "")
	}

	// run the dev event listener
	if !isSudoUser {
		appmeta_manager.Init()
		appmeta_manager.RegisterListener(func(pack *appmeta_manager.ApplicationEventPack) error {
			kubeconfig, err := nocalhost.GetKubeConfigFromProfile(pack.Ns, pack.AppName)
			if err != nil {
				return nil
			}
			nhApp, err := app.NewApplication(pack.AppName, pack.Ns, kubeconfig, false)
			if err != nil {
				return nil
			}
			if pack.Event.EventType == appmeta.DEV_END {
				log.Logf("Receive dev end event, stopping sync and pf for %s-%s-%s", pack.Ns, pack.AppName, pack.Event.ResourceName)
				if err := nhApp.StopSyncAndPortForwardProcess(pack.Event.ResourceName, true); err != nil {
					return nil
				}
			} else if pack.Event.EventType == appmeta.DEV_STA {
				profile, _ := nhApp.GetProfile()

				// ignore the event from local
				if profile.Identifier == pack.Event.Identifier {
					return nil
				}

				log.Logf("Receive dev start event, stopping pf for %s-%s-%s", pack.Ns, pack.AppName, pack.Event.ResourceName)
				if err := nhApp.StopAllPortForward(pack.Event.ResourceName); err != nil {
					return nil
				}
			}
			return nil
		})
		appmeta_manager.Start()
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if strings.Contains(strings.ToLower(err.Error()), "use of closed network connection") {
					log.Logf("Port %d has been closed", daemonListenPort())
					return
				}
				log.LogE(errors.Wrap(err, ""))
				continue
			}

			bytes, err := ioutil.ReadAll(conn)
			cmdType, err := command.ParseCommandType(bytes)
			if err != nil {
				log.LogE(err)
				continue
			}
			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Fatalf("DAEMON-RECOVER: %s", string(debug.Stack()))
					}
				}()
				handleCommand(conn, bytes, cmdType)
			}()
		}
	}()

	go func() {
		select {
		case <-tcpCtx.Done():
			log.Log("Stop listening tcp port for daemon server")
			_ = listener.Close()
		}
	}()

	// Recovering port forward
	if err = pfManager.RecoverAllPortForward(); err != nil {
		log.LogE(err)
	}

	// Recovering syncthing
	// nhctl sync bookinfo -d productpage --resume --kubeconfig /Users/xxx/.nh/plugin/kubeConfigs/293_config
	if err = recoverSyncthing(); err != nil {
		log.LogE(err)
	}

	select {
	case <-daemonCtx.Done():
		log.Log("Exit daemon server")
		return nil
	}
}

func handleCommand(conn net.Conn, bys []byte, cmdType command.DaemonCommandType) {
	var err error
	log.Infof("Handling %s command", cmdType)
	switch cmdType {
	case command.StartPortForward:
		startCmd := &command.PortForwardCommand{}
		errInfo := ""
		if err = json.Unmarshal(bys, startCmd); err != nil {
			log.LogE(errors.Wrap(err, ""))
			response(conn, &daemon_common.CommonResponse{ErrInfo: err.Error()})
			return
		}
		if err = handleStartPortForwardCommand(startCmd); err != nil {
			errInfo = err.Error()
			log.LogE(err)
		}
		response(conn, &daemon_common.CommonResponse{ErrInfo: errInfo})
	case command.StopPortForward:
		pfCmd := &command.PortForwardCommand{}
		errInfo := ""
		if err = json.Unmarshal(bys, pfCmd); err != nil {
			log.LogE(errors.Wrap(err, ""))
			response(conn, &daemon_common.CommonResponse{ErrInfo: err.Error()})
			return
		}
		if err = handleStopPortForwardCommand(pfCmd); err != nil {
			log.LogE(err)
			errInfo = err.Error()
		}
		response(conn, &daemon_common.CommonResponse{ErrInfo: errInfo})
	case command.StopDaemonServer:
		conn.Close()
		tcpCancelFunc()
		// todo: clean up resources
		daemonCancelFunc()
	case command.RestartDaemonServer:
		conn.Close()
		if err = handlerRestartDaemonServerCommand(isSudo); err != nil {
			log.LogE(err)
			return
		}
		log.Log("New daemon server is starting, exit this one")
		daemonCancelFunc()
	case command.GetDaemonServerInfo:
		info := &daemon_common.DaemonServerInfo{Version: version}
		response(conn, info)
	case command.GetDaemonServerStatus:
		status := &daemon_common.DaemonServerStatusResponse{PortForwardList: pfManager.ListAllRunningPortForwardGoRoutineProfile()}
		response(conn, status)
	case command.GetApplicationMeta:
		gamCmd := &command.GetApplicationMetaCommand{}
		if err = json.Unmarshal(bys, gamCmd); err != nil {
			log.LogE(errors.Wrap(err, ""))
			response(conn, &daemon_common.CommonResponse{ErrInfo: err.Error()})
			return
		}

		meta := appmeta_manager.GetApplicationMeta(gamCmd.NameSpace, gamCmd.AppName, gamCmd.KubeConfig)
		response(conn, meta)

	case command.GetApplicationMetas:
		gamsCmd := &command.GetApplicationMetasCommand{}
		if err = json.Unmarshal(bys, gamsCmd); err != nil {
			log.LogE(errors.Wrap(err, ""))
			response(conn, &daemon_common.CommonResponse{ErrInfo: err.Error()})
			return
		}

		metas := appmeta_manager.GetApplicationMetas(gamsCmd.NameSpace, gamsCmd.KubeConfig)
		response(conn, metas)
	}
}

func response(conn net.Conn, v interface{}) {
	defer conn.Close()
	bys, err := json.Marshal(v)
	if err != nil {
		log.LogE(errors.Wrap(err, ""))
	}
	if _, err = conn.Write(bys); err != nil {
		log.LogE(errors.Wrap(err, ""))
	}
	cw, ok := conn.(interface{ CloseWrite() error })
	if ok {
		cw.CloseWrite()
	}
}

func handlerRestartDaemonServerCommand(isSudoUser bool) error {
	tcpCancelFunc() // Stop listening tcp port
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

func handleStopPortForwardCommand(cmd *command.PortForwardCommand) error {
	// For compatibility
	return pfManager.StopPortForwardGoRoutine(cmd.LocalPort, cmd.RemotePort)
}

// If a port-forward already exist, skip it(don't do anything), and return an error
func handleStartPortForwardCommand(startCmd *command.PortForwardCommand) error {
	return pfManager.StartPortForwardGoRoutine(startCmd, true)
}
