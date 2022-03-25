/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package handler

import (
	"context"
	"encoding/json"
	"errors"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/resource"
	coreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/util/retry"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	_const "nocalhost/internal/nhctl/const"
	"time"
)

type PodHandler struct {
	factory      cmdutil.Factory
	podInterface coreV1.PodInterface
	info         *resource.Info
	config       *PodRouteConfig
}

func (h *PodHandler) GetPod() ([]v1.Pod, error) {
	pod, err := h.podInterface.Get(context.TODO(), h.info.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return []v1.Pod{*pod}, nil
}

func NewPodHandler(factory cmdutil.Factory, podInterface coreV1.PodInterface, info *resource.Info, config *PodRouteConfig) *PodHandler {
	return &PodHandler{
		factory:      factory,
		podInterface: podInterface,
		info:         info,
		config:       config,
	}
}

func (h *PodHandler) InjectVPNContainer() error {
	pod, err := h.podInterface.Get(context.TODO(), h.info.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	var newPod = &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        h.info.Name,
			Namespace:   h.info.Namespace,
			Labels:      pod.GetLabels(),
			Annotations: pod.GetAnnotations(),
		},
	}
	AddContainer(&newPod.Spec, h.config)
	annotations := newPod.GetAnnotations()
	pod.SetManagedFields(nil)
	pod.SetResourceVersion("")
	pod.SetUID("")
	pod.SetDeletionTimestamp(nil)
	marshal, err := json.Marshal(pod)
	annotations[_const.OriginWorkloadDefinition] = string(marshal)
	newPod.SetAnnotations(annotations)
	return createAfterDeletePod(h.podInterface, newPod)
}

func (h *PodHandler) Rollback(reset bool) error {
	get, err := h.podInterface.Get(context.TODO(), h.info.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	s, ok := get.GetAnnotations()[_const.OriginWorkloadDefinition]
	if !ok {
		return errors.New("can not found any origin resource definition")
	}
	var p v1.Pod
	err = json.Unmarshal([]byte(s), &p)
	if err != nil {
		return err
	}
	return createAfterDeletePod(h.podInterface, &p)
}

func createAfterDeletePod(podInterface coreV1.PodInterface, p *v1.Pod) error {
	zero := int64(0)
	err := podInterface.Delete(context.TODO(), p.Name, metav1.DeleteOptions{GracePeriodSeconds: &zero})
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	if err = retry.OnError(wait.Backoff{
		Steps:    10,
		Duration: 50 * time.Millisecond,
		Factor:   5.0,
		Jitter:   1,
	}, func(err error) bool {
		if !k8serrors.IsAlreadyExists(err) {
			return true
		}
		get, err := podInterface.Get(context.TODO(), p.Name, metav1.GetOptions{})
		if err != nil || get.Status.Phase != v1.PodRunning {
			return true
		}
		return false
	}, func() error {
		if _, err = podInterface.Create(context.TODO(), p, metav1.CreateOptions{}); err != nil {
			return err
		}
		return errors.New("")
	}); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}
	return nil
}
