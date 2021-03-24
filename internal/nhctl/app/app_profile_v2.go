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

package app

import (
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/profile"
	"os"
)

// Deprecated
//func (a *Application) convertDevPortForwardList() {
//	var err error
//	changed := false
//	for _, svcProfile := range a.AppProfileV2.SvcProfile {
//		if len(svcProfile.DevPortForwardList) > 0 {
//			continue // already convert
//		}
//		for _, portString := range svcProfile.DevPortList {
//			log.Debugf("Converting %s", portString)
//			changed = true
//			devPortForward := &profile.DevPortForward{
//				Way:    "",
//				Status: "",
//			}
//			svcProfile.DevPortForwardList = append(svcProfile.DevPortForwardList, devPortForward)
//
//			ports := strings.Split(portString, ":")
//			devPortForward.LocalPort, err = strconv.Atoi(ports[0])
//			if err != nil {
//				log.WarnE(errors.Wrap(err, ""), err.Error())
//			}
//			devPortForward.RemotePort, err = strconv.Atoi(ports[1])
//			if err != nil {
//				log.WarnE(errors.Wrap(err, ""), err.Error())
//			}
//
//			// find way and status
//			for _, statusString := range svcProfile.PortForwardStatusList {
//				if strings.Contains(statusString, portString) {
//					// eg: 8091:8091(MANUAL-LISTEN)
//					str := strings.Split(statusString, "(") // MANUAL-LISTEN)
//					str = strings.Split(str[1], ")")        // MANUAL-LISTEN
//					str = strings.Split(str[0], "-")
//					devPortForward.Way = str[0]
//					devPortForward.Status = str[1]
//					log.Debugf("%s's status is %s-%s", devPortForward.Way, devPortForward.Status)
//					break
//				}
//			}
//			// find pid
//			for _, pidString := range svcProfile.PortForwardPidList {
//				if strings.Contains(pidString, portString) {
//					// eg: 8091:8091-16768
//					pidStr := strings.Split(pidString, "-")[1]
//					devPortForward.Pid, err = strconv.Atoi(pidStr)
//					if err != nil {
//						log.WarnE(errors.Wrap(err, ""), err.Error())
//					}
//					log.Debugf("%s's pid is %d", pidString, devPortForward.Pid)
//					break
//				}
//			}
//
//		}
//		svcProfile.PortForwardPidList = nil
//		svcProfile.PortForwardStatusList = nil
//		svcProfile.DevPortList = nil
//	}
//	if changed {
//		_ = a.SaveProfile()
//	}
//}

func (a *Application) LoadAppProfileV2(retry bool) error {

	app, err := nocalhost.GetProfileV2(a.NameSpace, a.Name)
	if err != nil {
		return err
	}
	if app == nil {
		app = &profile.AppProfileV2{}
	}

	a.AppProfileV2 = app
	return nil
}

// Deprecated: no support for profile v1 any more
//func (a *Application) checkIfAppProfileIsV2() (bool, error) {
//	_, err := os.Stat(a.getProfileV2Path())
//	if err == nil {
//		return true, nil
//	}
//
//	if !os.IsNotExist(err) {
//		return false, errors.Wrap(err, "")
//	}
//	return false, nil
//}

func (a *Application) checkIfAppConfigIsV2() (bool, error) {
	_, err := os.Stat(a.GetConfigV2Path())
	if err == nil {
		return true, nil
	}

	if !os.IsNotExist(err) {
		return false, errors.Wrap(err, "")
	}
	return false, nil
}

// Deprecated: no support for profile v1 any more
//func (a *Application) UpgradeAppProfileV1ToV2() error {
//	err := ConvertAppProfileFileV1ToV2(a.getProfilePath(), a.getProfileV2Path())
//	if err != nil {
//		return err
//	}
//	return os.Rename(a.getProfilePath(), a.getProfilePath()+".bak")
//}

func (a *Application) UpgradeAppConfigV1ToV2() error {
	err := ConvertConfigFileV1ToV2(a.GetConfigPath(), a.GetConfigV2Path())
	if err != nil {
		return err
	}
	return os.Rename(a.GetConfigPath(), a.GetConfigPath()+".bak")
}
