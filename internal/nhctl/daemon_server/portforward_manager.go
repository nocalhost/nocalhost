/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package daemon_server

import (
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	k8s_runtime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"net"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/daemon_common"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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

func (p *PortForwardManager) StopPortForwardGoRoutine(cmd *command.PortForwardCommand) error {
	key := fmt.Sprintf("%d:%d", cmd.LocalPort, cmd.RemotePort)
	pfProfile, ok := p.pfList[key]
	if ok {
		pfProfile.Cancel()
		err := <-pfProfile.StopCh
		delete(p.pfList, key)
		return err
	}

	kube, err := nocalhost.GetKubeConfigFromProfile(cmd.NameSpace, cmd.AppName, cmd.Nid)
	if err != nil {
		return err
	}
	nocalhostApp, err := app.NewApplication(cmd.AppName, cmd.NameSpace, kube, false)
	if err != nil {
		return err
	}

	if cmd.ServiceType == "" {
		cmd.ServiceType = "deployment"
	}
	nhController := nocalhostApp.Controller(cmd.Service, base.SvcType(cmd.ServiceType))
	return nhController.DeletePortForwardFromDB(cmd.LocalPort, cmd.RemotePort)
}

// ListAllRunningPortForwardGoRoutineProfile
func (p *PortForwardManager) ListAllRunningPFGoRoutineProfile() []*daemon_common.PortForwardProfile {
	result := make([]*daemon_common.PortForwardProfile, 0)
	for _, v := range p.pfList {
		result = append(result, v)
	}
	return result
}

func (p *PortForwardManager) RecoverPortForwardForApplication(ns, appName, nid string) error {
	profile, err := nocalhost.GetProfileV2(ns, appName, nid)
	if err != nil {
		if errors.Is(err, nocalhost.ProfileNotFound) {
			log.Warn("Profile is not exist, so ignore for recovering port forward")
			return nil
		}
		return err
	}
	//if profile == nil {
	//	return errors.New(fmt.Sprintf("Profile not found %s-%s", ns, appName))
	//}

	for _, svcProfile := range profile.SvcProfile {
		for _, pf := range svcProfile.DevPortForwardList {
			if pf.Sudo == isSudo { // Only recover port-forward managed by this daemon server
				log.Logf("Recovering port-forward %d:%d of %s-%s-%s", pf.LocalPort, pf.RemotePort, nid, ns, appName)
				svcType := pf.ServiceType
				// For compatibility
				if svcType == "" {
					svcType = svcProfile.GetType()
				}
				if svcType == "" {
					svcType = "deployment"
				}
				err = p.StartPortForwardGoRoutine(
					&command.PortForwardCommand{
						CommandType: command.StartPortForward,
						NameSpace:   ns,
						AppName:     appName,
						Service:     svcProfile.GetName(),
						ServiceType: svcType,
						PodName:     pf.PodName,
						LocalPort:   pf.LocalPort,
						RemotePort:  pf.RemotePort,
						Role:        pf.Role,
						Nid:         nid,
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
	defer RecoverDaemonFromPanic()

	log.Info("Recovering all port-forward")
	// Find all app
	appMap, err := nocalhost.GetNsAndApplicationInfo()
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	for _, app := range appMap {
		wg.Add(1)
		go func(namespace, app, nid string) {
			defer wg.Done()
			if err = p.RecoverPortForwardForApplication(namespace, app, nid); err != nil {
				log.LogE(err)
			}
		}(app.Namespace, app.Name, app.Nid)
	}
	wg.Wait()
	return nil
}

// StartPortForwardGoRoutine Start a port-forward
// If saveToDB is true, record it to leveldb
func (p *PortForwardManager) StartPortForwardGoRoutine(startCmd *command.PortForwardCommand, saveToDB bool) error {

	localPort, remotePort := startCmd.LocalPort, startCmd.RemotePort
	key := fmt.Sprintf("%d:%d", localPort, remotePort)
	if _, ok := p.pfList[key]; ok {
		log.Logf("Port-forward %d:%d has been running in another go routine, stop it first", localPort, remotePort)
		if err := p.StopPortForwardGoRoutine(startCmd); err != nil {
			log.LogE(err)
		}
	}

	kube, err := nocalhost.GetKubeConfigFromProfile(startCmd.NameSpace, startCmd.AppName, startCmd.Nid)
	if err != nil {
		return err
	}
	nocalhostApp, err := app.NewApplication(startCmd.AppName, startCmd.NameSpace, kube, true)
	if err != nil {
		return err
	}

	address := fmt.Sprintf("0.0.0.0:%d", startCmd.LocalPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return errors.New(fmt.Sprintf("Port %d is unavailable: %s", startCmd.LocalPort, err.Error()))
	}
	_ = listener.Close()

	nhController := nocalhostApp.Controller(startCmd.Service, base.SvcType(startCmd.ServiceType))

	if saveToDB {
		// Check if port forward already exists
		if existed, _ := nhController.CheckIfPortForwardExists(localPort, remotePort); existed {
			return errors.New(fmt.Sprintf("Port forward %d:%d already exists", localPort, remotePort))
		}
		pf := &profile.DevPortForward{
			LocalPort:  localPort,
			RemotePort: remotePort,
			Role:       startCmd.Role,
			Status:     "New",
			Reason:     "Add",
			PodName:    startCmd.PodName,
			Updated:    time.Now().Format("2006-01-02 15:04:05"),
			//Pid:        0,
			//RunByDaemonServer: true,
			Sudo:            isSudo,
			DaemonServerPid: os.Getpid(),
			ServiceType:     startCmd.ServiceType,
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
		SvcName:    startCmd.Service,
		SvcType:    startCmd.ServiceType,
		Role:       startCmd.Role,
		AppName:    startCmd.AppName,
		LocalPort:  startCmd.LocalPort,
		RemotePort: startCmd.RemotePort,
	}
	go func() {
		defer RecoverDaemonFromPanic()

		log.Logf("Forwarding %d:%d", localPort, localPort)

		logDir := filepath.Join(nocalhost.GetLogDir(), "port-forward")
		if _, err = os.Stat(logDir); err != nil {
			if os.IsNotExist(err) {
				if err = os.MkdirAll(logDir, 0644); err != nil {
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
			), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0755)
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

			recoverFunc := func(err error) {
				if strings.Contains(err.Error(), "error creating error stream for port") && !strings.Contains(err.Error(), "Timeout occured") {
					defer RecoverDaemonFromPanic()
					p, err := ParseErrToForwardPort(err.Error())
					if err != nil {
						log.LogE(err)
						return
					}
					if p.LocalPort != localPort || p.RemotePort != remotePort {
						return
					}
					log.WarnE(err, fmt.Sprintf("[RuntimeErrorHandler]Port-forward %d:%d failed to create stream,"+
						" try to reconnecting", localPort, remotePort))
					select {
					case _, isOpen := <-stopCh:
						if isOpen {
							log.Infof("[RuntimeErrorHandler]Closing Port-forward %d:%d by stop chan", localPort, remotePort)
							close(stopCh)
						} else {
							log.Infof("[RuntimeErrorHandler]Port-forward %d:%d has been closed, do nothing", localPort, remotePort)
						}
					default:
						log.Infof("[RuntimeErrorHandler]Closing Port-forward %d:%d'", localPort, remotePort)
						close(stopCh)
					}
				}
			}

			k8s_runtime.ErrorHandlers = append(k8s_runtime.ErrorHandlers, recoverFunc)

			go func() {
				defer RecoverDaemonFromPanic()
				select {
				case <-readyCh:
					log.Infof("Port forward %d:%d is ready", localPort, remotePort)

					lastStatus := ""
					currentStatus := ""
					for {
						select {
						case <-heartbeatCtx.Done():
							log.Infof("Stop sending heart beat to %d", localPort)
							return
						default:
							log.Debugf("try to send port-forward heartbeat to %d", localPort)
							err := nocalhostApp.SendPortForwardTCPHeartBeat(fmt.Sprintf("%s:%v", "127.0.0.1", localPort))
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
				}
			}()

			go func() {
				defer RecoverDaemonFromPanic()

				select {
				//case errCh <- nocalhostApp.PortForwardAPod(
				//	clientgoutils.PortForwardAPodRequest{
				//		Listen: []string{"0.0.0.0"},
				//		Pod: corev1.Pod{
				//			ObjectMeta: metav1.ObjectMeta{
				//				Name:      startCmd.PodName,
				//				Namespace: startCmd.NameSpace,
				//			},
				//		},
				//		LocalPort: localPort,
				//		PodPort:   remotePort,
				//		Streams:   stream,
				//		StopCh:    stopCh,
				//		ReadyCh:   readyCh,
				//	},
				//):
				//	log.Logf("Port-forward %d:%d occurs errors", localPort, remotePort)
				case errCh <- nocalhostApp.PortForward(startCmd.PodName, localPort, remotePort, readyCh, stopCh, stream):
					log.Logf("Port-forward %d:%d occurs errors", localPort, remotePort)
				}
			}()

			select {
			case err := <-errCh:
				if err != nil {
					found, _ := regexp.Match("pods \"(.*?)\" not found", []byte(err.Error()))
					if strings.Contains(err.Error(), "unable to listen on any of the requested ports") || found {
						if found {
							log.Logf("Pod: %s not found, remove port-forward for this pod", startCmd.PodName)
						} else {
							log.WarnE(err, fmt.Sprintf("Unable to listen on port %d", localPort))
						}
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
					log.WarnE(err, fmt.Sprintf("Port-forward %d:%d failed, reconnecting after 30 seconds...", localPort, remotePort))
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

func ParseErrToForwardPort(errStr string) (*clientgoutils.ForwardPort, error) {
	// error creating error stream for port 20153 -> 20153: Timeout occured
	s1 := strings.Split(errStr, ":")[0]
	s1 = strings.Split(s1, "port")[1]
	ps := strings.Split(s1, "->")
	if len(ps) != 2 {
		return nil, errors.New(fmt.Sprintf("Failed to parse fp from %s", errStr))
	}
	for i := 0; i < len(ps); i++ {
		ps[i] = strings.TrimSpace(ps[i])
	}
	localPort, err := strconv.ParseInt(ps[0], 0, 0)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Failed to parse fp from %s", errStr))
	}
	remotePort, err := strconv.ParseInt(ps[1], 0, 0)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Failed to parse fp from %s", errStr))
	}
	return &clientgoutils.ForwardPort{LocalPort: int(localPort), RemotePort: int(remotePort)}, nil
}
