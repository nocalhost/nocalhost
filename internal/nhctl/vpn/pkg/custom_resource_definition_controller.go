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

type CustomResourceDefinitionController struct {
	factory   cmdutil.Factory
	clientset *kubernetes.Clientset
	namespace string
	resource  string
	name      string
}

func NewCustomResourceDefinitionController(factory cmdutil.Factory, clientset *kubernetes.Clientset, namespace, resource, name string) *CustomResourceDefinitionController {
	return &CustomResourceDefinitionController{
		factory:   factory,
		clientset: clientset,
		namespace: namespace,
		resource:  resource,
		name:      name,
	}
}

// ScaleToZero TODO needs to create a same pod name, but with different labels for using to click
func (crd *CustomResourceDefinitionController) ScaleToZero() (map[string]string, []v1.ContainerPort, string, error) {
	//topController := util.GetTopController(crd.factory, crd.clientset, crd.namespace, fmt.Sprintf("%s/%s", crd.getResource(), crd.name))
	info, err := util.GetUnstructuredObject(crd.factory, crd.namespace, fmt.Sprintf("%s/%s", crd.getResource(), crd.name))
	if err != nil {
		return nil, nil, "", err
	}
	helper := resource.NewHelper(info.Client, info.Mapping)
	if _, err = helper.Delete(crd.namespace, info.Name); err != nil {
		return nil, nil, "", err
	}
	u := info.Object.(*unstructured.Unstructured)
	u.SetManagedFields(nil)
	u.SetResourceVersion("")
	u.SetUID("")
	bytes, _ := u.MarshalJSON()
	return util.GetLabelSelector(info.Object).MatchLabels, util.GetPorts(info.Object), string(bytes), err
}

func (crd *CustomResourceDefinitionController) Cancel() error {
	return crd.Reset()
}

func (crd CustomResourceDefinitionController) getResource() string {
	return crd.resource
}

func (crd *CustomResourceDefinitionController) Reset() error {
	get, err := crd.clientset.CoreV1().
		Pods(crd.namespace).
		Get(context.TODO(), ToInboundPodName(crd.getResource(), crd.name), metav1.GetOptions{})
	if err != nil {
		return err
	}
	if a := get.GetAnnotations()[util.OriginData]; len(a) != 0 {
		var r unstructured.Unstructured
		if err = json.Unmarshal([]byte(a), &r); err == nil {
			if client, err := crd.factory.DynamicClient(); err == nil {
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
