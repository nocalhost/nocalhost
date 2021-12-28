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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"nocalhost/internal/nhctl/vpn/util"
	"strings"
)

type ServiceHandler struct {
	factory   cmdutil.Factory
	clientset *kubernetes.Clientset
	namespace string
	name      string
}

func NewServiceHandler(factory cmdutil.Factory, clientset *kubernetes.Clientset, namespace, name string) *ServiceHandler {
	return &ServiceHandler{
		factory:   factory,
		clientset: clientset,
		namespace: namespace,
		name:      name,
	}
}

// ScaleToZero get deployment, statefulset, replicaset, otherwise delete it
func (s *ServiceHandler) ScaleToZero() (map[string]string, []v1.ContainerPort, string, error) {
	service, err := s.clientset.CoreV1().Services(s.namespace).Get(context.TODO(), s.name, metav1.GetOptions{})
	if err != nil {
		return nil, nil, "", err
	}

	var va []util.ResourceTupleWithScale
	list, err := util.GetAndConsumeControllerObject(s.factory, s.namespace, labels.SelectorFromSet(service.Spec.Selector), func(u *unstructured.Unstructured) {
		replicas, _, err := unstructured.NestedInt64(u.Object, "spec", "replicas")
		if err == nil {
			va = append(va, util.ResourceTupleWithScale{
				Resource: strings.ToLower(u.GetKind()) + "s",
				Name:     u.GetName(),
				Scale:    int(replicas),
			})
			err = util.UpdateReplicasScale(s.clientset, s.namespace, util.ResourceTupleWithScale{
				Resource: strings.ToLower(u.GetKind()) + "s",
				Name:     u.GetName(),
				Scale:    0,
			})
			if err != nil {
			}
		}
	})
	if err != nil {
		return nil, nil, "", err
	}
	if len(list) == 0 {
		return nil, nil, "", nil
	}
	if len(va) != 0 {
		var ports []v1.ContainerPort
		for _, port := range service.Spec.Ports {
			ports = append(ports, v1.ContainerPort{
				Name:          port.Name,
				ContainerPort: port.Port,
				Protocol:      port.Protocol,
			})
		}
		marshal, _ := json.Marshal(va)
		return service.Spec.Selector, ports, string(marshal), nil
	} else {
		// CRD
		var result []string
		for _, info := range list {
			u := info.Object.(*unstructured.Unstructured)
			u.SetManagedFields(nil)
			u.SetResourceVersion("")
			u.SetUID("")
			bytes, _ := u.MarshalJSON()
			result = append(result, string(bytes))
			helper := resource.NewHelper(info.Client, info.Mapping)
			if _, err = helper.Delete(s.namespace, info.Name); err != nil {
				continue
			}
		}
		marshal, _ := json.Marshal(result)
		return util.GetLabelSelector(list[0].Object).MatchLabels, util.GetPorts(list[0].Object), string(marshal), err
	}
}

func (s ServiceHandler) getResource() string {
	return "services"
}

func (s ServiceHandler) ToInboundPodName() string {
	return fmt.Sprintf("%s-%s-shadow", s.getResource(), s.name)
}

func (s *ServiceHandler) Reset() error {
	get, err := s.clientset.CoreV1().
		Pods(s.namespace).
		Get(context.TODO(), s.ToInboundPodName(), metav1.GetOptions{})
	if err != nil {
		return err
	}
	return restore(s.factory, s.clientset, s.namespace, get.GetAnnotations()[util.OriginData])
}

func restore(factory cmdutil.Factory, clientset *kubernetes.Clientset, namespace string, data string) error {
	if o := data; len(o) != 0 {
		var resourceTupleList util.ResourceTupleWithScale
		if err := json.Unmarshal([]byte(o), &resourceTupleList); err == nil {
			_ = util.UpdateReplicasScale(clientset, namespace, resourceTupleList)
		}
		var u unstructured.Unstructured
		if err := json.Unmarshal([]byte(o), &u); err == nil {
			if client, err := factory.DynamicClient(); err == nil {
				gvrResource := client.Resource(schema.GroupVersionResource{
					Group:    u.GetObjectKind().GroupVersionKind().Group,
					Version:  u.GetObjectKind().GroupVersionKind().Version,
					Resource: strings.ToLower(u.GetObjectKind().GroupVersionKind().Kind) + "s",
				})
				gvrResource.Namespace(namespace).Create(context.TODO(), &u, metav1.CreateOptions{})
			}
		}
	}
	return nil
}
