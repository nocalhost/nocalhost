/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package core

import (
	"context"
	log "github.com/sirupsen/logrus"
	"net"
)

type Server struct {
	Listener net.Listener
	Handler  Handler
}

func (s *Server) Addr() net.Addr {
	return s.Listener.Addr()
}

func (s *Server) Close() error {
	return s.Listener.Close()
}

func (s *Server) Serve(ctx context.Context, h Handler) error {
	l := s.Listener
	go func() {
		select {
		case <-ctx.Done():
			if err := l.Close(); err != nil {
				log.Warnf("error while close listener, err: %v", err)
			}
		}
	}()
	for ctx.Err() == nil {
		conn, err := l.Accept()
		if err == nil {
			go h.Handle(ctx, conn)
		}
	}
	return nil
}
