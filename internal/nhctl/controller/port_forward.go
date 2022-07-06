/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"fmt"
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/profile"
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
			client, err := daemon_client.GetDaemonClient(portForward.Sudo)
			if err != nil {
				return err
			}
			return client.SendStopPortForwardCommand(
				&model.NocalHostResource{
					NameSpace:   c.NameSpace,
					Nid:         c.AppMeta.NamespaceId,
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

func StopPortForward(ns, nid, app, svc string, portForward *profile.DevPortForward) error {
	client, err := daemon_client.GetDaemonClient(portForward.Sudo)
	if err != nil {
		return err
	}
	return client.SendStopPortForwardCommand(
		&model.NocalHostResource{
			NameSpace:   ns,
			Nid:         nid,
			Application: app,
			Service:     svc,
			ServiceType: portForward.ServiceType,
			PodName:     portForward.PodName,
		}, portForward.LocalPort, portForward.RemotePort,
	)
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
			for _, portForward := range svcProfile.DevPortForwardList {
				if portForward.LocalPort == localPort && portForward.RemotePort == remotePort {
					portForward.Status = portStatus
					portForward.Reason = reason
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
	svcProfile, err := c.GetProfile()
	if err != nil {
		return nil, err
	}
	for _, pf := range svcProfile.DevPortForwardList {
		if pf.LocalPort == localPort && pf.RemotePort == remotePort {
			return pf, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("Pf %d:%d not found", localPort, remotePort))
}

func (c *Controller) PortForwardAfterDevStart(podName, containerName string) error {

	p := c.Config()

	if p.ContainerConfigs == nil {
		return nil
	}

	cc := p.GetContainerDevConfigOrDefault(containerName)
	if cc == nil {
		return nil
	}

	for _, pf := range cc.PortForward {
		lPort, rPort, err := utils.GetPortForwardForString(pf)
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
	client, err := daemon_client.GetDaemonClient(isAdmin)
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

	if err = client.SendStartPortForwardCommand(nhResource, localPort, remotePort, role, c.AppMeta.NamespaceId); err != nil {
		return err
	} else {
		return c.SetPortForwardedStatus(true) //  todo: move port-forward start
	}
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
