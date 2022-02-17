/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package handler

import (
	"context"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/resource"
	coreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"nocalhost/internal/nhctl/vpn/util"
)

type ServiceHandler struct {
	factory          cmdutil.Factory
	serviceInterface coreV1.ServiceInterface
	config           *PodRouteConfig
	info             *resource.Info
}

func NewServiceHandler(factory cmdutil.Factory, serviceInterface coreV1.ServiceInterface, info *resource.Info, config *PodRouteConfig) *ServiceHandler {
	return &ServiceHandler{
		factory:          factory,
		serviceInterface: serviceInterface,
		info:             info,
		config:           config,
	}
}

// InjectVPNContainer
// try to find controllers using pod, using UnstructuredHandler to handler it,
// if can not find supported svcType, then delete it after backup
func (h *ServiceHandler) InjectVPNContainer() error {
	svc, err := h.serviceInterface.Get(context.TODO(), h.info.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	client, err := h.factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	controller, err := util.GetTopControllerBaseOnPodLabel(
		h.factory, client.CoreV1().Pods(h.info.Namespace), h.info.Namespace, labels.SelectorFromSet(svc.Spec.Selector),
	)
	if err != nil {
		return err
	}
	sc := NewUnstructuredHandler(h.factory, controller, h.config)
	return sc.InjectVPNContainer()
}

func (h *ServiceHandler) Rollback(reset bool) error {
	svc, err := h.serviceInterface.Get(context.TODO(), h.info.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	client, err := h.factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	controller, err := util.GetTopControllerBaseOnPodLabel(
		h.factory, client.CoreV1().Pods(h.info.Namespace), h.info.Namespace, labels.SelectorFromSet(svc.Spec.Selector),
	)
	if err != nil {
		return err
	}
	sc := NewUnstructuredHandler(h.factory, controller, h.config)
	return sc.Rollback(reset)
}

func (h *ServiceHandler) GetPod() ([]corev1.Pod, error) {
	svc, err := h.serviceInterface.Get(context.TODO(), h.info.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	client, err := h.factory.KubernetesClientSet()
	if err != nil {
		return nil, err
	}
	podList, _ := client.CoreV1().Pods(h.info.Namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(svc.Spec.Selector).String(),
	})
	for i := 0; i < len(podList.Items); i++ {
		if podList.Items[i].DeletionTimestamp != nil {
			podList.Items = append(podList.Items[:i], podList.Items[i+1:]...)
			i--
		}
	}
	if len(podList.Items) == 0 {
		return nil, errors.New("can not find any pod")
	}
	return podList.Items, nil
}
