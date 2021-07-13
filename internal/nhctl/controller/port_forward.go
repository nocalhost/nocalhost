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

package controller

import (
	"fmt"
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"strconv"
	"strings"
	"time"
)

func (c *Controller) EndDevPortForward(localPort int, remotePort int) error {

	svcProfile, err := c.GetProfile()
	if err != nil {
		return err
	}

	for _, portForward := range svcProfile.DevPortForwardList {
		if portForward.LocalPort == localPort && portForward.RemotePort == remotePort {
			client, err := daemon_client.NewDaemonClient(portForward.Sudo)
			if err != nil {
				return err
			}
			return client.SendStopPortForwardCommand(
				&model.NocalHostResource{
					NameSpace:   c.NameSpace,
					Application: c.AppName,
					Service:     c.Name,
					ServiceType: string(c.Type),
					PodName:     "",
				}, localPort, remotePort,
			)
		}
	}
	return nil
}

func (c *Controller) StopAllPortForward() error {
	svcProfile, err := c.GetProfile()
	if err != nil {
		return err
	}

	for _, portForward := range svcProfile.DevPortForwardList {
		utils.Should(c.EndDevPortForward(portForward.LocalPort, portForward.RemotePort))
	}
	return nil
}

// StopPortForwardByPort port format 8080:80
func (c *Controller) StopPortForwardByPort(port string) error {

	ports := strings.Split(port, ":")
	localPort, err := strconv.Atoi(ports[0])
	if err != nil {
		return errors.Wrap(err, "")
	}
	remotePort, err := strconv.Atoi(ports[1])
	if err != nil {
		return errors.Wrap(err, "")
	}
	return c.EndDevPortForward(localPort, remotePort)
}

func (c *Controller) UpdatePortForwardStatus(localPort int, remotePort int, portStatus string, reason string) error {
	pf, err := c.GetPortForward(localPort, remotePort)
	if err != nil {
		return err
	}

	if pf.Status == portStatus {
		log.Logf(
			"Pf %d:%d's status is already %s, no need to update",
			pf.LocalPort, pf.RemotePort, pf.Status,
		)
		return nil
	}

	return c.UpdateSvcProfile(
		func(svcProfile *profile.SvcProfileV2) error {
			if svcProfile == nil {
				return errors.New("Failed to get controller profile")
			}

			for _, portForward := range svcProfile.DevPortForwardList {
				if portForward.LocalPort == localPort && portForward.RemotePort == remotePort {
					portForward.Status = portStatus
					portForward.Reason = reason
					//portForward.Pid = os.Getpid()
					portForward.Updated = time.Now().Format("2006-01-02 15:04:05")
					break
				}
			}
			return nil
		},
	)
}

// GetPortForward If not found return err
func (c *Controller) GetPortForward(localPort, remotePort int) (*profile.DevPortForward, error) {
	var err error
	svcProfile, err := c.GetProfile()
	if err != nil {
		return nil, err
	}
	for _, pf := range svcProfile.DevPortForwardList {
		if pf.LocalPort == localPort && pf.RemotePort == remotePort {
			return pf, nil
		}
	}
	log.Logf("type %s, name %s", c.Type, svcProfile.ActualName)
	return nil, errors.New(fmt.Sprintf("Pf %d:%d not found", localPort, remotePort))
}

//func (c *Controller) CheckPidPortStatus(ctx context.Context, sLocalPort, sRemotePort int, lock *sync.Mutex) {
//	for {
//		select {
//		case <-ctx.Done():
//			log.Info("Stop Checking port status")
//			return
//		default:
//			portStatus := port_forward.PidPortStatus(os.Getpid(), sLocalPort)
//			log.Infof("Checking Port %d:%d's status: %s", sLocalPort, sRemotePort, portStatus)
//			lock.Lock()
//			_ = c.UpdatePortForwardStatus(sLocalPort, sRemotePort, portStatus, "Check Pid")
//			lock.Unlock()
//			<-time.After(2 * time.Minute)
//		}
//	}
//}

func (c *Controller) PortForwardAfterDevStart(podName, containerName string) error {

	profileV2, err := c.GetProfile()
	if err != nil {
		return err
	}

	p := profileV2
	if p.ContainerConfigs == nil {
		return nil
	}
	cc := p.GetContainerDevConfigOrDefault(containerName)
	if cc == nil {
		return nil
	}
	//podName, err := c.GetNocalhostDevContainerPod()
	//if err != nil {
	//	return err
	//}
	for _, pf := range cc.PortForward {
		lPort, rPort, err := GetPortForwardForString(pf)
		if err != nil {
			log.WarnE(err, "")
			continue
		}
		log.Infof("Forwarding %d:%d", lPort, rPort)
		utils.Should(c.PortForward(podName, lPort, rPort, ""))
	}
	return nil
}

// PortForward Role: If set to "SYNC", means it is a pf used for syncthing
func (c *Controller) PortForward(podName string, localPort, remotePort int, role string) error {

	isAdmin := utils.IsSudoUser()
	client, err := daemon_client.NewDaemonClient(isAdmin)
	if err != nil {
		return err
	}
	nhResource := &model.NocalHostResource{
		NameSpace:   c.NameSpace,
		Application: c.AppName,
		Service:     c.Name,
		ServiceType: c.Type.String(),
		PodName:     podName,
	}

	if err = client.SendStartPortForwardCommand(nhResource, localPort, remotePort, role); err != nil {
		return err
	} else {
		log.Infof("Port-forward %d:%d has been started", localPort, remotePort)
		return c.SetPortForwardedStatus(true) //  todo: move port-forward start
	}
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

func (c *Controller) CheckIfPortForwardExists(localPort, remotePort int) (bool, error) {
	svcProfile, err := c.GetProfile()
	if err != nil {
		return false, err
	}
	for _, portForward := range svcProfile.DevPortForwardList {
		if portForward.LocalPort == localPort && portForward.RemotePort == remotePort {
			return true, nil
		}
	}
	return false, nil
}
