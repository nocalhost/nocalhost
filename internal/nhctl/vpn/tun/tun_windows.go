package tun

import (
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/sys/windows"
	wireguardtun "golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
	"net"
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
	if err = luid.AddIPAddress(net.IPNet{IP: ip, Mask: ipNet.Mask}); err != nil {
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
		if gw != "" {
			route.Gateway = net.ParseIP(gw)
		} else {
			route.Gateway = net.IPv4(0, 0, 0, 0)
		}
		if err := ifName.AddRoute(*route.Dest, route.Gateway, 0); err != nil {
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
	err := c.ifce.Close()
	if name, err := c.ifce.Name(); err == nil {
		if wt, err := wireguardtun.WintunPool.OpenAdapter(name); err == nil {
			_, err = wt.Delete(true)
		}
	}
	return err
}

func (c *winTunConn) Read(b []byte) (n int, err error) {
	return c.ifce.Read(b, 0)
}

func (c *winTunConn) Write(b []byte) (n int, err error) {
	return c.ifce.Write(b, 0)
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
