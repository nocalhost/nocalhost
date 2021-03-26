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
	"fmt"
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/internal/nhctl/syncthing/terminate"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

//func (a *Application) AppendPortForward(svcName string, devPortForward *profile.DevPortForward) {
//	a.GetSvcProfileV2(svcName).DevPortForwardList = append(a.GetSvcProfileV2(svcName).DevPortForwardList, devPortForward)
//}

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
	profileV2, err := profile.NewAppProfileV2(a.NameSpace, a.Name, false)
	if err != nil {
		return err
	}
	defer profileV2.CloseDb()

	svcProfile := profileV2.FetchSvcProfileV2FromProfile(svcName)
	if svcProfile == nil {
		return errors.New("Failed to get svc profile")
	}

	for _, portForward := range svcProfile.DevPortForwardList {
		if portForward.LocalPort == localPort && portForward.RemotePort == remotePort {
			portForward.Status = portStatus
			portForward.Reason = reason
			portForward.Pid = os.Getpid()
			portForward.Updated = time.Now().Format("2006-01-02 15:04:05")
			break
		}
	}

	var wg sync.WaitGroup{}
	wg.Wait()
	return profileV2.Save()
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
				return client.SendPortForwardCommand(&model.NocalHostResource{
					NameSpace:   a.NameSpace,
					Application: a.Name,
					Service:     svcName,
					PodName:     "",
				}, localPort, remotePort, command.StopPortForward)
			} else {
				log.Infof("Kill %v", *portForward)
				err := terminate.Terminate(portForward.Pid, true, "port-forward")
				if err != nil {
					return errors.Wrap(err, "")
				}
				indexToDelete = index
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

func (a *Application) PortForwardAfterDevStart(svcName string, containerName string, svcType SvcType) error {
	switch svcType {
	case Deployment:
		p := a.GetSvcProfileV2(svcName)
		if p.ContainerConfigs == nil {
			return nil
		}
		cc := p.GetContainerDevConfigOrDefault(containerName)
		if cc == nil {
			return nil
		}
		podName, err := a.GetNocalhostDevContainerPod(svcName)
		if err != nil {
			return err
		}
		for _, pf := range cc.PortForward {
			lPort, rPort, err := GetPortForwardForString(pf)
			if err != nil {
				log.WarnE(err, "")
				continue
			}
			log.Infof("Forwarding %d:%d", lPort, rPort)
			if err = a.PortForward(svcName, podName, lPort, rPort); err != nil {
				log.WarnE(err, "")
			}
		}
	default:
		return errors.New("SvcType not supported")
	}
	return nil
}

// portStr is like 8080:80 or :80
func GetPortForwardForString(portStr string) (int, int, error) {
	var err error
	s := strings.Split(portStr, ":")
	if len(s) < 2 {
		return 0, 0, errors.New(fmt.Sprintf("Wrong format of port: %s", portStr))
	}
	var localPort, remotePort int
	sLocalPort := s[0]
	if sLocalPort == "" {
		// get random port in local
		if localPort, err = ports.GetAvailablePort(); err != nil {
			return 0, 0, err
		}
	} else if localPort, err = strconv.Atoi(sLocalPort); err != nil {
		return 0, 0, errors.Wrap(err, fmt.Sprintf("Wrong format of local port: %s.", sLocalPort))
	}
	if remotePort, err = strconv.Atoi(s[1]); err != nil {
		return 0, 0, errors.Wrap(err, fmt.Sprintf("wrong format of remote port: %s, skipped", s[1]))
	}
	return localPort, remotePort, nil
}
