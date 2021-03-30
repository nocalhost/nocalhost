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
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/syncthing/daemon"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"os/exec"
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
	profile, err := nocalhost.GetProfileV2(ns, appName, nil)
	if err != nil {
		return err
	}
	if profile == nil {
		return errors.New("Profile not found")
	}

	nhctlPath, err := exec.LookPath(utils.GetNhctlBinName())
	if err != nil {
		return errors.Wrap(err, "")
	}

	for _, svcProfile := range profile.SvcProfile {
		if svcProfile.Syncing {
			// nhctl sync bookinfo -d productpage --resume --kubeconfig /Users/xinxinhuang/.nh/plugin/kubeConfigs/293_config
			args := []string{nhctlPath, "sync", appName, "-d", svcProfile.ActualName, "--resume", "-n", ns}
			log.Logf("Resuming syncthing of %s-%s-%s", ns, appName, svcProfile.ActualName)
			if err = daemon.RunSubProcess(args, nil, false); err != nil {
				log.LogE(err)
			}
		}
	}

	return nil
}
