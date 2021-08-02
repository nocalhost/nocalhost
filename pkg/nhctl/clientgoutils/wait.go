/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package clientgoutils

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/rest"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

func (c *ClientGoUtils) WaitForResourceReady(
	resourceType ResourceType, name string, isReady func(object runtime.Object) (bool, error),
) error {
	var runtimeObject runtime.Object
	var restClient rest.Interface
	switch resourceType {
	case DeploymentType:
		runtimeObject = &v1.Deployment{}
		restClient = c.ClientSet.AppsV1().RESTClient()
	case JobType:
		runtimeObject = &batchv1.Job{}
		restClient = c.ClientSet.BatchV1().RESTClient()
	case StatefulSetType:
		runtimeObject = &v1.StatefulSet{}
		restClient = c.ClientSet.AppsV1().RESTClient()
	default:
		return errors.New("can not watch resource type " + string(resourceType))
	}

	f, err := fields.ParseSelector(fmt.Sprintf("metadata.name=%s", name))
	if err != nil {
		return errors.Wrap(err, "")
	}
	watchlist := cache.NewListWatchFromClient(
		restClient,
		string(resourceType),
		c.namespace,
		f, //fields.Everything()
	)

	stop := make(chan struct{})
	defer close(stop)
	exit := make(chan bool)
	_, controller := cache.NewInformer(
		// also take a look at NewSharedIndexInformer
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
		case <-c.ctx.Done():
			return err
		case <-exit:
			return err
		default:
			time.Sleep(time.Second * 2)
		}
	}
	return err
}

func (c *ClientGoUtils) WaitDeploymentToBeReady(name string) error {
	return c.WaitForResourceReady(DeploymentType, name, isDeploymentReady)
}

func (c *ClientGoUtils) WaitStatefulSetToBeReady(name string) error {
	return c.WaitForResourceReady(StatefulSetType, name, isStatefulSetReady)
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

func isStatefulSetReady(obj runtime.Object) (bool, error) {
	o, ok := obj.(*v1.StatefulSet)
	if !ok {
		return true, errors.Errorf("expected a *StatefulSet, got %T", obj)
	}

	//for _, c := range o.Status.Conditions {
	//	if c.Status == "True" {
	//		return true, nil
	//	}
	//}
	if o.Status.ReadyReplicas == 1 {
		return true, nil
	}
	return false, nil
}

func (c *ClientGoUtils) WaitJobToBeReady(name, format string) error {
	// metadata.name
	f, err := fields.ParseSelector(fmt.Sprintf("%s=%s", format, name))
	if err != nil {
		return errors.Wrap(err, "")
	}
	watchlist := cache.NewListWatchFromClient(
		c.ClientSet.BatchV1().RESTClient(),
		"jobs",
		c.namespace,
		f, //fields.Everything()
	)
	stop := make(chan struct{})
	exit := make(chan int)
	_, controller := cache.NewInformer(
		// also take a look at NewSharedIndexInformer
		watchlist,
		&batchv1.Job{},
		0, //Duration is int64
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
			},
			DeleteFunc: func(obj interface{}) {
				fmt.Printf("Job %s deleted\n", name)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
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
