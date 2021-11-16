/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package core

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
)

var (
	// ErrorEmptyChain is an error that implies the chain is empty.
	ErrorEmptyChain = errors.New("empty chain")
)

// Chain is a proxy chain that holds a list of proxy node groups.
type Chain struct {
	Retries int
	node    *Node
}

// NewChain creates a proxy chain with a list of proxy nodes.
// It creates the node groups automatically, one group per node.
func NewChain(retry int, node *Node) *Chain {
	return &Chain{Retries: retry, node: node}
}

func (c *Chain) Node() *Node {
	return c.node
}

// IsEmpty checks if the chain is empty.
// An empty chain means that there is no proxy node or node group in the chain.
func (c *Chain) IsEmpty() bool {
	return c == nil || c.node == nil
}

// DialContext connects to the address on the named network using the provided context.
func (c *Chain) DialContext(ctx context.Context, network, address string) (conn net.Conn, err error) {
	for i := 0; i < int(math.Max(float64(1), float64(c.Retries))); i++ {
		conn, err = c.dial(ctx, network, address)
		if err == nil {
			break
		}
	}
	return
}

func (c *Chain) dial(ctx context.Context, network, address string) (net.Conn, error) {
	ipAddr := address
	if address != "" {
		ipAddr = c.resolve(address)
	}

	if c.IsEmpty() {
		return nil, ErrorEmptyChain
	}

	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}

	cc, err := c.Node().Client.ConnectContext(ctx, conn, network, ipAddr)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return cc, nil
}

func (*Chain) resolve(addr string) string {
	if host, port, err := net.SplitHostPort(addr); err == nil {
		if ips, err := net.LookupIP(host); err == nil && len(ips) > 0 {
			return fmt.Sprintf("%s:%s", ips[0].String(), port)
		}
	}
	return addr
}

// getConn obtains a connection to the last node of the chain.
func (c *Chain) getConn(_ context.Context) (net.Conn, error) {
	if c.IsEmpty() {
		return nil, ErrorEmptyChain
	}
	return c.Node().Client.Dial(c.Node().Addr)
}
