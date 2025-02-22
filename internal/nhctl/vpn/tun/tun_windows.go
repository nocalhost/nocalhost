/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package tun

import (
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/sys/windows"
	wireguardtun "golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
	"net"
	"net/netip"
	"os"
	"time"
)

func createTun(cfg Config) (net.Conn, *net.Interface, error) {
	ip, ipNet, err := net.ParseCIDR(cfg.Addr)
	if err != nil {
		return nil, nil, err
	}
	interfaceName := "wg1"
	if len(cfg.Name) != 0 {
		interfaceName = cfg.Name
	}
	tunDevice, err := wireguardtun.CreateTUN(interfaceName, cfg.MTU)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create TUN device: %w", err)
	}
	_ = os.Setenv("luid", fmt.Sprintf("%d", tunDevice.(*wireguardtun.NativeTun).LUID()))

	luid := winipcfg.LUID(tunDevice.(*wireguardtun.NativeTun).LUID())
	ipAddr, err := netip.ParseAddr(ip.String())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse IP address: %w", err)
	}
	ones, _ := ipNet.Mask.Size()
	prefix := netip.PrefixFrom(ipAddr, ones)
	if err = luid.AddIPAddress(prefix); err != nil {
		return nil, nil, err
	}

	if err = addTunRoutes(luid, cfg.Gateway, cfg.Routes...); err != nil {
		return nil, nil, err
	}

	row2, _ := luid.Interface()
	iface, _ := net.InterfaceByIndex(int(row2.InterfaceIndex))
	return &winTunConn{ifce: tunDevice, addr: &net.IPAddr{IP: ip}}, iface, nil
}

func addTunRoutes(ifName winipcfg.LUID, gw string, routes ...IPRoute) error {
	_ = ifName.FlushRoutes(windows.AF_INET)
	for _, route := range routes {
		if route.Dest == nil {
			continue
		}

		destIP, err := netip.ParseAddr(route.Dest.IP.String())
		if err != nil {
			return fmt.Errorf("failed to parse destination IP: %w", err)
		}
		ones, _ := route.Dest.Mask.Size()
		destPrefix := netip.PrefixFrom(destIP, ones)

		var gwAddr netip.Addr
		if gw != "" {
			gwAddr, err = netip.ParseAddr(gw)
			if err != nil {
				return fmt.Errorf("failed to parse gateway IP: %w", err)
			}
		} else {
			gwAddr = netip.IPv4Unspecified()
		}

		if err := ifName.AddRoute(destPrefix, gwAddr, 0); err != nil {
			return err
		}
	}
	return nil
}

type winTunConn struct {
	ifce wireguardtun.Device
	addr net.Addr
}

func (c *winTunConn) Close() error {
	if nativeTun, ok := c.ifce.(*wireguardtun.NativeTun); ok {
		nativeTun.Close()
	}
	return c.ifce.Close()
}

func (c *winTunConn) Read(b []byte) (n int, err error) {
	sizes := make([]int, 1)
	n, err = c.ifce.Read([][]byte{b}, sizes, 0)
	if err != nil {
		return 0, err
	}
	return sizes[0], nil
}

func (c *winTunConn) Write(b []byte) (n int, err error) {
	return c.ifce.Write([][]byte{b}, 0)
}

func (c *winTunConn) LocalAddr() net.Addr {
	return c.addr
}

func (c *winTunConn) RemoteAddr() net.Addr {
	return &net.IPAddr{}
}

func (c *winTunConn) SetDeadline(time.Time) error {
	return &net.OpError{Op: "set", Net: "tun", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (c *winTunConn) SetReadDeadline(time.Time) error {
	return &net.OpError{Op: "set", Net: "tun", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (c *winTunConn) SetWriteDeadline(time.Time) error {
	return &net.OpError{Op: "set", Net: "tun", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}
