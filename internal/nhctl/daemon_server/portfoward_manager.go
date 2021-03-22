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
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/model"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"strings"
	"time"
)

type PortForwardManager struct {
	pfList map[string]context.CancelFunc
}

func NewPortForwardManager() *PortForwardManager {
	return &PortForwardManager{pfList: map[string]context.CancelFunc{}}
}

func (p *PortForwardManager) StopPortForwardGoRoutine(localPort, remotePort int) {
	key := fmt.Sprintf("%d:%d", localPort, remotePort)
	cancel, ok := p.pfList[key]
	if !ok {
		log.Logf("Port-forward %d:%d is not managed by this PortForwardManger")
		return
	}
	cancel()
	//delete(p.pfList, key)
}

func (p *PortForwardManager) StartPortForwardGoRoutine(svc *model.NocalHostResource, localPort, remotePort int) error {
	key := fmt.Sprintf("%d:%d", localPort, remotePort)
	ctx, cancel := context.WithCancel(context.TODO())
	p.pfList[key] = cancel
	go func() {
		log.Logf("Forwarding %d:%d", localPort, localPort)
		nocalhostApp, err := app.NewApplication(svc.Application, svc.NameSpace, "", true)
		if err != nil {
			delete(p.pfList, key)
			return
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
			pfCtx, _ := context.WithCancel(ctx)

			// stream is used to tell the port forwarder where to place its output or
			// where to expect input if needed. For the port forwarding we just need
			// the output eventually
			stream := genericclioptions.IOStreams{
				In:     os.Stdin,
				Out:    os.Stdout,
				ErrOut: os.Stderr,
			}

			go func() {
				select {
				case <-readyCh:
					log.Infof("Port forward %d:%d is ready", localPort, remotePort)
					go func() {
						checkLocalPortStatus(heartbeatCtx, svc, localPort, remotePort)
					}()
					go func() {
						sendHeartBeat(heartbeatCtx, localPort)
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
					return
				}
			}()

			select {
			case err := <-errCh:
				if err != nil {
					if strings.Contains(err.Error(), "unable to listen on any of the requested ports") {
						log.Warnf("Unable to listen on port %d", localPort)
						err2 := updatePortForwardStatus(svc, localPort, remotePort, "DISCONNECTED", fmt.Sprintf("Unable to listen on port %d", localPort))
						if err2 != nil {
							log.LogE(err2)
						}
						delete(p.pfList, key)
						return
					}
					log.WarnE(err, "Port-forward failed, reconnecting after 30 seconds...")
					heartBeatCancel()
					err = updatePortForwardStatus(svc, localPort, remotePort, "RECONNECTING", fmt.Sprintf("Port-forward %d:%d failed, reconnecting after 30 seconds...", localPort, remotePort))
					if err != nil {
						log.LogE(err)
					}
				} else {
					log.Warn("Reconnecting after 30 seconds...")
					heartBeatCancel()
					err = updatePortForwardStatus(svc, localPort, remotePort, "RECONNECTING", "Reconnecting after 30 seconds...")
					if err != nil {
						log.LogE(err)
					}
				}
				<-time.After(30 * time.Second)
				log.Info("Reconnecting...")
			case <-ctx.Done():
				log.Log("Port-forward %d:%d done", localPort, remotePort)
				delete(p.pfList, key)
				return
			}
		}
	}()
	return nil
}
