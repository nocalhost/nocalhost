/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package core

import (
	"context"
	"net"
)

// Client is a proxy client.
// A client is divided into two layers: connector and transporter.
// Connector is responsible for connecting to the destination address through this proxy.
// Transporter performs a handshake with this proxy.
type Client struct {
	Connector
	Transporter
}

// Connector is responsible for connecting to the destination address.
type Connector interface {
	ConnectContext(ctx context.Context, conn net.Conn, network, address string) (net.Conn, error)
}

// Transporter is responsible for handshaking with the proxy server.
type Transporter interface {
	Dial(addr string) (net.Conn, error)
}
