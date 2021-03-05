/*
Copyright 2020 The Nocalhost Authors.
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
	"nocalhost/internal/nhctl/syncthing/terminate"
	"nocalhost/pkg/nhctl/log"
	"time"
)

func (a *Application) AppendPortForward(svcName string, devPortForward *DevPortForward) {
	a.GetSvcProfileV2(svcName).DevPortForwardList = append(a.GetSvcProfileV2(svcName).DevPortForwardList, devPortForward)
}

func (a *Application) SetPortForwardPid(svcName string, localPort int, remotePort int, pid int) error {
	err := a.ReadBeforeWriteProfile()
	if err != nil {
		return err
	}
	found := false
	svcProfile := a.GetSvcProfileV2(svcName)
	for _, portForward := range svcProfile.DevPortForwardList {
		if portForward.LocalPort == localPort && portForward.RemotePort == remotePort {
			portForward.Pid = pid
			portForward.Updated = time.Now().Format("2006-01-02 15:04:05")
			found = true
			break
		}
	}
	if !found {
		newPF := &DevPortForward{
			LocalPort:  localPort,
			RemotePort: remotePort,
			Way:        "",
			Status:     "",
			Reason:     "",
			Updated:    time.Now().Format("2006-01-02 15:04:05"),
			Pid:        pid,
		}
		svcProfile.DevPortForwardList = append(svcProfile.DevPortForwardList, newPF)
	}
	return a.SaveProfile()
}

func (a *Application) UpdatePortForwardStatus(svcName string, localPort int, remotePort int, portStatus string, reason string) error {
	err := a.ReadBeforeWriteProfile()
	if err != nil {
		return err
	}
	for _, portForward := range a.GetSvcProfileV2(svcName).DevPortForwardList {
		if portForward.LocalPort == localPort && portForward.RemotePort == remotePort {
			portForward.Status = portStatus
			portForward.Reason = reason
			portForward.Updated = time.Now().Format("2006-01-02 15:04:05")
			break
		}
	}
	return a.SaveProfile()
}

func (a *Application) EndDevPortForward(svcName string, localPort int, remotePort int) {
	indexToDelete := -1
	for index, portForward := range a.GetSvcProfileV2(svcName).DevPortForwardList {
		if portForward.LocalPort == localPort && portForward.RemotePort == remotePort {
			log.Infof("Kill %v", *portForward)
			err := terminate.Terminate(portForward.Pid, true, "port-forward")
			if err != nil {
				log.Warn(err.Error())
			}
			indexToDelete = index
			time.Sleep(3 * time.Second)
			break
		}
	}

	// remove portForward from DevPortForwardList
	if indexToDelete > -1 {

		originList := a.GetSvcProfileV2(svcName).DevPortForwardList
		newList := make([]*DevPortForward, 0)
		for index, p := range originList {
			if index != indexToDelete {
				newList = append(newList, p)
			}
		}
		a.GetSvcProfileV2(svcName).DevPortForwardList = newList

		//a.GetSvcProfileV2(svcName).DevPortForwardList = append(a.GetSvcProfileV2(svcName).DevPortForwardList[:indexToDelete], a.GetSvcProfileV2(svcName).DevPortForwardList[indexToDelete+1:]...)
		err := a.SaveProfile()
		if err != nil {
			log.WarnE(err, err.Error())
		}
	}
}
