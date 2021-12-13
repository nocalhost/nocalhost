/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package core

import (
	"context"
	"net"
	"nocalhost/internal/nhctl/vpn/tun"
)

// Handler is a proxy server handler
type Handler interface {
	Init(options ...HandlerOptionFunc)
	Handle(context.Context, net.Conn)
}

// HandlerOptions describes the options for Handler.
type HandlerOptions struct {
	Chain    *Chain
	Node     *Node
	IPRoutes []tun.IPRoute
}

// HandlerOptionFunc allows a common way to set handler options.
type HandlerOptionFunc func(opts *HandlerOptions)

// ChainHandlerOption sets the Chain option of HandlerOptions.
func ChainHandlerOption(chain *Chain) HandlerOptionFunc {
	return func(opts *HandlerOptions) {
		opts.Chain = chain
	}
}

// NodeHandlerOption set the server node for server handler.
func NodeHandlerOption(node *Node) HandlerOptionFunc {
	return func(opts *HandlerOptions) {
		opts.Node = node
	}
}

// IPRoutesHandlerOption sets the IP routes for tun tunnel.
func IPRoutesHandlerOption(routes ...tun.IPRoute) HandlerOptionFunc {
	return func(opts *HandlerOptions) {
		opts.IPRoutes = routes
	}
}
