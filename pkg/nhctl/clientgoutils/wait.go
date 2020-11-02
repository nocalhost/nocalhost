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

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

// namespace : use "" to watch all namespaces
func (c *ClientGoUtils) WaitJobToBeReady(namespace string, name string) error {

	f, err := fields.ParseSelector(fmt.Sprintf("metadata.name=%s", name))
	if err != nil {
		return err
	}
	watchlist := cache.NewListWatchFromClient(
		c.ClientSet.BatchV1().RESTClient(),
		"jobs",
		namespace,
		//fields.Everything(),
		f,
	)
	stop := make(chan struct{})
	exit := make(chan int)
	_, controller := cache.NewInformer( // also take a look at NewSharedIndexInformer
		watchlist,
		&batchv1.Job{},
		0, //Duration is int64
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				//fmt.Printf("job added: %s \n", obj.(runtime.Object))
			},
			DeleteFunc: func(obj interface{}) {
				fmt.Printf("job deleted: %s \n", name)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				fmt.Printf("job %s changed : ", name)
				if completed, _ := waitForJob(newObj.(runtime.Object), name); completed {
					close(stop)
					exit <- 1
				}
			},
		},
	)
	//defer close(stop)
	go controller.Run(stop)

	select {
	case <-exit:
		break
	}
	return nil
}
