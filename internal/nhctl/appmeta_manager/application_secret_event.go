/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package appmeta_manager

import (
	"encoding/json"
	"fmt"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/pkg/nhctl/log"
	"runtime/debug"
	"sync"
)

var (
	eventListener = []func(*ApplicationEventPack) error{
		func(pack *ApplicationEventPack) error {
			if pack.Event.EventType == appmeta.DEV_STA {
				log.Infof(
					"Resource(%s - %s - %s), Name(%s) Start DevMode ",
					pack.Ns, pack.AppName, pack.Event.DevType,
					pack.Event.ResourceName,
				)
			} else {
				log.Infof(
					"Resource(%s - %s - %s), Name(%s) End DevMode ",
					pack.Ns, pack.AppName, pack.Event.DevType,
					pack.Event.ResourceName,
				)
			}
			return nil
		},
	}

	Events  []*ApplicationEventPack
	lock    = sync.NewCond(&sync.Mutex{})
	isInit  bool
	startCh = make(chan struct{}, 1)
)

type ApplicationEventPack struct {
	Event           *appmeta.ApplicationEvent
	Ns, AppName     string
	KubeConfigBytes []byte
}

func RegisterListener(fun func(*ApplicationEventPack) error) {
	lock.L.Lock()
	defer lock.L.Unlock()
	eventListener = append(eventListener, fun)
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
	lock.L.Lock()
	defer lock.L.Unlock()
	if len(Events) == 0 {
		defer lock.Broadcast()
	}
	Events = append(Events, a)
}

func EventPop() *ApplicationEventPack {
	lock.L.Lock()
	defer lock.L.Unlock()
	if len(Events) == 0 {
		lock.Wait()
	}

	pop := Events[0]
	Events = Events[1:]
	return pop
}

func Init() {

	if isInit {
		return
	}
	lock.L.Lock()
	if isInit {
		return
	}
	isInit = true
	lock.L.Unlock()

	log.Info("Application Event Listener Start Up...")
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Fatalf("DAEMON-RECOVER: %s", string(debug.Stack()))
			}
		}()

		select {
		case <-startCh:
			for {
				pop := EventPop()
				for _, el := range eventListener {
					pop.consume(el, 5)
				}
			}
		}
	}()
}

func Start() {
	select {
	case _, ok := <-startCh:
		if ok {
			return
		}
	default:
		close(startCh)
	}
}
