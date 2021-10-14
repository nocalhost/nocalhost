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
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/controller"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/nocalhost_path"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/syncthing"
	"nocalhost/internal/nhctl/syncthing/daemon"
	"nocalhost/internal/nhctl/syncthing/network/req"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	utils2 "nocalhost/pkg/nhctl/utils"
	"strings"
	"sync"
	"time"
)

func recoverSyncthing() error {
	log.Log("Recovering syncthing")
	appMap, err := nocalhost.GetNsAndApplicationInfo()
	if err != nil {
		return err
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
	return nil
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

// reconnectedSyncthingIfNeeded will reconnect syncthing immediately if syncthing service is not available
func reconnectedSyncthingIfNeeded() {
	defer RecoverDaemonFromPanic()
	clone := appmeta_manager.GetAllApplicationMetasWithDeepClone()
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
			svcType, err1 := base.SvcTypeOfMutate(svcProfile.GetType())
			if err1 != nil {
				continue
			}
			svc := &controller.Controller{
				NameSpace:   meta.Ns,
				AppName:     meta.Application,
				Name:        svcProfile.GetName(),
				Type:        svcType,
				Identifier:  appProfile.Identifier,
				DevModeType: svcProfile.DevModeType,
				AppMeta:     meta,
			}
			if !svc.IsProcessor() || !svc.IsInDevMode() {
				continue
			}
			// reconnect two times:
			// the first time: using old port-forward, just create a new syncthing process
			//   detect syncthing service is available or not, if it's still not available
			// the second time: redo port-forward, and create a new syncthing process
			go func(svc *controller.Controller, appProfile *profile.AppProfileV2, svcProfile *profile.SvcProfileV2,
				meta *appmeta.ApplicationMeta) {
				defer RecoverDaemonFromPanic()
				for i := 0; i < 2; i++ {
					if err = retry.OnError(wait.Backoff{
						Steps:    3,
						Duration: 10 * time.Millisecond,
						Factor:   5,
					}, func(err error) bool {
						return err != nil
					}, func() error {
						status := svc.NewSyncthingHttpClient(2).GetSyncthingStatus()
						if status.Status != req.Disconnected {
							return nil
						}
						return errors.New("needs to reconnect")
					}); err == nil {
						break
					}
					log.LogDebugf("prepare to restore syncthing, name: %s\n", svcProfile.GetName())
					// TODO using developing container, otherwise will using default containerDevConfig
					if err = doReconnectSyncthing(svc, "", appProfile.Kubeconfig, i == 1); err != nil {
						log.PErrorf(
							"error while reconnect syncthing, ns: %s, app: %s, svc: %s, type: %s, err: %v",
							meta.Ns, meta.Application, svcProfile.GetName(), svcProfile.GetType(), err)
					}
				}
			}(svc, appProfile, svcProfile, meta)
		}
	}
}

// doReconnectSyncthing reconnect syncthing, if redoPortForward is true, needs to redo port-forward
func doReconnectSyncthing(svc *controller.Controller, container string, kubeconfigPath string, redoPortForward bool) error {
	svcProfile, err := svc.GetProfile()
	if err != nil {
		return err
	}
	// stop syncthing process with pid
	_ = svc.FindOutSyncthingProcess(func(pid int) error { return syncthing.Stop(pid, true) })
	// stop syncthing process with keywords
	str := strings.ReplaceAll(svc.GetApplicationSyncDir(), nocalhost_path.GetNhctlHomeDir(), "")
	utils2.KillSyncthingProcess(str)
	flag := false
	if config, err := svc.GetConfig(); err == nil {
		if cfg := config.GetContainerDevConfig(container); cfg != nil && cfg.Sync != nil {
			flag = cfg.Sync.Type == _const.DefaultSyncType
		}
	}
	// if reconnected is true, means needs to stop port-forward
	if redoPortForward {
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
		err = pfManager.StopPortForwardGoRoutine(p)
		if err != nil {
			log.LogE(err)
		}
		if svc.Client, err = clientgoutils.NewClientGoUtils(kubeconfigPath, svc.NameSpace); err != nil {
			return err
		}
		if p.PodName, err = svc.BuildPodController().GetNocalhostDevContainerPod(); err != nil {
			return err
		}
		if err = pfManager.StartPortForwardGoRoutine(p, true); err != nil {
			log.LogE(err)
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
