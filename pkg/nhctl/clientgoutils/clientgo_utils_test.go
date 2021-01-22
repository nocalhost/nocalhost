/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clientgoutils

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"nocalhost/pkg/nhctl/log"
	"testing"
	"time"
)

func TestNewClientGoUtils(t *testing.T) {
	namespace := "nh6ihig"
	client, _ := NewClientGoUtils("/Users/xinxinhuang/Library/Application Support/Lens/kubeconfigs/c3bbeccc-b61a-411a-af39-3d07bfe91017", namespace)
	//client.WaitJobToBeReady()

	f, err := fields.ParseSelector(fmt.Sprintf("involvedObject.kind=%s,involvedObject.name=%s", "ReplicaSet", "details-59c787d477"))
	//f, err := fields.ParseSelector(fmt.Sprintf("involvedObject.kind=%s", "ReplicaSet"))
	if err != nil {
		panic(err)
	}
	//f := fields.Everything()

	watchlist := cache.NewListWatchFromClient(
		client.ClientSet.CoreV1().RESTClient(),
		"events",
		namespace,
		f,
	)
	stop := make(chan struct{})
	exit := make(chan int)
	_, controller := cache.NewInformer( // also take a look at NewSharedIndexInformer
		watchlist,
		&corev1.Event{},
		0, //Duration is int64
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				event := obj.(runtime.Object).(*corev1.Event)
				fmt.Printf("Add %s %s %f\n", event.Name, event.Message, time.Now().Sub(event.LastTimestamp.Time).Seconds())
				//if event.Type == "Warning" && time.Now().Sub(event.LastTimestamp.Time).Seconds() < 10 {
				//	//if event.Reason == "FailedCreate" {
				//	//	//return errors.New(fmt.Sprintf("Latest ReplicaSet failed to be ready : %s", event.Message))
				//	//	log.Warnf("Latest ReplicaSet failed to be ready : %s", event.Message)
				//	//	//exit <- 1
				//	//}
				//	log.Warnf("Warning event: %s", event.Message)
				//}
			},
			DeleteFunc: func(obj interface{}) {
				event := obj.(runtime.Object).(*corev1.Event)
				fmt.Printf("Delete %s %s\n", event.Name, event.Message)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				oldEvent := oldObj.(runtime.Object).(*corev1.Event)
				event := newObj.(runtime.Object).(*corev1.Event)
				fmt.Printf("Update old %s: %s \n", oldEvent.Name, oldEvent.Message)
				fmt.Printf("Update new %s: %s \n", event.Name, event.Message)
				if event.Type == "Warning" {
					//if event.Reason == "FailedCreate" {
					//	//return errors.New(fmt.Sprintf("Latest ReplicaSet failed to be ready : %s", event.Message))
					//	log.Warnf("Latest ReplicaSet failed to be ready : %s", event.Message)
					//	exit <- 1
					//}
					log.Warnf("Warning event: %s", event.Message)
				}
			},
		},
	)
	//defer close(stop)
	go controller.Run(stop)

	select {
	case <-exit:
		fmt.Println("Closing...")
		close(stop)
	}

}

func TestClientGoUtils_Create(t *testing.T) {
	client, err := NewClientGoUtils("", "nh6ihig")
	if err != nil {
		panic(err)
	}
	secret, err := client.GetSecret("aaa")
	if err != nil {
		fmt.Printf("err:%s", err.Error())
		fmt.Printf("%v", secret)
		fmt.Println(secret.Name)
	} else {
		fmt.Printf("%v\n", secret)
	}
}
