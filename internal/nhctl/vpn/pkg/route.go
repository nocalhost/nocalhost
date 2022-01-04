/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package pkg

import (
	"context"
	"crypto/tls"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"nocalhost/internal/nhctl/vpn/core"
	"nocalhost/internal/nhctl/vpn/tlsconfig"
	"nocalhost/internal/nhctl/vpn/tun"
	"strings"
)

type Route struct {
	ServeNodes []string // -L tun
	ChainNode  string   // -F tcp
	Retries    int
}

func (r *Route) parseChain() (*core.Chain, error) {
	// parse the base nodes
	node, err := parseChainNode(r.ChainNode)
	if err != nil {
		return nil, err
	}
	return core.NewChain(r.Retries, node), nil
}

func parseChainNode(ns string) (*core.Node, error) {
	node, err := core.ParseNode(ns)
	if err != nil {
		return nil, err
	}
	node.Client = &core.Client{
		Connector:   core.UDPOverTCPTunnelConnector(),
		Transporter: core.TCPTransporter(),
	}
	return node, nil
}

func (r *Route) GenRouters() ([]router, error) {
	chain, err := r.parseChain()
	if err != nil && !errors.Is(err, core.ErrorInvalidNode) {
		return nil, err
	}

	routers := make([]router, 0, len(r.ServeNodes))
	for _, serveNode := range r.ServeNodes {
		node, err := core.ParseNode(serveNode)
		if err != nil {
			return nil, err
		}

		tunRoutes := parseIPRoutes(node.Get("route"))
		gw := net.ParseIP(node.Get("gw")) // default gateway
		for i := range tunRoutes {
			if tunRoutes[i].Gateway == nil {
				tunRoutes[i].Gateway = gw
			}
		}

		var ln net.Listener
		switch node.Transport {
		case "tcp":
			tcpListener, _ := core.TCPListener(node.Addr)
			ln = tls.NewListener(tcpListener, tlsconfig.Server)
		case "tun":
			config := tun.Config{
				Name:    node.Get("name"),
				Addr:    node.Get("net"),
				MTU:     node.GetInt("mtu"),
				Routes:  tunRoutes,
				Gateway: node.Get("gw"),
			}
			ln, err = tun.Listener(config)
		}
		if err != nil {
			return nil, err
		}

		var handler core.Handler
		switch node.Protocol {
		case "tun":
			handler = core.TunHandler()
			handler.Init(
				core.ChainHandlerOption(chain),
				core.NodeHandlerOption(node),
				core.IPRoutesHandlerOption(tunRoutes...),
			)
		default:
			handler = core.TCPHandler()
		}

		routers = append(routers, router{
			node:   node,
			server: &core.Server{Listener: ln, Handler: handler},
		})
	}
	return routers, nil
}

type router struct {
	node   *core.Node
	server *core.Server
}

func (r *router) Serve(ctx context.Context) error {
	log.Debugf("%s on %s", r.node.Protocol, r.server.Addr())
	return r.server.Serve(ctx, r.server.Handler)
}

func (r *router) Close() error {
	if r == nil || r.server == nil {
		return nil
	}
	return r.server.Close()
}

func parseIPRoutes(routeStringList string) (routes []tun.IPRoute) {
	if len(routeStringList) == 0 {
		return
	}

	routeList := strings.Split(routeStringList, ",")
	for _, route := range routeList {
		if _, ipNet, _ := net.ParseCIDR(strings.TrimSpace(route)); ipNet != nil {
			routes = append(routes, tun.IPRoute{Dest: ipNet})
		}
	}
	return
}
