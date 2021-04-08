package appmeta_manager

import (
	"encoding/json"
	"errors"
	"fmt"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/pkg/nhctl/log"
	"sync"
)

var (
	EventListener = []func(*ApplicationEventPack) error{
		func(pack *ApplicationEventPack) error {
			if pack.Event.EventType == appmeta.DEV_STA {
				log.Infof("Resource(%s - %s - %s), Name(%s) Start DevMode ", pack.Ns, pack.AppName, pack.Event.DevType, pack.Event.ResourceName)
			} else {
				log.Infof("Resource(%s - %s - %s), Name(%s) End DevMode ", pack.Ns, pack.AppName, pack.Event.DevType, pack.Event.ResourceName)
			}
			return errors.New("")
		},
	}

	Events []*ApplicationEventPack
	lock   = sync.NewCond(&sync.Mutex{})
)

type ApplicationEventPack struct {
	Event                   *appmeta.ApplicationEvent
	Ns, AppName, KubeConfig string
}

func (pk *ApplicationEventPack) desc() string {
	marshal, _ := json.Marshal(pk.Event)
	return fmt.Sprintf("Ns '%s', App '%s' Event '%s'", pk.Ns, pk.AppName, marshal)
}

func (pk *ApplicationEventPack) consume(fun func(*ApplicationEventPack) error, retry int) {
	if err := fun(pk); err != nil && retry > 0 {
		log.Errorf("Error occur while consume %v, %s, retry until zero, %d...", pk.desc(), err.Error(), retry)
		pk.consume(fun, retry-1)
	}
	if r := recover(); r != nil && retry > 0 {
		log.Errorf("Panic occur while consume %v, %v, retry until zero, %d...", pk.desc(), r, retry)
		pk.consume(fun, retry-1)
	}
}

func EventPush(a *ApplicationEventPack) {
	log.Info("do push")
	lock.L.Lock()
	defer lock.L.Unlock()
	if len(Events) == 0 {
		defer lock.Broadcast()
	}
	Events = append(Events, a)
}

func EventPop() *ApplicationEventPack {
	log.Info("do pop")
	lock.L.Lock()
	defer lock.L.Unlock()
	if len(Events) == 0 {
		log.Info("lock wait")
		lock.Wait()
	}

	pop := Events[0]
	Events = Events[1:]
	return pop
}

func Run() bool {
	log.Info("Application Event Listener Start Up...")
	go func() {
		for {
			pop := EventPop()
			for _, el := range EventListener {
				pop.consume(el, 5)
			}
		}
	}()

	return true
}
