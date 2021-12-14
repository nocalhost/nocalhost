package cron

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

// Graceful shutdown
type Graceful struct {
	queue []func(os.Signal)
}

func (g *Graceful) AddFunc(fn func(os.Signal)) {
	g.queue = append(g.queue, fn)
}

func (g *Graceful) Wait(hint string) {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	s := <-ch
	log.Printf(hint)

	for _, fn := range g.queue {
		fn(s)
	}
}
