/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package server

import (
	"context"
	"fmt"
	"github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	"github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	"github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	"github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	"github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	"github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	"log"
	"net"
	"time"

	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

const (
	grpcKeepaliveTime        = 30 * time.Second
	grpcKeepaliveTimeout     = 5 * time.Second
	grpcKeepaliveMinTime     = 30 * time.Second
	grpcMaxConcurrentStreams = 1000 * 000
)

// RunManagementServer starts an xDS server at the given port.
func RunManagementServer(ctx context.Context, srv server.Server, port uint) {
	// gRPC golang library sets a very small upper bound for the number gRPC/h2
	// streams over a single TCP connection. If a proxy multiplexes requests over
	// a single connection to the management server, then it might lead to
	// availability problems. Keepalive timeouts based on connection_keepalive parameter https://www.envoyproxy.io/docs/envoy/latest/configuration/overview/examples#dynamic
	var grpcOptions []grpc.ServerOption
	grpcOptions = append(grpcOptions,
		grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    grpcKeepaliveTime,
			Timeout: grpcKeepaliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             grpcKeepaliveMinTime,
			PermitWithoutStream: true,
		}),
	)
	grpcServer := grpc.NewServer(grpcOptions...)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
	}

	RegisterServer(grpcServer, srv)

	log.Printf("management server listening on %d\n", port)
	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			log.Println(err)
		}
	}()
	<-ctx.Done()

	grpcServer.GracefulStop()
}

// RegisterServer registers with v3 services.
func RegisterServer(grpcServer *grpc.Server, server server.Server) {
	// register services
	envoy_service_discovery_v3.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	envoy_service_endpoint_v3.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	envoy_service_cluster_v3.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	envoy_service_route_v3.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	envoy_service_route_v3.RegisterScopedRoutesDiscoveryServiceServer(grpcServer, server)
	envoy_service_listener_v3.RegisterListenerDiscoveryServiceServer(grpcServer, server)
	envoy_service_secret_v3.RegisterSecretDiscoveryServiceServer(grpcServer, server)
	envoy_service_runtime_v3.RegisterRuntimeDiscoveryServiceServer(grpcServer, server)
}
