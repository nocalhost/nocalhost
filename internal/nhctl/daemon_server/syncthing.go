/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package daemon_server

import (
	"fmt"
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/syncthing/daemon"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"sync"
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
