package utils

import (
	"context"
	"sync"
	"time"
)

type Printer struct {
	ctx              context.Context
	statusNotifyChan chan string

	sameContentBackOffSecond int

	once    sync.Once
	printer func(string)
}

func NewPrinter(printerFunc func(string)) *Printer {
	pr := &Printer{
		context.TODO(),
		make(chan string, 10),
		60,
		sync.Once{},
		printerFunc,
	}

	pr.once.Do(
		func() {
			go func() {
				someContentRecorder := make(map[string]int64, 0)

				running := true
				for {
					select {
					case <-pr.ctx.Done():
						running = false

					case c := <-pr.statusNotifyChan:
						if nano, ok := someContentRecorder[c]; !ok ||
							nano < time.Now().UnixNano() {
							pr.printer(c)

							someContentRecorder[c] = time.
								Now().
								Add(time.Second * time.Duration(pr.sameContentBackOffSecond)).
								UnixNano()
						}
					}
					if !running {
						break
					}
				}
			}()
		},
	)

	return pr
}

func (se *Printer) ChangeContent(content string) {
	if se == nil {
		return
	}
	select {
	case se.statusNotifyChan <- content:
	default:
	}
}

func (se *Printer) Done() {
	se.ctx.Done()
}
