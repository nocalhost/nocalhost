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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"nocalhost/internal/nhctl/vpn/util"
	"strings"
)

type PodHandler struct {
	factory   cmdutil.Factory
	clientset *kubernetes.Clientset
	namespace string
	name      string
}

func NewPodHandler(factory cmdutil.Factory, clientset *kubernetes.Clientset, namespace, name string) *PodHandler {
	return &PodHandler{
		factory:   factory,
		clientset: clientset,
		namespace: namespace,
		name:      name,
	}
}

// ScaleToZero TODO needs to create a same pod name, but with different labels for using to click
func (pod *PodHandler) ScaleToZero() (map[string]string, []v1.ContainerPort, string, error) {
	topController := util.GetTopController(pod.factory, pod.clientset, pod.namespace, fmt.Sprintf("%s/%s", pod.getResource(), pod.name))
	zero := int64(0)
	// controllerBy is empty
	if len(topController.Name) == 0 || len(topController.Resource) == 0 {
		object, err := util.GetUnstructuredObject(pod.factory, pod.namespace, fmt.Sprintf("%s/%s", pod.getResource(), pod.name))
		if err != nil {
			return nil, nil, "", err
		}
		u := object.Object.(*unstructured.Unstructured)
		u.SetManagedFields(nil)
		u.SetUID("")
		u.SetResourceVersion("")
		marshal, _ := json.Marshal(u)
		_ = pod.clientset.CoreV1().Pods(pod.namespace).Delete(context.TODO(), pod.name, metav1.DeleteOptions{
			GracePeriodSeconds: &zero,
		})
		return u.GetLabels(), util.GetPorts(u), string(marshal), nil
	}
	object, err := util.GetUnstructuredObject(pod.factory, pod.namespace, fmt.Sprintf("%s/%s", topController.Resource, topController.Name))
	helper := resource.NewHelper(object.Client, object.Mapping)
	if _, err = helper.DeleteWithOptions(pod.namespace, object.Name, &metav1.DeleteOptions{
		GracePeriodSeconds: &zero,
	}); err != nil {
		return nil, nil, "", err
	}
	u := object.Object.(*unstructured.Unstructured)
	u.SetManagedFields(nil)
	u.SetUID("")
	u.SetResourceVersion("")
	bytes, _ := u.MarshalJSON()
	return util.GetLabelSelector(object.Object).MatchLabels, util.GetPorts(object.Object), string(bytes), err
}

func (pod PodHandler) getResource() string {
	return "pods"
}

func (pod PodHandler) ToInboundPodName() string {
	return pod.name
}

func (pod *PodHandler) Reset() error {
	get, err := pod.clientset.CoreV1().
		Pods(pod.namespace).
		Get(context.TODO(), pod.ToInboundPodName(), metav1.GetOptions{})
	if err != nil {
		return err
	}
	if a := get.GetAnnotations()[util.OriginData]; len(a) != 0 {
		var r unstructured.Unstructured
		if err = json.Unmarshal([]byte(a), &r); err == nil {
			if client, err := pod.factory.DynamicClient(); err == nil {
				gvr := schema.GroupVersionResource{
					Group:    r.GetObjectKind().GroupVersionKind().Group,
					Version:  r.GetObjectKind().GroupVersionKind().Version,
					Resource: strings.ToLower(r.GetObjectKind().GroupVersionKind().Kind) + "s",
				}
				_, _ = client.Resource(gvr).Create(context.TODO(), &r, metav1.CreateOptions{})
			}
		}
	}
	return nil
}
