/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	pkgresource "k8s.io/cli-runtime/pkg/resource"
	runtimeresource "k8s.io/cli-runtime/pkg/resource"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"nocalhost/internal/nhctl/common/base"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/controller"
	"nocalhost/internal/nhctl/nocalhost"
	"strings"
)

type UnstructuredHandler struct {
	factory cmdutil.Factory
	config  *PodRouteConfig
	info    *runtimeresource.Info
}

func NewUnstructuredHandler(
	factory cmdutil.Factory,
	info *runtimeresource.Info,
	config *PodRouteConfig,
) *UnstructuredHandler {
	return &UnstructuredHandler{
		factory: factory,
		info:    info,
		config:  config,
	}
}

type P struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

// InjectVPNContainer
// todo if can't find any pod with same label, needs to create a pod or not ?
func (h *UnstructuredHandler) InjectVPNContainer() error {
	gvk := h.info.Mapping.Resource
	svcType := fmt.Sprintf("%s.%s.%s", gvk.Resource, gvk.Version, gvk.Group)
	// resources.version.group
	devModeAction, err := nocalhost.GetDevModeActionBySvcType(base.SvcType(svcType))
	if err != nil {
		return err
	}
	u := h.info.Object.(*unstructured.Unstructured)
	podTempSpec, err := controller.GetPodTemplateFromSpecPath(devModeAction.PodTemplatePath, u.Object)
	if err != nil {
		return err
	}
	helper := pkgresource.NewHelper(h.info.Client, h.info.Mapping)
	controller.RemoveUselessInfo(u)
	originalSpecJson, err := json.Marshal(u)
	if err != nil {
		return errors.WithStack(err)
	}
	pat, _ := json.Marshal([]P{{
		Op:    "replace",
		Path:  fmt.Sprintf("/metadata/annotations/%s", strings.ReplaceAll(_const.OriginWorkloadDefinition, "/", "~1")),
		Value: string(originalSpecJson),
	}})
	// 1, backup origin resource definition
	_, err = helper.Patch(h.info.Namespace, h.info.Name, types.JSONPatchType, pat, &metav1.PatchOptions{})
	if err != nil {
		return err
	}
	origin := *podTempSpec
	// 2, inject vpn sidecar
	AddContainer(&podTempSpec.Spec, h.config)
	pathItems := strings.Split(devModeAction.PodTemplatePath, "/")
	bytes, _ := json.Marshal([]P{{
		Op:    "replace",
		Path:  strings.Join(append(pathItems, "spec"), "/"),
		Value: podTempSpec.Spec,
	}})
	_, err = helper.Patch(h.info.Namespace, h.info.Name, types.JSONPatchType, bytes, &metav1.PatchOptions{})
	if err != nil {
		return errors.Wrap(err, "error while inject proxy container, exiting...")
	}
	// 3, remove readiness probe, liveness probe, startup probe
	removePatch, backupPatch := patch(origin, pathItems)
	_, err = helper.Patch(h.info.Namespace, h.info.Name, types.JSONPatchType, removePatch, &metav1.PatchOptions{})
	if err != nil {
		return err
	}
	// 4, backup origin probe definition
	p, _ := json.Marshal([]P{{
		Op:    "replace",
		Path:  fmt.Sprintf("/metadata/annotations/%s", strings.ReplaceAll(_const.OriginProbeDefinition, "/", "~1")),
		Value: string(backupPatch),
	}})
	_, err = helper.Patch(h.info.Namespace, h.info.Name, types.JSONPatchType, p, &metav1.PatchOptions{})
	return err
}

func (h *UnstructuredHandler) Rollback(reset bool) error {
	helper := pkgresource.NewHelper(h.info.Client, h.info.Mapping)
	u := h.info.Object.(*unstructured.Unstructured)
	if reset {
		s, ok := u.GetAnnotations()[_const.OriginWorkloadDefinition]
		if ok {
			return errors.New("can not found any origin workloads definition")
		}
		var uu unstructured.Unstructured
		if err := json.Unmarshal([]byte(s), &uu); err != nil {
			return err
		}
		controller.RemoveUselessInfo(&uu)
		_, err := helper.Replace(h.info.Namespace, h.info.Name, true, &uu)
		return err
	}

	gvk := h.info.Mapping.Resource
	svcType := fmt.Sprintf("%s.%s.%s", gvk.Resource, gvk.Version, gvk.Group)
	bySvcType, err := nocalhost.GetDevModeActionBySvcType(base.SvcType(svcType))
	if err != nil {
		return err
	}
	podTempSpec, err := controller.GetPodTemplateFromSpecPath(bySvcType.PodTemplatePath, u.Object)
	if err != nil {
		return err
	}
	pathItems := strings.Split(bySvcType.PodTemplatePath, "/")
	// 1, remove vpn sidecar
	RemoveContainer(&podTempSpec.Spec)
	bytes, _ := json.Marshal([]P{{
		Op:    "replace",
		Path:  strings.Join(append(pathItems, "spec"), "/"),
		Value: podTempSpec.Spec,
	}})
	_, err = helper.Patch(h.info.Namespace, h.info.Name, types.JSONPatchType, bytes, &metav1.PatchOptions{})
	if err != nil {
		return err
	}
	// 2, rollback liveness probe, startup probe, readiness probe
	s, ok := u.GetAnnotations()[_const.OriginProbeDefinition]
	if !ok {
		return errors.New("can not find origin probe definition")
	}
	_, err = helper.Patch(h.info.Namespace, h.info.Name, types.JSONPatchType, []byte(s), &metav1.PatchOptions{})
	if err != nil {
		return errors.Wrapf(err, "error while restore probe of resource: %s %s, ignore",
			h.info.Mapping.GroupVersionKind.GroupKind().String(), h.info.Name)
	}
	return nil
}

func (h *UnstructuredHandler) GetPod() ([]corev1.Pod, error) {
	gvk := h.info.Mapping.Resource
	svcType := fmt.Sprintf("%s.%s.%s", gvk.Resource, gvk.Version, gvk.Group)
	// resources.version.group
	devModeAction, err := nocalhost.GetDevModeActionBySvcType(base.SvcType(svcType))
	if err != nil {
		return nil, err
	}
	u := h.info.Object.(*unstructured.Unstructured)
	podTempSpec, err := controller.GetPodTemplateFromSpecPath(devModeAction.PodTemplatePath, u.Object)
	if err != nil {
		return nil, err
	}
	set, err := h.factory.KubernetesClientSet()
	if err != nil {
		return nil, err
	}
	list, err := set.CoreV1().Pods(h.info.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(podTempSpec.GetLabels()).String(),
	})
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(list.Items); i++ {
		if list.Items[i].DeletionTimestamp != nil {
			list.Items = append(list.Items[:i], list.Items[i+1:]...)
			i--
		}
	}
	if len(list.Items) == 0 {
		return nil, errors.New("can not find pod")
	}
	return list.Items, nil
}
