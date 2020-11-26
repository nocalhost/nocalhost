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
	"context"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/rest"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

func (c *ClientGoUtils) WaitForResourceReady(ctx context.Context, resourceType ResourceType, namespace string, name string, isReady func(object runtime.Object) (bool, error)) error {
	var runtimeObject runtime.Object
	var restClient rest.Interface
	switch resourceType {
	case DeploymentType:
		runtimeObject = &v1.Deployment{}
		restClient = c.ClientSet.AppsV1().RESTClient()
	case JobType:
		runtimeObject = &batchv1.Job{}
		restClient = c.ClientSet.BatchV1().RESTClient()
	default:
		return errors.New("can not watch resource type " + string(resourceType))
	}

	f, err := fields.ParseSelector(fmt.Sprintf("metadata.name=%s", name))
	if err != nil {
		return err
	}
	watchlist := cache.NewListWatchFromClient(
		restClient,
		string(resourceType),
		namespace,
		f, //fields.Everything()
	)

	stop := make(chan struct{})
	defer close(stop)
	exit := make(chan bool)
	_, controller := cache.NewInformer( // also take a look at NewSharedIndexInformer
		watchlist,
		runtimeObject,
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
			},
			DeleteFunc: func(obj interface{}) {
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				obj, ok := newObj.(runtime.Object)
				if !ok {
					err = errors.New("can not get a runtime object")
					exit <- true
					return
				}
				b, err2 := isReady(obj)
				if err2 != nil || b {
					err = err2
					exit <- true
					return
				}
			},
		},
	)
	go controller.Run(stop)

	for {
		select {
		case <-ctx.Done():
			return err
		case <-exit:
			return err
		default:
			time.Sleep(time.Second * 2)
		}
	}
	return err
}

func (c *ClientGoUtils) WaitDeploymentToBeReady(namespace string, name string, timeout time.Duration) error {
	ctx, _ := context.WithTimeout(context.TODO(), timeout)
	return c.WaitForResourceReady(ctx, DeploymentType, namespace, name, isDeploymentReady)
}

func isDeploymentReady(obj runtime.Object) (bool, error) {
	o, ok := obj.(*v1.Deployment)
	if !ok {
		return true, errors.Errorf("expected a *apps.Deployment, got %T", obj)
	}

	for _, c := range o.Status.Conditions {
		if c.Type == v1.DeploymentAvailable && c.Status == "True" {
			log.Debug("Deployment is Available")
			return true, nil
		}
	}
	log.Debug("Deployment has not been ready yet")
	return false, nil
}

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
		f, //fields.Everything()
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
