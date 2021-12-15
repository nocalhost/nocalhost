/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package core

import (
	"context"
	"net"
)

type Client struct {
	Connector
	Transporter
}

type Connector interface {
	ConnectContext(ctx context.Context, conn net.Conn, network, address string) (net.Conn, error)
}

type Transporter interface {
	Dial(addr string) (net.Conn, error)
}
