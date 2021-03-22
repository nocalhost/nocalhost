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
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"net"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/daemon_common"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"strings"
	"time"
)

var (
	isSudo    = false
	pfManager *PortForwardManager
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
	if isSudoUser {
		if !utils.IsSudoUser() {
			return errors.New("Failed to start daemon server with sudo")
		}
	}
	isSudo = isSudoUser
	ports.IsPortAvailable("0.0.0.0", daemonListenPort())
	address := fmt.Sprintf("%s:%d", "0.0.0.0", daemonListenPort())
	listener, err := net.Listen("tcp4", address)
	if err != nil {
		return errors.Wrap(err, "")
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.WarnE(errors.Wrap(err, ""), "")
			continue
		}
		rBytes := make([]byte, 2048)
		_, err = conn.Read(rBytes)
		conn.Close()
		if err != nil {
			log.LogE(errors.Wrap(err, ""))
			continue
		}
		cmdType, err := command.ParseCommandType(rBytes)
		if err != nil {
			log.LogE(err)
			continue
		}
		go handleCommand(rBytes, cmdType)
	}

	defer listener.Close()
	return nil
}

func handleCommand(bys []byte, cmdType command.DaemonCommandType) {
	log.Infof("Handling %s command", cmdType)
	switch cmdType {
	case command.StartPortForward:
		startCmd := &command.PortForwardCommand{}
		err := json.Unmarshal(bys, startCmd)
		if err != nil {
			log.LogE(errors.Wrap(err, ""))
			return
		}
		err = handleStartPortForwardCommand(startCmd)
		if err != nil {
			log.LogE(err)
		}
	case command.RestartPortForward:
		restartCmd := &command.PortForwardCommand{}
		err := json.Unmarshal(bys, restartCmd)
		if err != nil {
			log.LogE(errors.Wrap(err, ""))
		}
		err = handleRestartPortForwardCommand(restartCmd)
		if err != nil {
			log.LogE(err)
		}
	case command.StopPortForward:
		pfCmd := &command.PortForwardCommand{}
		err := json.Unmarshal(bys, pfCmd)
		if err != nil {
			log.LogE(errors.Wrap(err, ""))
			return
		}

	}

}

func handleStopPortForwardCommand(cmd *command.PortForwardCommand) error {
	list, err := nocalhost.ListPortForward(cmd.NameSpace, cmd.AppName, cmd.Service)
	if err != nil {
		return err
	}
	for _, pf := range list {
		if pf.LocalPort == cmd.LocalPort && pf.RemotePort == cmd.RemotePort {
			// Stop it
			// Find port-forward go routine
		}
	}
}

// If a port-forward exists, stop it first, and then start it
// If a port-forward doesn't exist, start it
func handleRestartPortForwardCommand(startCmd *command.PortForwardCommand) error {
	// Check if port forward already exists
	list, err := nocalhost.ListPortForward(startCmd.NameSpace, startCmd.AppName, startCmd.Service)
	if err != nil {
		return err
	}
	for _, pf := range list {
		if pf.LocalPort == startCmd.LocalPort && pf.RemotePort == startCmd.RemotePort {
			// Stop it
		}
	}
}

// If a port-forward already exist, skip it(don't do anything), and return an error
func handleStartPortForwardCommand(startCmd *command.PortForwardCommand) error {
	// Check if port forward already exists
	list, err := nocalhost.ListPortForward(startCmd.NameSpace, startCmd.AppName, startCmd.Service)
	if err != nil {
		return err
	}
	for _, pf := range list {
		if pf.LocalPort == startCmd.LocalPort && pf.RemotePort == startCmd.RemotePort {
			return errors.New(fmt.Sprintf("Port forward %d:%d already exists", startCmd.LocalPort, startCmd.RemotePort))
		}
	}

	lPort := startCmd.LocalPort
	rPort := startCmd.RemotePort

	pf := &profile.DevPortForward{
		LocalPort:         lPort,
		RemotePort:        rPort,
		Way:               "",
		Status:            "New",
		Reason:            "Add",
		Updated:           time.Now().Format("2006-01-02 15:04:05"),
		Pid:               0,
		RunByDaemonServer: true,
		Sudo:              isSudo,
		DaemonServerPid:   os.Getpid(),
	}
	err = nocalhost.AddPortForward(startCmd.NameSpace, startCmd.AppName, startCmd.Service, pf)
	if err != nil {
		return err
	}

	return nil
}

func updatePortForwardStatus(svc *model.NocalHostResource, localPort, remotePort int, status, reason string) error {
	dbPortForwardLocker.Lock()
	defer dbPortForwardLocker.Unlock()
	return nocalhost.UpdatePortForwardStatus(svc.NameSpace, svc.Application, svc.Service, localPort, remotePort, status, reason)
}
