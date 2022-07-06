/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package daemon_server

import (
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"net"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/daemon_common"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/dbutils"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/nocalhost/db"
	"nocalhost/internal/nhctl/nocalhost_path"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/internal/nhctl/watcher"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"path/filepath"
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
		var err error
		select {
		case err = <-pfProfile.StopCh:
		default:
		}
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
	nhController, err := nocalhostApp.Controller(cmd.Service, base.SvcType(cmd.ServiceType))
	if err != nil {
		return err
	}
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

	var found bool
	for _, svcProfile := range profile.SvcProfile {
		for _, pf := range svcProfile.DevPortForwardList {
			if pf.Sudo == isSudo { // Only recover port-forward managed by this daemon server
				found = true
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
						LocalPort:   pf.LocalPort,
						RemotePort:  pf.RemotePort,
						Role:        pf.Role,
						Nid:         nid,

						PodName:         pf.PodName,
						OwnerName:       pf.OwnerName,
						OwnerKind:       pf.OwnerKind,
						OwnerApiVersion: pf.OwnerApiVersion,
						Labels:          pf.Labels,
					}, false,
				)
				if err != nil {
					log.LogE(err)
				}
			}
		}
	}
	if found {
		if err = p.recordPortForward(ns, nid, appName, func() bool { return true }); err != nil {
			log.Info(err)
		}
	}
	return nil
}

func (p *PortForwardManager) RecoverAllPortForward() {
	defer utils.RecoverFromPanic()

	log.Info("Recovering all port-forward")
	var scanned bool
	if err := db.GetOrCreatePortForwardLevelDBFunc(
		true, func(utils *dbutils.LevelDBUtils) {
			if v, err := utils.Get([]byte("scanned")); err == nil && len(v) != 0 {
				scanned = true
			}
		},
	); err != nil {
		log.Infof("Error while opening port-forward level-db")
	}

	// Find all app
	appMap, err := nocalhost.GetNsAndApplicationInfo(scanned, true)
	if err != nil {
		log.LogE(err)
		return
	}

	var lock sync.WaitGroup
	for _, application := range appMap {
		lock.Add(1)
		func(namespace, app, nid string, lock *sync.WaitGroup) {
			defer lock.Done()
			time.Sleep(time.Millisecond * 50)
			if err = p.RecoverPortForwardForApplication(namespace, app, nid); err != nil {
				log.LogE(err)
			}
		}(application.Namespace, application.Name, application.Nid, &lock)
	}
	lock.Wait()

	if err := db.GetOrCreatePortForwardLevelDBFunc(
		false, func(utils *dbutils.LevelDBUtils) {
			_ = utils.Put([]byte("scanned"), []byte("true"))
		},
	); err != nil {
		log.Infof("Error while writing port-forward level-db")
	}
}

func (p *PortForwardManager) recordPortForward(ns, nid, app string, isPortForwarding func() bool) error {
	path := filepath.Join(nocalhost_path.GetAppDbDir(ns, app, nid), "portforward")
	if isPortForwarding() {
		return ioutil.WriteFile(path, nil, 0644)
	} else {
		return os.Remove(path)
	}
}

func GetTopController(refs []v1.OwnerReference, client *clientgoutils.ClientGoUtils) *v1.OwnerReference {
	controller := clientgoutils.GetControllerOfNoCopy(refs)
	if controller == nil {
		return nil
	}

	if gv, err := schema.ParseGroupVersion(controller.APIVersion); err != nil {
		return controller
	} else {
		gvr, nsScope, e := client.ResourceForGVK(gv.WithKind(controller.Kind))
		if e != nil || !nsScope {
			return controller
		}

		unstructured, e := client.GetUnstructured(gvr.Resource, controller.Name)
		if e != nil {
			return controller
		}

		ownerRef := GetTopController(unstructured.GetOwnerReferences(), client)
		if ownerRef == nil {
			return controller
		}
		return ownerRef
	}
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

	nhController, err := nocalhostApp.Controller(startCmd.Service, base.SvcType(startCmd.ServiceType))
	if err != nil {
		return err
	}

	_ = p.recordPortForward(startCmd.NameSpace, startCmd.Nid, startCmd.AppName, func() bool { return true })

	howToGetCurrentPod := func() (*corev1.Pod, error) {
		// first find the pod should be port-forward
		var currentPod *corev1.Pod

		if currentPod, err = nhController.Client.GetPod(startCmd.PodName); err != nil && len(startCmd.Labels) > 0 {
			if pods, _ := nhController.Client.Labels(startCmd.Labels).ListPods(); len(pods) > 0 {
				for _, pod := range pods {

					if startCmd.OwnerName != "" {
						controller := GetTopController(pod.GetOwnerReferences(), nhController.Client)
						if controller == nil {
							continue
						}

						if controller.Name == startCmd.OwnerName && controller.APIVersion == startCmd.OwnerApiVersion &&
							controller.Kind == startCmd.OwnerKind {

							err = nil
							currentPod = &pod
							break
						}
					} else {
						err = nil
						currentPod = &pod
						break
					}
				}
			}
		}

		return currentPod, err
	}

	var currentPod *corev1.Pod
	if saveToDB {
		// Check if port forward already exists
		if existed, _ := nhController.CheckIfPortForwardExists(localPort, remotePort); existed {
			return errors.New(fmt.Sprintf("Port forward %d:%d already exists", localPort, remotePort))
		}

		pf := &profile.DevPortForward{
			LocalPort:       localPort,
			RemotePort:      remotePort,
			Role:            startCmd.Role,
			Status:          "New",
			Reason:          "Add",
			PodName:         startCmd.PodName,
			Updated:         time.Now().Format("2006-01-02 15:04:05"),
			Sudo:            isSudo,
			DaemonServerPid: os.Getpid(),
			ServiceType:     startCmd.ServiceType,
		}

		if currentPod, err = howToGetCurrentPod(); err != nil {
			return err
		} else {
			pf.Labels = currentPod.Labels
			controller := GetTopController(currentPod.GetOwnerReferences(), nhController.Client)

			if controller != nil {
				pf.OwnerName = controller.Name
				pf.OwnerApiVersion = controller.APIVersion
				pf.OwnerKind = controller.Kind
			}
		}

		log.Logf("Saving port-forward %d:%d to db", pf.LocalPort, pf.RemotePort)
		p.lock.Lock()
		err = nhController.AddPortForwardToDB(pf)
		p.lock.Unlock()
		if err != nil {
			return err
		}
	} else {
		if currentPod, err = howToGetCurrentPod(); err != nil {
			return err
		}
	}

	startCmd.PodName = currentPod.Name

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
		defer utils.RecoverFromPanic()

		log.Logf("Forwarding %d:%d", localPort, remotePort)

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
			), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0755,
		)
		if err != nil {
			log.LogE(err)
		}

		sleepBackOff := 15 * time.Second

		for {
			// stopCh control the port forwarding lifecycle. When it gets closed the
			// port forward will terminate
			stopCh := make(chan struct{}, 1)
			// readyCh communicate when the port forward is ready to get traffic
			readyCh := make(chan struct{})
			//heartbeatCtx, heartBeatCancel := context.WithCancel(ctx)
			errCh := make(chan error, 1)

			// stream is used to tell the port forwarder where to place its output or
			// where to expect input if needed. For the port forwarding we just need
			// the output eventually
			stream := genericclioptions.IOStreams{
				In:     stdout,
				Out:    stdout,
				ErrOut: stdout,
			}

			// if pods is deleted, try to close port-forward
			watcher.NewSimpleWatcher(
				nhController.Client,
				"pods",
				v1.ListOptions{FieldSelector: fields.OneTermEqualSelector("metadata.name", startCmd.PodName).String()},
				stopCh,
				nil,
				func(key string, quitChan <-chan struct{}) {
					defer utils.RecoverFromPanic()
					select {
					case errCh <- k8serrors.NewNotFound(schema.GroupResource{Resource: "pods"}, startCmd.PodName):
						close(errCh)
					default:
					}
				},
			)

			go func() {
				defer utils.RecoverFromPanic()
				select {
				case <-readyCh:
					log.Infof("Port forward %d:%d is ready", localPort, remotePort)
					p.lock.Lock()
					_ = nhController.UpdatePortForwardStatus(localPort, remotePort, "LISTEN", "listen")
					p.lock.Unlock()
				case <-time.After(60 * time.Second):
					log.Infof("Waiting Port forward %d:%d timeout", localPort, remotePort)
				}
			}()

			go func() {
				defer utils.RecoverFromPanic()
				errCh <- nocalhostApp.PortForward(startCmd.PodName, localPort, remotePort, readyCh, stopCh, stream)
				log.Logf("Port-forward %d:%d occurs errors", localPort, remotePort)
			}()

			var block = true

			select {
			case errs := <-errCh:

				closeChanGracefully(stopCh)
				closeChanGracefully(readyCh)

				reconnectMsg := fmt.Sprintf("Reconnecting after %s seconds...", sleepBackOff.String())

				if errs != nil && k8serrors.IsNotFound(errs) {

					// if pod not found, try to get pod by labels
					if pod, err := howToGetCurrentPod(); err != nil {
						p.lock.Lock()
						err = nhController.UpdatePortForwardStatus(
							localPort, remotePort, "RECONNECTING",
							reconnectMsg,
						)
						p.lock.Unlock()

						// Avoid overloading the api with multiple requests
						sleepBackOff += 15 * time.Second
						if sleepBackOff.Seconds() > 60 {
							sleepBackOff = 60 * time.Second
						}
					} else {

						log.Logf("New pod %s for port-forward found", pod.Name)
						startCmd.PodName = pod.Name
						block = false
					}

				} else if errs != nil && strings.Contains(errs.Error(), "failed to find socat") {

					log.Logf("failed to find socat, err: %v", errs)
					p.lock.Lock()
					err = nhController.UpdatePortForwardStatus(
						localPort, remotePort, "Socat not found", "failed to find socat",
					)
					p.lock.Unlock()
					if err != nil {
						log.LogE(err)
					}
					delete(p.pfList, key)
					return
				} else {

					log.Warn(reconnectMsg)
					p.lock.Lock()
					err = nhController.UpdatePortForwardStatus(
						localPort, remotePort, "RECONNECTING",
						reconnectMsg,
					)
					p.lock.Unlock()
					if err != nil {
						log.LogE(err)
					}
				}

				if block {
					<-time.After(sleepBackOff)
					log.Infof("Reconnecting %d:%d...", localPort, remotePort)
				}

			case <-ctx.Done():
				log.Logf("Port-forward %d:%d done", localPort, remotePort)
				log.Log("Stopping pf routine")
				closeChanGracefully(stopCh)
				//delete(p.pfList, key)
				log.Logf("Delete port-forward %d:%d record", localPort, remotePort)
				err = nhController.DeletePortForwardFromDB(localPort, remotePort)
				if err != nil {
					log.LogE(err)
				}

				p.recordPortForward(
					startCmd.NameSpace, startCmd.Nid, startCmd.AppName, func() bool {
						return nhController.IsPortForwarding()
					},
				)

				if pfProfile, ok := p.pfList[key]; ok {
					pfProfile.StopCh <- err
				}
				return
			}
		}
	}()
	return nil
}

func closeChanGracefully(stopCh chan struct{}) {
	select {
	case _, ok := <-stopCh:
		if ok {
			close(stopCh)
		}
	default:
		close(stopCh)
	}
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
