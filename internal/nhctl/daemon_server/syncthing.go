/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package daemon_server

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/appmeta_manager"
	"nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/controller"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/nocalhost_path"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/syncthing"
	"nocalhost/internal/nhctl/syncthing/daemon"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	utils2 "nocalhost/pkg/nhctl/utils"
	"strings"
	"sync"
	"time"
)

func recoverSyncthing() {
	log.Log("Recovering syncthing")
	appMap, err := nocalhost.GetNsAndApplicationInfo(true, true)
	if err != nil {
		return
	}

	wg := sync.WaitGroup{}
	for _, a := range appMap {
		wg.Add(1)
		go func(namespace, app, nid string) {
			defer wg.Done()
			if err = recoverSyncthingForApplication(namespace, app, nid); err != nil {
				log.LogE(err)
			}
		}(a.Namespace, a.Name, a.Nid)
	}
	wg.Wait()
}

func recoverSyncthingForApplication(ns, appName, nid string) error {
	profile, err := nocalhost.GetProfileV2(ns, appName, nid)
	if err != nil {
		if errors.Is(err, nocalhost.ProfileNotFound) {
			log.Warnf("Profile is not exist, so ignore for recovering for syncthing")
			return nil
		}

		return err
	}
	if profile == nil {
		return errors.New(fmt.Sprintf("Profile not found %s-%s", ns, appName))
	}

	nhctlPath, err := utils.GetNhctlPath()
	if err != nil {
		return err
	}

	for _, svcProfile := range profile.SvcProfile {
		if svcProfile.Syncing {
			// nhctl sync bookinfo -d productpage --resume --kubeconfig
			args := []string{nhctlPath, "sync", appName, "-d", svcProfile.GetName(), "--resume", "-n", ns}
			log.Logf("Resuming syncthing of %s-%s-%s", ns, appName, svcProfile.GetName())
			if err = daemon.RunSubProcess(args, nil, false); err != nil {
				log.LogE(err)
			}
		}
	}

	return nil
}

// reconnectSyncthingIfNeededWithPeriod will reconnect syncthing period if syncthing service is not available
func reconnectSyncthingIfNeededWithPeriod(duration time.Duration) {
	tick := time.NewTicker(duration)
	for {
		select {
		case <-tick.C:
			reconnectedSyncthingIfNeeded()
		}
	}
}

// namespace-nid-appName-serviceType-serviceName
var maps sync.Map

func toKey(controller2 *controller.Controller) string {
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		controller2.NameSpace,
		controller2.AppMeta.NamespaceId,
		controller2.AppName,
		controller2.Type,
		controller2.Name,
	)
}

type backoff struct {
	times    int
	lastTime time.Time
	nextTime time.Time
}

// reconnectedSyncthingIfNeeded will reconnect syncthing immediately if syncthing service is not available
func reconnectedSyncthingIfNeeded() {

	defer utils.RecoverFromPanic()
	clone := appmeta_manager.GetAllApplicationMetas()

	if clone == nil {
		return
	}

	for _, meta := range clone {
		if meta == nil || meta.DevMeta == nil {
			continue
		}
		appProfile, err := nocalhost.GetProfileV2(meta.Ns, meta.Application, meta.NamespaceId)
		if err != nil {
			continue
		}
		for _, svcProfile := range appProfile.SvcProfile {
			if svcProfile == nil || appmeta.HasDevStartingSuffix(svcProfile.Name) {
				continue
			}
			svcType, err1 := nocalhost.SvcTypeOfMutate(svcProfile.GetType())
			if err1 != nil {
				continue
			}

			svc, err := controller.NewController(meta.Ns, svcProfile.GetName(), meta.Application, appProfile.Identifier,
				svcType, nil, meta)
			if err != nil {
				log.WarnE(err, "")
				continue
			}

			if !svc.IsProcessor() {
				continue
			}
			// reconnect two times:
			// pre each time, check syncthing connections, if remote device connection is connected, no needs to recover
			// if remote device connection is not connected, but have this connection, just to do port-forward
			// the first time: using old port-forward, just create a new syncthing process
			//   detect syncthing service is available or not, if it's still not available
			// the second time: redo port-forward, and create a new syncthing process
			go func(svc *controller.Controller) {
				defer utils.RecoverFromPanic()
				var err error
				for i := 0; i < 2; i++ {
					if err = retry.OnError(wait.Backoff{
						Steps:    3,
						Duration: 10 * time.Millisecond,
						Factor:   5,
					}, func(err error) bool {
						return err != nil
					}, func() error {
						connected, err := svc.NewSyncthingHttpClient(2).SystemConnections()
						if connected && err == nil {
							return nil
						}
						return errors.New("needs to reconnect")
					}); err == nil {
						maps.Delete(toKey(svc))
						return
					}
					v, _ := maps.LoadOrStore(toKey(svc), &backoff{times: 0, lastTime: time.Now(), nextTime: time.Now()})
					if v.(*backoff).nextTime.Sub(time.Now()).Seconds() > 0 {
						return
					}
					v.(*backoff).times++
					// if last 5 min failed 5 times, will delay 3 min to retry
					if time.Now().Sub(v.(*backoff).lastTime).Minutes() >= 5 && v.(*backoff).times >= 5 {
						v.(*backoff).nextTime = time.Now().Add(time.Minute * 3)
					} else {
						v.(*backoff).lastTime = time.Now()
					}

					log.LogDebugf("prepare to restore syncthing, name: %s", svc.Name)
					// TODO using developing container, otherwise will using default containerDevConfig
					if err = doReconnectSyncthing(svc, "", appProfile.Kubeconfig, i == 1); err != nil {
						log.Errorf(
							"error while reconnect syncthing, ns: %s, app: %s, svc: %s, type: %s, err: %v",
							svc.AppMeta.Ns, svc.AppMeta.Application, svc.Name, svc.Type, err)
					}
				}
			}(svc)
		}
	}
}

// doReconnectSyncthing reconnect syncthing, if redoPortForward is true, needs to redo port-forward
func doReconnectSyncthing(svc *controller.Controller, container string, kubeconfigPath string, redoPortForward bool) error {
	svcProfile, err := svc.GetProfile()
	if err != nil {
		return err
	}
	client := svc.NewSyncthingHttpClient(2)
	if isConnected, err := client.SystemConnections(); isConnected {
		return nil
	} else
	// have connections but status is not connected, just to do port-forward, not needs to create a new syncthing process
	if !isConnected && err == nil {
		return doPortForward(svc, svcProfile, kubeconfigPath)
	}
	// using api to showdown server gracefully
	for i := 0; i < 5; i++ {
		_, _ = client.Post("/rest/system/shutdown", "")
	}
	// stop syncthing process with pid
	_ = svc.FindOutSyncthingProcess(func(pid int) error { return syncthing.Stop(pid, true) })
	// stop syncthing process with keywords
	str := strings.ReplaceAll(svc.GetSyncDir(), nocalhost_path.GetNhctlHomeDir(), "")
	utils2.KillSyncthingProcess(str)
	flag := false
	config := svc.Config()
	if cfg := config.GetContainerDevConfig(container); cfg != nil && cfg.Sync != nil {
		flag = cfg.Sync.Type == _const.DefaultSyncType
	}
	// if reconnected is true, means needs to stop port-forward
	if redoPortForward {
		if err = doPortForward(svc, svcProfile, kubeconfigPath); err != nil {
			return err
		}
	}
	newSyncthing, err := svc.NewSyncthing(container, svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin, flag)
	if err != nil {
		return err
	}
	// starts up a local syncthing
	if err = newSyncthing.Run(context.TODO()); err != nil {
		return err
	}
	return svc.SetSyncingStatus(true)
}

func doPortForward(svc *controller.Controller, svcProfile *profile.SvcProfileV2, kubeconfigPath string) error {
	p := &command.PortForwardCommand{
		NameSpace:   svc.NameSpace,
		AppName:     svc.AppName,
		Service:     svc.Name,
		ServiceType: svc.Type.String(),
		LocalPort:   svcProfile.RemoteSyncthingPort,
		RemotePort:  svcProfile.RemoteSyncthingPort,
		Role:        "SYNC",
		Nid:         svc.AppMeta.NamespaceId,
	}
	var err error
	if err = pfManager.StopPortForwardGoRoutine(p); err != nil {
		log.LogE(err)
	}
	if svc.Client, err = clientgoutils.NewClientGoUtils(kubeconfigPath, svc.NameSpace); err != nil {
		return err
	}
	if p.PodName, err = svc.GetDevModePodName(); err != nil {
		return err
	}
	if err = pfManager.StartPortForwardGoRoutine(p, true); err != nil {
		log.LogE(err)
	}
	return nil
}
