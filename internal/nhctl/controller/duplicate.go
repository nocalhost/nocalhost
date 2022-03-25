/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/tidwall/sjson"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/model"
	"nocalhost/pkg/nhctl/log"
	"strings"
)

const (
	IdentifierKey         = "identifier"
	OriginWorkloadNameKey = "origin-workload-name"
	OriginWorkloadTypeKey = "origin-workload-type"
)

// ReplaceDuplicateModeImage Create a duplicate deployment instead of replacing image
func (c *Controller) ReplaceDuplicateModeImage(ctx context.Context, ops *model.DevStartOptions) error {
	c.Client.Context(ctx)

	um, err := c.GetUnstructured()
	if err != nil {
		return err
	}

	if c.IsInReplaceDevMode() {
		od, err := GetAnnotationFromUnstructured(um, _const.OriginWorkloadDefinition)
		if err != nil {
			return err
		}

		if um, err = c.Client.GetUnstructuredFromString(od); err != nil {
			return err
		}
	}

	RemoveUselessInfo(um)

	var podTemplate *v1.PodTemplateSpec
	if !c.DevModeAction.Create {

		um.SetName(c.getDuplicateResourceName())
		um.SetLabels(c.getDuplicateLabelsMap())
		um.SetResourceVersion("")

		if podTemplate, err = GetPodTemplateFromSpecPath(c.DevModeAction.PodTemplatePath, um.Object); err != nil {
			return err
		}

		podTemplate.Labels = c.getDuplicateLabelsMap()
		podTemplate.Annotations = c.getDevContainerAnnotations(ops.Container, podTemplate.Annotations)

		devContainer, sideCarContainer, devModeVolumes, err :=
			c.genContainersAndVolumes(&podTemplate.Spec, ops.Container, ops.DevImage, ops.StorageClass, true)
		if err != nil {
			return err
		}

		patchDevContainerToPodSpec(&podTemplate.Spec, ops.Container, devContainer, sideCarContainer, devModeVolumes)

		jsonObj, err := um.MarshalJSON()
		if err != nil {
			return errors.WithStack(err)
		}

		lm := map[string]interface{}{"matchLabels": c.getDuplicateLabelsMap()}
		lmJson, _ := json.Marshal(lm)

		var jsonStr string
		pathItems := strings.Split(c.DevModeAction.PodTemplatePath, "/")
		pis := make([]string, 0)
		for _, item := range pathItems {
			if item != "" {
				pis = append(pis, item)
			}
		}
		path := strings.Join(pis[:len(pis)-1], ".")
		selectorPath := path + "." + "selector" // /spec/selector
		if jsonStr, err = sjson.SetRaw(string(jsonObj), selectorPath, string(lmJson)); err != nil {
			return errors.WithStack(err)
		}

		if jsonStr, err = sjson.Delete(jsonStr, "status"); err != nil {
			return errors.WithStack(err)
		}

		pss, _ := json.Marshal(podTemplate)
		templatePath := strings.Join(pis, ".")
		if jsonStr, err = sjson.SetRaw(jsonStr, templatePath, string(pss)); err != nil {
			return errors.WithStack(err)
		}

		infos, err := c.Client.GetResourceInfoFromString(jsonStr, true)
		if err != nil {
			return err
		}

		if len(infos) != 1 {
			return errors.New(fmt.Sprintf("ResourceInfo' num is %d(not 1?)", len(infos)))
		}

		log.Infof("Creating %s(%v)", infos[0].Name, infos[0].Object.GetObjectKind().GroupVersionKind())
		err = c.Client.ApplyResourceInfo(infos[0], nil)
		if err != nil {
			return err
		}

		gvk := infos[0].Object.GetObjectKind().GroupVersionKind()
		kind := gvk.Kind
		if gvk.Version != "" {
			kind += "." + gvk.Version
		}
		if gvk.Group != "" {
			kind += "." + gvk.Group
		}

		for _, item := range c.DevModeAction.ScalePatches {
			log.Infof("Patching %s", item.Patch)
			if err = c.Client.Patch(kind, infos[0].Name, item.Patch, item.Type); err != nil {
				return err
			}
		}

		c.patchAfterDevContainerReplaced(ops.Container, kind, infos[0].Name)
	} else {
		labelsMap := c.getDuplicateLabelsMap()

		if podTemplate, err = GetPodTemplateFromSpecPath(c.DevModeAction.PodTemplatePath, um.Object); err != nil {
			return err
		}
		podTemplate.Annotations = c.getDevContainerAnnotations(ops.Container, podTemplate.Annotations)
		genDeploy := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:        c.getDuplicateResourceName(),
				Labels:      labelsMap,
			},
			Spec: appsv1.DeploymentSpec{
				Template: *podTemplate,
			},
		}
		genDeploy.Spec.Selector = &metav1.LabelSelector{MatchLabels: labelsMap}
		genDeploy.Spec.Template.Labels = labelsMap
		genDeploy.ResourceVersion = ""
		genDeploy.Spec.Template.Spec.NodeName = ""

		devContainer, sideCarContainer, devModeVolumes, err :=
			c.genContainersAndVolumes(
				&genDeploy.Spec.Template.Spec, ops.Container, ops.DevImage, ops.StorageClass, true,
			)
		if err != nil {
			return err
		}

		patchDevContainerToPodSpec(
			&genDeploy.Spec.Template.Spec, ops.Container, devContainer, sideCarContainer, devModeVolumes,
		)

		genDeploy.Spec.Template.Spec.RestartPolicy = v1.RestartPolicyAlways

		podTemplate = &genDeploy.Spec.Template

		// Create generated deployment
		if _, err = c.Client.CreateDeploymentAndWait(genDeploy); err != nil {
			return err
		}

		c.patchAfterDevContainerReplaced(ops.Container, genDeploy.Kind, genDeploy.Name)
	}

	delete(podTemplate.Labels, "pod-template-hash")
	c.devModePodLabels = podTemplate.Labels

	c.waitDevPodToBeReady()
	return nil
}
