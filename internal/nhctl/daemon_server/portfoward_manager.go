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
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s_runtime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type PortForwardManager struct {
	pfList map[string]context.CancelFunc
	lock   sync.Mutex
}

func NewPortForwardManager() *PortForwardManager {
	return &PortForwardManager{pfList: map[string]context.CancelFunc{}}
}

func (p *PortForwardManager) StopPortForwardGoRoutine(localPort, remotePort int) error {
	key := fmt.Sprintf("%d:%d", localPort, remotePort)
	cancel, ok := p.pfList[key]
	if !ok {
		return errors.New(fmt.Sprintf("Port-forward %d:%d is not managed by this PortForwardManger", localPort, remotePort))
	}
	cancel()
	delete(p.pfList, key)
	return nil
}

func (p *PortForwardManager) StartPortForwardGoRoutine(svc *model.NocalHostResource, localPort, remotePort int) error {

	key := fmt.Sprintf("%d:%d", localPort, remotePort)
	if _, ok := p.pfList[key]; ok {
		return errors.New(fmt.Sprintf("Port-forward %d:%d has been running in another go routine", localPort, remotePort))
	}

	nocalhostApp, err := app.NewApplication(svc.Application, svc.NameSpace, "", true)
	if err != nil {
		return err
	}

	// Check if port forward already exists
	if nocalhostApp.CheckIfPortForwardExists(svc.Service, localPort, remotePort) {
		return errors.New(fmt.Sprintf("Port forward %d:%d already exists", localPort, remotePort))
	}

	pf := &profile.DevPortForward{
		LocalPort:         localPort,
		RemotePort:        remotePort,
		Way:               "",
		Status:            "New",
		Reason:            "Add",
		Updated:           time.Now().Format("2006-01-02 15:04:05"),
		Pid:               0,
		RunByDaemonServer: true,
		Sudo:              isSudo,
		DaemonServerPid:   os.Getpid(),
	}

	p.lock.Lock()
	err = nocalhostApp.AddPortForwardToDB(svc.Service, pf)
	p.lock.Unlock()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.TODO())
	p.pfList[key] = cancel
	go func() {
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

		stdout, err := os.OpenFile(filepath.Join(logDir, fmt.Sprintf("%s_%s_%s_%s_%d_%d", svc.NameSpace, svc.Application, svc.Service, svc.PodName, localPort, remotePort)), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			log.LogE(err)
		}

		for {
			// stopCh control the port forwarding lifecycle. When it gets closed the
			// port forward will terminate
			stopCh := make(chan struct{}, 1)
			// readyCh communicate when the port forward is ready to get traffic
			readyCh := make(chan struct{})
			//endCh := make(chan struct{})
			heartbeatCtx, heartBeatCancel := context.WithCancel(ctx)
			errCh := make(chan error, 1)
			pfCtx, pfCancel := context.WithCancel(ctx)

			// stream is used to tell the port forwarder where to place its output or
			// where to expect input if needed. For the port forwarding we just need
			// the output eventually
			stream := genericclioptions.IOStreams{
				In:     stdout,
				Out:    stdout,
				ErrOut: stdout,
			}

			k8s_runtime.ErrorHandlers = append(k8s_runtime.ErrorHandlers, func(err error) {
				if strings.Contains(err.Error(), "error creating error stream for port") {
					log.Warnf("Port-forward %d:%d failed to create stream, try to reconnecting", localPort, remotePort)
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
			})

			go func() {
				select {
				case <-readyCh:
					log.Infof("Port forward %d:%d is ready", localPort, remotePort)
					go func() {
						nocalhostApp.CheckPidPortStatus(heartbeatCtx, svc.Service, localPort, remotePort, &p.lock)
						//checkLocalPortStatus(heartbeatCtx, svc, localPort, remotePort)
					}()
					go func() {
						nocalhostApp.SendHeartBeat(heartbeatCtx, "127.0.0.0", localPort)
					}()
				}
			}()

			go func() {
				select {
				case errCh <- nocalhostApp.PortForwardAPod(clientgoutils.PortForwardAPodRequest{
					Listen: []string{"0.0.0.0"},
					Pod: corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name:      svc.PodName,
							Namespace: svc.NameSpace,
						},
					},
					LocalPort: localPort,
					PodPort:   remotePort,
					Streams:   stream,
					StopCh:    stopCh,
					ReadyCh:   readyCh,
				}):

				case <-pfCtx.Done():
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
					return
				}
			}()

			select {
			case err := <-errCh:
				if err != nil {
					if strings.Contains(err.Error(), "unable to listen on any of the requested ports") {
						log.Warnf("Unable to listen on port %d", localPort)
						p.lock.Lock()
						err2 := nocalhostApp.UpdatePortForwardStatus(svc.Service, localPort, remotePort, "DISCONNECTED", fmt.Sprintf("Unable to listen on port %d", localPort))
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
					err = nocalhostApp.UpdatePortForwardStatus(svc.Service, localPort, remotePort, "RECONNECTING", "Port-forward failed, reconnecting after 30 seconds...")
					p.lock.Unlock()
					if err != nil {
						log.LogE(err)
					}
				} else {
					log.Warn("Reconnecting after 30 seconds...")
					heartBeatCancel()
					p.lock.Lock()
					err = nocalhostApp.UpdatePortForwardStatus(svc.Service, localPort, remotePort, "RECONNECTING", "Reconnecting after 30 seconds...")
					p.lock.Unlock()
					if err != nil {
						log.LogE(err)
					}
				}
				<-time.After(30 * time.Second)
				log.Info("Reconnecting...")
			case <-ctx.Done():
				log.Logf("Port-forward %d:%d done", localPort, remotePort)
				pfCancel()
				delete(p.pfList, key)
				err = nocalhostApp.DeletePortForwardFromDB(svc.Service, localPort, remotePort)
				if err != nil {
					log.LogE(err)
				}
				return
			}
		}
	}()
	return nil
}
