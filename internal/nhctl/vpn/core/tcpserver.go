/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package core

import (
	"bytes"
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"nocalhost/internal/nhctl/vpn/util"
	"time"
)

type fakeUDPTunConnector struct {
}

// UDPOverTCPTunnelConnector creates a connector for UDP-over-TCP
func UDPOverTCPTunnelConnector() Connector {
	return &fakeUDPTunConnector{}
}

func (c *fakeUDPTunConnector) Connect(ctx context.Context, conn net.Conn, network, address string) (net.Conn, error) {
	switch network {
	case "tcp", "tcp4", "tcp6":
		return nil, fmt.Errorf("%s unsupported", network)
	}
	_ = conn.SetDeadline(time.Time{})
	targetAddr, _ := net.ResolveUDPAddr("udp", address)
	return newFakeUDPTunnelConnOverTCP(conn, targetAddr)
}

type fakeUdpHandler struct {
}

func TCPHandler() Handler {
	return &fakeUdpHandler{}
}

func (h *fakeUdpHandler) Init(...HandlerOptionFunc) {
}

func (h *fakeUdpHandler) Handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	if util.Debug {
		log.Debugf("[tcpserver] %s -> %s\n", conn.RemoteAddr(), conn.LocalAddr())
	}
	h.handleUDPTunnel(ctx, conn)
}

func (h *fakeUdpHandler) transportUDP(relay, peer net.PacketConn) (err error) {
	errChan := make(chan error, 2)
	var clientAddr net.Addr
	go func() {
		b := util.MPool.Get().([]byte)
		defer util.MPool.Put(b)

		for {
			n, laddr, err := relay.ReadFrom(b)
			if err != nil {
				errChan <- err
				return
			}
			if clientAddr == nil {
				clientAddr = laddr
			}
			dgram, err := ReadDatagramPacket(bytes.NewReader(b[:n]))
			if err != nil {
				log.Errorln(err)
				errChan <- err
				return
			}

			raddr, err := net.ResolveUDPAddr("udp", dgram.Addr())
			if err != nil {
				log.Debugf("[tcpserver-udp] addr error, addr: %s, err: %v", dgram.Addr(), err)
				continue // drop silently
			}
			if _, err := peer.WriteTo(dgram.Data, raddr); err != nil {
				errChan <- err
				return
			}
			if util.Debug {
				log.Debugf("[tcpserver-udp] %s >>> %s length: %d", relay.LocalAddr(), raddr, len(dgram.Data))
			}
		}
	}()

	go func() {
		b := util.MPool.Get().([]byte)
		defer util.MPool.Put(b)

		for {
			n, raddr, err := peer.ReadFrom(b)
			if err != nil {
				errChan <- err
				return
			}
			if clientAddr == nil {
				continue
			}
			buf := bytes.Buffer{}
			dgram := NewDatagramPacket(raddr, b[:n])
			_ = dgram.Write(&buf)
			if _, err := relay.WriteTo(buf.Bytes(), clientAddr); err != nil {
				errChan <- err
				return
			}
			if util.Debug {
				log.Debugf("[tcpserver-udp] %s <<< %s length: %d", relay.LocalAddr(), raddr, len(dgram.Data))
			}
		}
	}()

	return <-errChan
}

func (h *fakeUdpHandler) handleUDPTunnel(ctx context.Context, conn net.Conn) {
	// serve tunnel udp, tunnel <-> remote, handle tunnel udp request
	bindAddr, _ := net.ResolveUDPAddr("udp", ":0")
	uc, err := net.ListenUDP("udp", bindAddr)
	if err != nil {
		log.Debugf("[tcpserver] udp-tun %s -> %s : %s", conn.RemoteAddr(), bindAddr, err)
		return
	}
	defer uc.Close()
	if util.Debug {
		log.Debugf("[tcpserver] udp-tun %s <- %s\n", conn.RemoteAddr(), uc.LocalAddr())
	}
	log.Debugf("[tcpserver] udp-tun %s <-> %s", conn.RemoteAddr(), uc.LocalAddr())
	_ = h.tunnelServerUDP(ctx, conn, uc)
	log.Debugf("[tcpserver] udp-tun %s >-< %s", conn.RemoteAddr(), uc.LocalAddr())
	return
}

func (h *fakeUdpHandler) tunnelServerUDP(ctx context.Context, cc net.Conn, pc net.PacketConn) (err error) {
	errChan := make(chan error, 2)

	go func() {
		b := util.MPool.Get().([]byte)
		defer util.MPool.Put(b)

		for {
			n, addr, err := pc.ReadFrom(b)
			if err != nil {
				log.Debugf("[udp-tun] %s : %s", cc.RemoteAddr(), err)
				errChan <- err
				return
			}

			// pipe from peer to tunnel
			datagramPacket := NewDatagramPacket(addr, b[:n])
			if err = datagramPacket.Write(cc); err != nil {
				log.Debugf("[tcpserver] udp-tun %s <- %s : %s", cc.RemoteAddr(), datagramPacket.Addr(), err)
				errChan <- err
				return
			}
			if util.Debug {
				log.Debugf("[tcpserver] udp-tun %s <<< %s length: %d", cc.RemoteAddr(), datagramPacket.Addr(), len(datagramPacket.Data))
			}
		}
	}()

	go func() {
		for {
			datagramPacket, err := ReadDatagramPacket(cc)
			if err != nil {
				log.Debugf("[udp-tun] %s -> 0 : %v", cc.RemoteAddr(), err)
				errChan <- err
				return
			}

			// pipe from tunnel to peer
			addr, err := net.ResolveUDPAddr("udp", datagramPacket.Addr())
			if err != nil {
				log.Debugf("[tcpserver-udp] addr error, addr: %s, err: %v", datagramPacket.Addr(), err)
				continue // drop silently
			}
			if _, err = pc.WriteTo(datagramPacket.Data, addr); err != nil {
				log.Debugf("[tcpserver] udp-tun %s -> %s : %s", cc.RemoteAddr(), addr, err)
				errChan <- err
				return
			}
			if util.Debug {
				log.Debugf("[tcpserver] udp-tun %s >>> %s length: %d", cc.RemoteAddr(), addr, len(datagramPacket.Data))
			}
		}
	}()

	return <-errChan
}

// fake udp connect over tcp
type fakeUDPTunnelConn struct {
	// tcp connection
	net.Conn
	targetAddr net.Addr
}

func newFakeUDPTunnelConnOverTCP(conn net.Conn, targetAddr net.Addr) (net.Conn, error) {
	return &fakeUDPTunnelConn{
		Conn:       conn,
		targetAddr: targetAddr,
	}, nil
}

func (c *fakeUDPTunnelConn) Read(b []byte) (n int, err error) {
	n, _, err = c.ReadFrom(b)
	return
}

func (c *fakeUDPTunnelConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	dgram, err := ReadDatagramPacket(c.Conn)
	if err != nil {
		log.Errorln(err)
		return
	}
	n = copy(b, dgram.Data)
	addr, err = net.ResolveUDPAddr("udp", dgram.Addr())
	if err != nil {
		log.Debugf("[tcpserver-udp] addr error, addr: %s, err: %v", dgram.Addr(), err)
	}
	return
}

func (c *fakeUDPTunnelConn) Write(b []byte) (n int, err error) {
	return c.WriteTo(b, c.targetAddr)
}

func (c *fakeUDPTunnelConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	dgram := NewDatagramPacket(addr, b)
	if err = dgram.Write(c.Conn); err != nil {
		return
	}
	return len(b), nil
}

func (c *fakeUDPTunnelConn) Close() error {
	return c.Conn.Close()
}

func (c *fakeUDPTunnelConn) CloseWrite() error {
	if cc, ok := c.Conn.(interface{ CloseWrite() error }); ok {
		return cc.CloseWrite()
	}
	return nil
}

func (c *fakeUDPTunnelConn) CloseRead() error {
	if cc, ok := c.Conn.(interface{ CloseRead() error }); ok {
		return cc.CloseRead()
	}
	return nil
}
