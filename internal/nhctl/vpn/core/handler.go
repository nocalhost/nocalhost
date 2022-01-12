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

type Handler interface {
	Init(options ...HandlerOptionFunc)
	Handle(context.Context, net.Conn)
}

type HandlerOptions struct {
	Chain    *Chain
	Node     *Node
	IPRoutes []tun.IPRoute
}

type HandlerOptionFunc func(opts *HandlerOptions)

func ChainHandlerOption(chain *Chain) HandlerOptionFunc {
	return func(opts *HandlerOptions) {
		opts.Chain = chain
	}
}

func NodeHandlerOption(node *Node) HandlerOptionFunc {
	return func(opts *HandlerOptions) {
		opts.Node = node
	}
}

func IPRoutesHandlerOption(routes ...tun.IPRoute) HandlerOptionFunc {
	return func(opts *HandlerOptions) {
		opts.IPRoutes = routes
	}
}
