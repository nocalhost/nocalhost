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
	"nocalhost/internal/nhctl/daemon_handler"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/syncthing/daemon"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"runtime/debug"
	"strings"
)

var (
	isSudo                      = false
	version                     = "1.0"
	commitId                    = ""
	pfManager                   *PortForwardManager
	tcpCtx, tcpCancelFunc       = context.WithCancel(context.Background()) // For stopping listening tcp port
	daemonCtx, daemonCancelFunc = context.WithCancel(context.Background()) // For exiting current daemon server
	startUpPath                 string
	upgrading                   bool
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

func StartDaemon(isSudoUser bool, v string, c string) error {
	startUpPath, _ = utils.GetNhctlPath()

	version = v
	commitId = c
	if isSudoUser && !utils.IsSudoUser() {
		return errors.New("Failed to start daemon server with sudo")
	}
	isSudo = isSudoUser // Mark daemon server if it is run as sudo
	//ports.IsPortAvailable("0.0.0.0", daemonListenPort())
	address := fmt.Sprintf("%s:%d", "0.0.0.0", daemonListenPort())
	listener, err := net.Listen("tcp4", address)
	if err != nil {
		return errors.Wrap(err, "")
	}

	// run the dev event listener
	if !isSudoUser {
		appmeta_manager.Init()
		appmeta_manager.RegisterListener(
			func(pack *appmeta_manager.ApplicationEventPack) error {
				kubeconfig, err := nocalhost.GetKubeConfigFromProfile(pack.Ns, pack.AppName)
				if err != nil {
					return nil
				}
				nhApp, err := app.NewApplication(pack.AppName, pack.Ns, kubeconfig, true)
				if err != nil {
					return nil
				}

				if pack.Event.EventType == appmeta.DEV_END {
					log.Logf(
						"Receive dev end event, stopping sync and pf for %s-%s-%s", pack.Ns, pack.AppName,
						pack.Event.ResourceName,
					)
					nhController := nhApp.Controller(pack.Event.ResourceName, pack.Event.DevType.Origin())
					if err := nhController.StopSyncAndPortForwardProcess(true); err != nil {
						return nil
					}
				} else if pack.Event.EventType == appmeta.DEV_STA {
					profile, _ := nhApp.GetProfile()

					// ignore the event from local
					if profile.Identifier == pack.Event.Identifier {
						return nil
					}

					log.Logf(
						"Receive dev start event, stopping pf for %s-%s-%s", pack.Ns, pack.AppName,
						pack.Event.ResourceName,
					)
					nhController := nhApp.Controller(pack.Event.ResourceName, pack.Event.DevType.Origin())
					if err := nhController.StopAllPortForward(); err != nil {
						return nil
					}
				}
				return nil
			},
		)
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
			cmdType, clientStack, err := command.ParseBaseCommand(bytes)
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
				handleCommand(conn, bytes, cmdType, clientStack)
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
	if err = recoverSyncthing(); err != nil {
		log.LogE(err)
	}

	select {
	case <-daemonCtx.Done():
		log.Log("Exit daemon server")
		return nil
	}
}

func handleCommand(conn net.Conn, bys []byte, cmdType command.DaemonCommandType, clientStack string) {
	var err error
	log.Infof("Handling %s command", cmdType)

	defer func() {
		if err != nil {
			log.Log("Client Stack: " + clientStack)
		}
	}()

	// prevent elder version to send cmd to daemon
	if clientStack == "" {
		err = Process(conn, func(conn net.Conn) (interface{}, error) {
			return nil, errors.New(
				fmt.Sprintf("There are multiple nhctl detected on your device, and current nhctl's version "+
					" is too old.  please update the current nhctl's version up to %s and try again.",
					version))
		})
		return
	}

	switch cmdType {
	case command.StartPortForward:
		err = Process(conn, func(conn net.Conn) (interface{}, error) {
			startCmd := &command.PortForwardCommand{}
			if err = json.Unmarshal(bys, startCmd); err != nil {
				return nil, err
			}

			if err = handleStartPortForwardCommand(startCmd); err != nil {
				return nil, err
			}

			return nil, nil
		})

	case command.StopPortForward:
		err = Process(conn, func(conn net.Conn) (interface{}, error) {
			pfCmd := &command.PortForwardCommand{}
			if err = json.Unmarshal(bys, pfCmd); err != nil {
				return nil, err
			}
			if err = handleStopPortForwardCommand(pfCmd); err != nil {
				return nil, err
			}

			return nil, nil
		})

	case command.StopDaemonServer:
		err = Process(conn, func(conn net.Conn) (interface{}, error) {
			return nil, nil
		})

		tcpCancelFunc()
		// todo: clean up resources
		daemonCancelFunc()

	case command.RestartDaemonServer:
		err = Process(conn, func(conn net.Conn) (interface{}, error) {
			if upgrading {
				return nil, errors.New("DaemonServer is upgrading, please try it later")
			}
			baseCmd := &command.BaseCommand{}
			if err = json.Unmarshal(bys, baseCmd); err != nil {
				return nil, err
			}
			return nil, handlerRestartDaemonServerCommand(isSudo, baseCmd.ClientPath)
		})

		log.Log("New daemon server is starting, exit this one")
		daemonCancelFunc()

	case command.GetDaemonServerInfo:
		err = Process(conn, func(conn net.Conn) (interface{}, error) {
			return &daemon_common.DaemonServerInfo{
				Version: version, CommitId: commitId, NhctlPath: startUpPath, Upgrading: upgrading,
			}, nil
		})

	case command.GetDaemonServerStatus:
		err = Process(conn, func(conn net.Conn) (interface{}, error) {
			return &daemon_common.DaemonServerStatusResponse{
				PortForwardList: pfManager.ListAllRunningPFGoRoutineProfile(),
			}, nil
		})

	case command.GetApplicationMeta:
		err = Process(conn, func(conn net.Conn) (interface{}, error) {
			gamCmd := &command.GetApplicationMetaCommand{}
			if err = json.Unmarshal(bys, gamCmd); err != nil {
				return nil, err
			}

			return appmeta_manager.GetApplicationMeta(gamCmd.NameSpace, gamCmd.AppName, []byte(gamCmd.KubeConfigContent)), nil
		})

	case command.GetApplicationMetas:
		err = Process(conn, func(conn net.Conn) (interface{}, error) {
			gamsCmd := &command.GetApplicationMetasCommand{}
			if err = json.Unmarshal(bys, gamsCmd); err != nil {
				return nil, err
			}

			return appmeta_manager.GetApplicationMetas(gamsCmd.NameSpace, []byte(gamsCmd.KubeConfigContent)), nil
		})

	case command.GetResourceInfo:
		err = Process(conn, func(conn net.Conn) (interface{}, error) {
			cmd := &command.GetResourceInfoCommand{}
			if err = json.Unmarshal(bys, cmd); err != nil {
				return nil, errors.Wrap(err, "")
			}

			return daemon_handler.HandleGetResourceInfoRequest(cmd), nil
		})

	case command.UpdateApplicationMeta:
		err = Process(conn, func(conn net.Conn) (interface{}, error) {
			cmd := &command.UpdateApplicationMetaCommand{}
			if err = json.Unmarshal(bys, cmd); err != nil {
				return nil, errors.Wrap(err, "")
			}
			return appmeta_manager.UpdateApplicationMetasManually(
				cmd.Namespace, []byte(cmd.KubeConfig), cmd.SecretName, cmd.Secret,
			), nil
		})
	}
}

func Process(conn net.Conn, fun func(conn net.Conn) (interface{}, error)) error {
	defer conn.Close()

	resp := command.BaseResponse{}

	result, errFromFun := fun(conn)
	if errFromFun != nil {
		log.LogE(errFromFun)
		resp.Status = command.FAIL
		resp.Msg = errFromFun.Error()
	} else {
		resp.Status = command.SUCCESS

		if result != nil {
			if bs, err := json.Marshal(&result); err != nil {
				resp.Status = command.INTERNAL_FAIL
				resp.Msg = err.Error()
			} else {
				resp.Data = bs
			}
		}
	}

	// try marshal again if fail
	bys, err := json.Marshal(&resp)
	if err != nil {
		log.LogE(errors.Wrap(err, ""))

		resp.Status = command.INTERNAL_FAIL
		resp.Msg = resp.Msg + fmt.Sprintf(" | INTERNAL_FAIL:[%s]", err.Error())

		if bys, err = json.Marshal(&resp); err != nil {
			log.LogE(errors.Wrap(err, ""))
			return err
		}
	}

	if _, err = conn.Write(bys); err != nil {
		log.LogE(errors.Wrap(err, ""))
		return err
	}

	cw, ok := conn.(interface{ CloseWrite() error })
	if ok {
		err := cw.CloseWrite()
		return err
	}

	return errFromFun
}

func handlerRestartDaemonServerCommand(isSudoUser bool, clientPath string) error {
	if upgrading {
		return errors.New("DaemonServer is upgrading, please try it later.")
	} else {
		upgrading = true
	}
	var nhctlPath string
	var err error

	if utils.IsWindows() {
		if clientPath == "" {
			return errors.New("ClientPath can not be nil in windows")
		}
		if nhctlPath, err = daemon_common.CopyNhctlBinaryToTmpDir(clientPath); err != nil {
			return err
		}
	} else {
		if nhctlPath, err = utils.GetNhctlPath(); err != nil {
			return err
		}
	}

	daemonArgs := []string{nhctlPath, "daemon", "start"}
	if isSudoUser {
		daemonArgs = append(daemonArgs, "--sudo", "true")
	}
	tcpCancelFunc() // Stop listening tcp port
	return daemon.RunSubProcess(daemonArgs, nil, false)
}

func handleStopPortForwardCommand(cmd *command.PortForwardCommand) error {
	// For compatibility
	return pfManager.StopPortForwardGoRoutine(cmd)
}

// If a port-forward already exist, skip it(don't do anything), and return an error
func handleStartPortForwardCommand(startCmd *command.PortForwardCommand) error {
	return pfManager.StartPortForwardGoRoutine(startCmd, true)
}
