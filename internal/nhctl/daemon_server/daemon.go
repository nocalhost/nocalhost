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
	"net"
	"nocalhost/internal/nhctl/daemon_common"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/syncthing/daemon"
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"os/exec"
	"strings"
)

var (
	isSudo                      = false
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

func StartDaemon(isSudoUser bool) error {
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

	// Recovering port forward
	if err = pfManager.RecoverAllPortForward(); err != nil {
		log.LogE(err)
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
			rBytes := make([]byte, 2048)
			n, err := conn.Read(rBytes)
			if err != nil {
				log.LogE(errors.Wrap(err, ""))
				continue
			}
			rBytes = rBytes[0:n]
			cmdType, err := command.ParseCommandType(rBytes)
			if err != nil {
				log.LogE(err)
				continue
			}
			go handleCommand(conn, rBytes, cmdType)
		}
	}()

	for {
		select {
		case <-tcpCtx.Done():
			log.Log("Stop listening tcp port for daemon server")
			_ = listener.Close()
		case <-daemonCtx.Done():
			log.Log("Exit daemon server")
			return nil
		}
	}
}

func handleCommand(conn net.Conn, bys []byte, cmdType command.DaemonCommandType) {
	defer conn.Close()
	var err error
	log.Infof("Handling %s command", cmdType)
	switch cmdType {
	case command.StartPortForward:
		startCmd := &command.PortForwardCommand{}
		if err = json.Unmarshal(bys, startCmd); err != nil {
			log.LogE(errors.Wrap(err, ""))
			return
		}
		if err = handleStartPortForwardCommand(startCmd); err != nil {
			log.LogE(err)
		}
	case command.StopPortForward:
		pfCmd := &command.PortForwardCommand{}
		if err = json.Unmarshal(bys, pfCmd); err != nil {
			log.LogE(errors.Wrap(err, ""))
			return
		}
		if err = handleStopPortForwardCommand(pfCmd); err != nil {
			log.LogE(err)
		}
	case command.StopDaemonServer:
		tcpCancelFunc()
		// todo: clean up resources
		daemonCancelFunc()

	case command.RestartDaemonServer:
		if err = handlerRestartDaemonServerCommand(isSudo); err != nil {
			log.LogE(err)
			return
		}
		log.Log("New daemon server is starting, exit this one")
		daemonCancelFunc()
	case command.GetDaemonServerInfo:
		info := daemon_common.NewDaemonServerInfo()
		bys, err := json.Marshal(info)
		if err != nil {
			log.Log(errors.Wrap(err, ""))
		}
		if _, err = conn.Write(bys); err != nil {
			log.LogE(errors.Wrap(err, ""))
		} else {
			log.Log("Daemon Server info has been return")
		}
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

// If a port-forward exists, stop it first, and then start it
// If a port-forward doesn't exist, start it
//func handleRestartPortForwardCommand(startCmd *command.PortForwardCommand) error {
//	// Check if port forward already exists
//	list, err := nocalhost.ListPortForward(startCmd.NameSpace, startCmd.AppName, startCmd.Service)
//	if err != nil {
//		return err
//	}
//	for _, pf := range list {
//		if pf.LocalPort == startCmd.LocalPort && pf.RemotePort == startCmd.RemotePort {
//			// Stop it
//		}
//	}
//}

// If a port-forward already exist, skip it(don't do anything), and return an error
func handleStartPortForwardCommand(startCmd *command.PortForwardCommand) error {
	return pfManager.StartPortForwardGoRoutine(&model.NocalHostResource{
		NameSpace:   startCmd.NameSpace,
		Application: startCmd.AppName,
		Service:     startCmd.Service,
		PodName:     startCmd.PodName,
	}, startCmd.LocalPort, startCmd.RemotePort, true)
}
