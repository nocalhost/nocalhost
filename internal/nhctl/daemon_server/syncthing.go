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
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/syncthing/daemon"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
)

func recoverSyncthing() error {
	log.Log("Recovering syncthing")
	appMap, err := nocalhost.GetNsAndApplicationInfo()
	if err != nil {
		return err
	}

	for ns, apps := range appMap {
		for _, appName := range apps {
			if err = recoverSyncthingForApplication(ns, appName); err != nil {
				log.LogE(err)
			}
		}
	}
	return nil
}

func recoverSyncthingForApplication(ns, appName string) error {
	profile, err := nocalhost.GetProfileV2(ns, appName)
	if err != nil {
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
			args := []string{nhctlPath, "sync", appName, "-d", svcProfile.ActualName, "--resume", "-n", ns}
			log.Logf("Resuming syncthing of %s-%s-%s", ns, appName, svcProfile.ActualName)
			if err = daemon.RunSubProcess(args, nil, false); err != nil {
				log.LogE(err)
			}
		}
	}

	return nil
}
