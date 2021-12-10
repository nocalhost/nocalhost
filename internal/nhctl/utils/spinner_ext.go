package utils

import (
	"context"
	"sync"
)

func NewSpinnerExt(printer func(string)) *spinnerExt {
	return &spinnerExt{
		context.TODO(),
		make(chan Msg, 10),
		sync.Once{},
		printer,
	}
}

type spinnerExt struct {
	ctx              context.Context
	statusNotifyChan chan Msg

	once    sync.Once
	printer func(string)
}

func (se *spinnerExt) Start() {
	se.once.Do(
		func() {
			go func() {

				cache := make(map[string]string, 0)
				running := true
				for {
					select {
					case <-se.ctx.Done():
						running = false
					case c := <-se.statusNotifyChan:
						if cache[c.Tag] != c.Content {
							se.printer(c.Content)
							cache[c.Tag] = c.Content
						}
					}
					if !running {
						break
					}
				}
			}()
		},
	)
}

func (se *spinnerExt) ChangeContent(tag, content string) {
	select {
	case se.statusNotifyChan <- Msg{tag, content}:
	default:
	}
}

func (se *spinnerExt) Done() {
	se.ctx.Done()
}

type Msg struct {
	Tag     string
	Content string
}
