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
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/syncthing/terminate"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"time"
)

func (a *Application) AppendPortForward(svcName string, devPortForward *profile.DevPortForward) {
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
		newPF := &profile.DevPortForward{
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

	for _, portForward := range a.GetSvcProfileV2(svcName).DevPortForwardList {
		if portForward.LocalPort == localPort && portForward.RemotePort == remotePort {
			portForward.Status = portStatus
			portForward.Reason = reason
			portForward.Pid = os.Getpid()
			portForward.Updated = time.Now().Format("2006-01-02 15:04:05")
			break
		}
	}
	return a.SaveProfile()
}

func (a *Application) EndDevPortForward(svcName string, localPort int, remotePort int) error {

	svcProfile := a.GetSvcProfileV2(svcName)

	indexToDelete := -1
	for index, portForward := range svcProfile.DevPortForwardList {
		if portForward.LocalPort == localPort && portForward.RemotePort == remotePort {
			if portForward.RunByDaemonServer {
				isAdmin := utils.IsSudoUser()
				client, err := daemon_client.NewDaemonClient(isAdmin)
				if err != nil {
					return err
				}
				err = client.SendPortForwardCommand(&model.NocalHostResource{
					NameSpace:   a.NameSpace,
					Application: a.Name,
					Service:     svcName,
					PodName:     "",
				}, localPort, remotePort, command.StopPortForward)
				if err != nil {
					return err
				}

			} else {
				log.Infof("Kill %v", *portForward)
				err := terminate.Terminate(portForward.Pid, true, "port-forward")
				if err != nil {
					return errors.Wrap(err, "")
				}
				indexToDelete = index
				time.Sleep(2 * time.Second)
			}
			break
		}
	}

	// remove portForward from DevPortForwardList
	if indexToDelete > -1 {

		originList := svcProfile.DevPortForwardList
		newList := make([]*profile.DevPortForward, 0)
		for index, p := range originList {
			if index != indexToDelete {
				newList = append(newList, p)
			}
		}
		svcProfile.DevPortForwardList = newList

		return a.SaveProfile()
	}

	return nil
}
