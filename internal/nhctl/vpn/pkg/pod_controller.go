/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"nocalhost/internal/nhctl/vpn/util"
)

type PodController struct {
	factory   cmdutil.Factory
	clientset *kubernetes.Clientset
	namespace string
	name      string
}

func NewPodController(factory cmdutil.Factory, clientset *kubernetes.Clientset, namespace, name string) *PodController {
	return &PodController{
		factory:   factory,
		clientset: clientset,
		namespace: namespace,
		name:      name,
	}
}

// ScaleToZero TODO needs to create a same pod name, but with different labels for using to click
func (pod *PodController) ScaleToZero() (map[string]string, []v1.ContainerPort, string, error) {
	topController := util.GetTopController(pod.factory, pod.clientset, pod.namespace, fmt.Sprintf("%s/%s", pod.getResource(), pod.name))
	// controllerBy is empty
	if len(topController.Name) == 0 || len(topController.Resource) == 0 {
		get, err := pod.clientset.CoreV1().Pods(pod.namespace).Get(context.TODO(), pod.name, metav1.GetOptions{})
		if err != nil {
			return nil, nil, "", err
		}
		marshal, _ := json.Marshal(get)
		_ = pod.clientset.CoreV1().Pods(pod.namespace).Delete(context.TODO(), pod.name, metav1.DeleteOptions{})
		return get.GetLabels(), get.Spec.Containers[0].Ports, string(marshal), nil
	}
	object, err := util.GetUnstructuredObject(pod.factory, pod.namespace, fmt.Sprintf("%s/%s", topController.Resource, topController.Name))
	helper := resource.NewHelper(object.Client, object.Mapping)
	//pod.f = func() error {
	//	_, err = helper.Create(pod.namespace, true, object.Object)
	//	return err
	//}
	if _, err = helper.Delete(pod.namespace, object.Name); err != nil {
		return nil, nil, "", err
	}
	marshal, _ := json.Marshal(object.Object)
	return util.GetLabelSelector(object.Object).MatchLabels, util.GetPorts(object.Object), string(marshal), err
}

func (pod *PodController) Cancel() error {
	return pod.Reset()
}

func (pod PodController) getResource() string {
	return "pods"
}

func (pod *PodController) Reset() error {
	get, err := pod.clientset.CoreV1().
		Pods(pod.namespace).
		Get(context.TODO(), ToInboundPodName(pod.getResource(), pod.name), metav1.GetOptions{})
	if err != nil {
		return err
	}
	if a := get.GetAnnotations()[util.OriginData]; len(a) != 0 {
		var r unstructured.Unstructured
		if err = json.Unmarshal([]byte(a), &r); err != nil {
			return err
		}
		r.SetResourceVersion("")
		client, err := pod.factory.DynamicClient()
		if err != nil {
			return err
		}
		mapper, err := pod.factory.ToRESTMapper()
		if err != nil {
			return err
		}
		mapping, err := mapper.RESTMapping(r.GetObjectKind().GroupVersionKind().GroupKind(), r.GetObjectKind().GroupVersionKind().Version)
		if err != nil {
			return err
		}
		_, err = client.Resource(mapping.Resource).Create(context.TODO(), &r, metav1.CreateOptions{})
		return err
	}
	return nil
}
