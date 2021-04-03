package appmeta_manager

import (
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/pkg/nhctl/log"
	"sync"
)

var (
	EventListener = []func(*ApplicationEventPack){
		func(pack *ApplicationEventPack) {
			if pack.Event.EventType == appmeta.DEV_STA {
				log.Infof("Resource(%s - %s - %s), Name(%s) Start DevMode ", pack.Ns, pack.AppName, pack.Event.DevType, pack.Event.ResourceName)
			} else {
				log.Infof("Resource(%s - %s - %s), Name(%s) End DevMode ", pack.Ns, pack.AppName, pack.Event.DevType, pack.Event.ResourceName)
			}
		},
	}

	Events []*ApplicationEventPack
	lock   = sync.NewCond(&sync.Mutex{})
)

type ApplicationEventPack struct {
	Event                   *appmeta.ApplicationEvent
	Ns, AppName, KubeConfig string
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
				el(pop)
			}
		}
	}()

	return true
}
