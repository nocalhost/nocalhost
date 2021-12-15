/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package tun

import (
	"errors"
	"fmt"
	"net"
	"nocalhost/internal/nhctl/vpn/util"
	"syscall"

	"github.com/docker/libcontainer/netlink"
	"github.com/milosgajdos/tenus"
	log "github.com/sirupsen/logrus"
	"github.com/songgao/water"
)

func createTun(cfg Config) (conn net.Conn, itf *net.Interface, err error) {
	ip, ipNet, err := net.ParseCIDR(cfg.Addr)
	if err != nil {
		return
	}

	ifce, err := water.New(water.Config{
		DeviceType: water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{
			Name: cfg.Name,
		},
	})
	if err != nil {
		return
	}

	link, err := tenus.NewLinkFrom(ifce.Name())
	if err != nil {
		return
	}

	mtu := cfg.MTU
	if mtu <= 0 {
		mtu = util.DefaultMTU
	}

	cmd := fmt.Sprintf("ip link set dev %s mtu %d", ifce.Name(), mtu)
	log.Debug("[tun]", cmd)
	if er := link.SetLinkMTU(mtu); er != nil {
		err = fmt.Errorf("%s: %v", cmd, er)
		return
	}

	cmd = fmt.Sprintf("ip address add %s dev %s", cfg.Addr, ifce.Name())
	log.Debug("[tun]", cmd)
	if er := link.SetLinkIp(ip, ipNet); er != nil {
		err = fmt.Errorf("%s: %v", cmd, er)
		return
	}

	cmd = fmt.Sprintf("ip link set dev %s up", ifce.Name())
	log.Debug("[tun]", cmd)
	if er := link.SetLinkUp(); er != nil {
		err = fmt.Errorf("%s: %v", cmd, er)
		return
	}

	if err = addTunRoutes(ifce.Name(), cfg.Routes...); err != nil {
		return
	}

	itf, err = net.InterfaceByName(ifce.Name())
	if err != nil {
		return
	}

	conn = &tunConn{
		ifce: ifce,
		addr: &net.IPAddr{IP: ip},
	}
	return
}

func addTunRoutes(ifName string, routes ...IPRoute) error {
	for _, route := range routes {
		if route.Dest == nil {
			continue
		}
		cmd := fmt.Sprintf("ip route add %s dev %s", route.Dest.String(), ifName)
		log.Debugf("[tun] %s", cmd)
		if err := netlink.AddRoute(route.Dest.String(), "", "", ifName); err != nil && !errors.Is(err, syscall.EEXIST) {
			return fmt.Errorf("%s: %v", cmd, err)
		}
	}
	return nil
}
