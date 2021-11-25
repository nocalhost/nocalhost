package tools

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

// Graceful shutdown
type Graceful struct {
	ch    chan os.Signal
	queue []func(os.Signal)
}

func (g *Graceful) AddFunc(fn func(os.Signal)) {
	g.queue = append(g.queue, fn)
}

func (g *Graceful) Wait() {
	g.ch = make(chan os.Signal)
	signal.Notify(g.ch, syscall.SIGINT, syscall.SIGTERM)

	s := <- g.ch
	log.Printf("Received os signal: %s", s)

	for _, fn := range g.queue {
		fn(s)
	}
}
