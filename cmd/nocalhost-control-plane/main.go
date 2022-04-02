package main

import (
	"context"
	"flag"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"nocalhost/internal/nocalhost-control-plane/resource"
	c "nocalhost/internal/nocalhost-control-plane/server"
	"os"
	"os/signal"
	"syscall"
)

var (
	port uint
)

func init() {
	flag.UintVar(&port, "port", 18000, "xDS management server port")
	flag.Parse()
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	processor := resource.NewProcessor(nil)
	processor.Start(ctx)
	srv := server.NewServer(ctx, processor.Snapshot, nil)
	go c.RunManagementServer(ctx, srv, port)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	select {
	case <-ctx.Done():
	case <-interrupt:
	}
	cancel()
}
