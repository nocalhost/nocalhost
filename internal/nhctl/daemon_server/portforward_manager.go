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
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s_runtime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/daemon_common"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

type PortForwardManager struct {
	pfList map[string]*daemon_common.PortForwardProfile
	lock   sync.Mutex
}

func NewPortForwardManager() *PortForwardManager {
	return &PortForwardManager{pfList: map[string]*daemon_common.PortForwardProfile{}}
}

func (p *PortForwardManager) StopPortForwardGoRoutine(localPort, remotePort int) error {
	key := fmt.Sprintf("%d:%d", localPort, remotePort)
	pfProfile, ok := p.pfList[key]
	if !ok {
		return errors.New(
			fmt.Sprintf(
				"Port-forward %d:%d is not managed by this PortForwardManger", localPort, remotePort,
			),
		)
	}
	pfProfile.Cancel()
	err := <-pfProfile.StopCh
	delete(p.pfList, key)
	return err
}

// ListAllRunningPortForwardGoRoutineProfile
func (p *PortForwardManager) ListAllRunningPFGoRoutineProfile() []*daemon_common.PortForwardProfile {
	result := make([]*daemon_common.PortForwardProfile, 0)
	for _, v := range p.pfList {
		result = append(result, v)
	}
	return result
}

func (p *PortForwardManager) RecoverPortForwardForApplication(ns, appName string) error {
	profile, err := nocalhost.GetProfileV2(ns, appName)
	if err != nil {
		if errors.Is(err, nocalhost.ProfileNotFound){
			log.Warnf("Profile is not exist, so ignore for recovering")
			return nil
		}
		return err
	}
	if profile == nil {
		return errors.New(fmt.Sprintf("Profile not found %s-%s", ns, appName))
	}

	for _, svcProfile := range profile.SvcProfile {
		for _, pf := range svcProfile.DevPortForwardList {
			if pf.RunByDaemonServer && pf.Sudo == isSudo { // Only recover port-forward managed by this daemon server
				log.Logf("Recovering port-forward %d:%d of %s-%s", pf.LocalPort, pf.RemotePort, ns, appName)
				svcType := pf.ServiceType
				// For compatibility
				if svcType == "" {
					svcType = svcProfile.Type
				}
				if svcType == "" {
					svcType = "deployment"
				}
				err = p.StartPortForwardGoRoutine(
					&command.PortForwardCommand{
						CommandType: command.StartPortForward,
						NameSpace:   ns,
						AppName:     appName,
						Service:     svcProfile.ActualName,
						ServiceType: svcType,
						PodName:     pf.PodName,
						LocalPort:   pf.LocalPort,
						RemotePort:  pf.RemotePort,
						Role:        pf.Role,
					}, false,
				)
				if err != nil {
					log.LogE(err)
				}
			}
		}
	}
	return nil
}

func (p *PortForwardManager) RecoverAllPortForward() error {
	log.Info("Recovering port-forward")
	// Find all app
	appMap, err := nocalhost.GetNsAndApplicationInfo()
	if err != nil {
		return err
	}

	for ns, apps := range appMap {
		for _, appName := range apps {
			if err = p.RecoverPortForwardForApplication(ns, appName); err != nil {
				log.LogE(err)
			}
		}
	}
	return nil
}

// Start a port-forward
// If saveToDB is true, record it to leveldb
func (p *PortForwardManager) StartPortForwardGoRoutine(startCmd *command.PortForwardCommand, saveToDB bool) error {

	localPort, remotePort := startCmd.LocalPort, startCmd.RemotePort
	key := fmt.Sprintf("%d:%d", localPort, remotePort)
	if _, ok := p.pfList[key]; ok {
		log.Logf("Port-forward %d:%d has been running in another go routine, stop it first", localPort, remotePort)
		if err := p.StopPortForwardGoRoutine(localPort, remotePort); err != nil {
			return err
		}
	}

	kube, err := nocalhost.GetKubeConfigFromProfile(startCmd.NameSpace, startCmd.AppName)
	if err != nil {
		return err
	}
	nocalhostApp, err := app.NewApplication(startCmd.AppName, startCmd.NameSpace, kube, true)
	if err != nil {
		return err
	}

	nhController := nocalhostApp.Controller(startCmd.Service, appmeta.SvcType(startCmd.ServiceType))

	if saveToDB {
		// Check if port forward already exists
		if existed, _ := nhController.CheckIfPortForwardExists(localPort, remotePort); existed {
			return errors.New(fmt.Sprintf("Port forward %d:%d already exists", localPort, remotePort))
		}
		pf := &profile.DevPortForward{
			LocalPort:         localPort,
			RemotePort:        remotePort,
			Role:              startCmd.Role,
			Status:            "New",
			Reason:            "Add",
			PodName:           startCmd.PodName,
			Updated:           time.Now().Format("2006-01-02 15:04:05"),
			Pid:               0,
			RunByDaemonServer: true,
			Sudo:              isSudo,
			DaemonServerPid:   os.Getpid(),
			ServiceType:       startCmd.ServiceType,
		}

		p.lock.Lock()
		log.Logf("Saving port-forward %d:%d to db", pf.LocalPort, pf.RemotePort)
		err = nhController.AddPortForwardToDB(pf)
		p.lock.Unlock()
		if err != nil {
			return err
		}
	}

	ctx, cancel := context.WithCancel(context.TODO())
	p.pfList[key] = &daemon_common.PortForwardProfile{
		Cancel:     cancel,
		StopCh:     make(chan error, 1),
		NameSpace:  startCmd.NameSpace,
		AppName:    startCmd.AppName,
		LocalPort:  startCmd.LocalPort,
		RemotePort: startCmd.RemotePort,
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Fatalf("DAEMON-RECOVER: %s", string(debug.Stack()))
			}
		}()

		log.Logf("Forwarding %d:%d", localPort, localPort)

		logDir := filepath.Join(nocalhost.GetLogDir(), "port-forward")
		if _, err = os.Stat(logDir); err != nil {
			if os.IsNotExist(err) {
				err = os.MkdirAll(logDir, 0644)
				if err != nil {
					log.LogE(errors.Wrap(err, ""))
				}
			} else {
				log.LogE(errors.Wrap(err, ""))
			}
		}

		stdout, err := os.OpenFile(
			filepath.Join(
				logDir, fmt.Sprintf(
					"%s_%s_%s_%d_%d", startCmd.NameSpace, startCmd.AppName, startCmd.Service, localPort, remotePort,
				),
			), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0755,
		)
		if err != nil {
			log.LogE(err)
		}

		for {
			// stopCh control the port forwarding lifecycle. When it gets closed the
			// port forward will terminate
			stopCh := make(chan struct{}, 1)
			// readyCh communicate when the port forward is ready to get traffic
			readyCh := make(chan struct{})
			heartbeatCtx, heartBeatCancel := context.WithCancel(ctx)
			errCh := make(chan error, 1)

			// stream is used to tell the port forwarder where to place its output or
			// where to expect input if needed. For the port forwarding we just need
			// the output eventually
			stream := genericclioptions.IOStreams{
				In:     stdout,
				Out:    stdout,
				ErrOut: stdout,
			}

			k8s_runtime.ErrorHandlers = append(
				k8s_runtime.ErrorHandlers, func(err error) {
					if strings.Contains(err.Error(), "error creating error stream for port") {
						log.Warnf(
							"Port-forward %d:%d failed to create stream, try to reconnecting", localPort, remotePort,
						)
						select {
						case _, isOpen := <-stopCh:
							if isOpen {
								log.Infof("Closing Port-forward %d:%d' by stop chan", localPort, remotePort)
								close(stopCh)
							} else {
								log.Infof("Port-forward %d:%d has been closed, do nothing", localPort, remotePort)
							}
						default:
							log.Infof("Closing Port-forward %d:%d'", localPort, remotePort)
							close(stopCh)
						}
					}
				},
			)

			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Fatalf("DAEMON-RECOVER: %s", string(debug.Stack()))
					}
				}()

				select {
				case <-readyCh:
					log.Infof("Port forward %d:%d is ready", localPort, remotePort)
					go func() {
						defer func() {
							if r := recover(); r != nil {
								log.Fatalf("DAEMON-RECOVER: %s", string(debug.Stack()))
							}
						}()

						lastStatus := ""
						currentStatus := ""
						for {
							select {
							case <-heartbeatCtx.Done():
								log.Infof("Stop sending heart beat to %d", localPort)
								return
							default:
								log.Infof("try to send port-forward heartbeat to %d", localPort)
								err := nocalhostApp.SendPortForwardTCPHeartBeat(
									fmt.Sprintf(
										"%s:%v", "127.0.0.1", localPort,
									),
								)
								if err != nil {
									log.WarnE(err, "")
									currentStatus = "HeartBeatLoss"
								} else {
									currentStatus = "LISTEN"
								}
								if lastStatus != currentStatus {
									lastStatus = currentStatus
									p.lock.Lock()
									nhController.UpdatePortForwardStatus(localPort, remotePort, lastStatus, "Heart Beat")
									p.lock.Unlock()
								}
								<-time.After(30 * time.Second)
							}
						}
					}()
				}
			}()

			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Fatalf("DAEMON-RECOVER: %s", string(debug.Stack()))
					}
				}()

				select {
				case errCh <- nocalhostApp.PortForwardAPod(
					clientgoutils.PortForwardAPodRequest{
						Listen: []string{"0.0.0.0"},
						Pod: corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{
								Name:      startCmd.PodName,
								Namespace: startCmd.NameSpace,
							},
						},
						LocalPort: localPort,
						PodPort:   remotePort,
						Streams:   stream,
						StopCh:    stopCh,
						ReadyCh:   readyCh,
					},
				):
					log.Logf("Port-forward %d:%d occurs errors", localPort, remotePort)
				}
			}()

			select {
			case err := <-errCh:
				if err != nil {
					if strings.Contains(err.Error(), "unable to listen on any of the requested ports") {
						log.Warnf("Unable to listen on port %d", localPort)
						p.lock.Lock()
						err2 := nhController.UpdatePortForwardStatus(localPort, remotePort, "DISCONNECTED",
							fmt.Sprintf("Unable to listen on port %d", localPort))
						p.lock.Unlock()
						if err2 != nil {
							log.LogE(err2)
						}
						delete(p.pfList, key)
						return
					}
					log.WarnE(err, "Port-forward failed, reconnecting after 30 seconds...")
					heartBeatCancel()
					p.lock.Lock()
					err = nhController.UpdatePortForwardStatus(localPort, remotePort, "RECONNECTING",
						"Port-forward failed, reconnecting after 30 seconds...")
					p.lock.Unlock()
					if err != nil {
						log.LogE(err)
					}
				} else {
					log.Warn("Reconnecting after 30 seconds...")
					heartBeatCancel()
					p.lock.Lock()
					err = nhController.UpdatePortForwardStatus(localPort, remotePort, "RECONNECTING",
						"Reconnecting after 30 seconds...")
					p.lock.Unlock()
					if err != nil {
						log.LogE(err)
					}
				}
				<-time.After(30 * time.Second)
				log.Info("Reconnecting...")
			case <-ctx.Done():
				log.Logf("Port-forward %d:%d done", localPort, remotePort)
				log.Log("Stopping pf routine")
				select {
				case _, ok := <-stopCh:
					if ok {
						log.Logf("Stopping port-forward %d-%d by stopCH", localPort, remotePort)
						close(stopCh)
					} else {
						log.Logf("Port-forward %d-%d has already been stopped", localPort, remotePort)
					}
				default:
					log.Logf("Stopping port-forward %d-%d", localPort, remotePort)
					close(stopCh)
				}
				//delete(p.pfList, key)
				log.Logf("Delete port-forward %d:%d record", localPort, remotePort)
				err = nhController.DeletePortForwardFromDB(localPort, remotePort)
				if err != nil {
					log.LogE(err)
				}
				if pfProfile, ok := p.pfList[key]; ok {
					pfProfile.StopCh <- err
				}
				return
			}
		}
	}()
	return nil
}
