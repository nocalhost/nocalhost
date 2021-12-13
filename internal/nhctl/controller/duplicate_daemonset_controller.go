/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/model"
	"nocalhost/pkg/nhctl/log"
)

type DuplicateDaemonSetController struct {
	*Controller
}

func (d *DuplicateDaemonSetController) GetNocalhostDevContainerPod() (string, error) {
	checkPodsList, err := d.GetPodList()
	if err != nil {
		return "", err
	}
	return findDevPod(checkPodsList)
}

// ReplaceImage For DaemonSet, we don't replace the DaemonSet' image
// but create a deployment with dev container instead
func (d *DuplicateDaemonSetController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {

	d.Client.Context(ctx)
	ds, err := d.Client.GetDaemonSet(d.GetName())
	if err != nil {
		return err
	}

	labelsMap, err := d.getDuplicateLabelsMap()
	if err != nil {
		return err
	}

	// Create a deployment from DaemonSet spec
	generatedDeployment := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   d.getDuplicateResourceName(),
			Labels: labelsMap,
		},
		Spec: v1.DeploymentSpec{
			Template: ds.Spec.Template,
		},
	}
	generatedDeployment.Spec.Selector = &metav1.LabelSelector{MatchLabels: labelsMap}
	generatedDeployment.Spec.Template.Labels = labelsMap
	generatedDeployment.ResourceVersion = ""
	generatedDeployment.Spec.Template.Spec.NodeName = ""

	devContainer, sideCarContainer, devModeVolumes, err :=
		d.genContainersAndVolumes(&generatedDeployment.Spec.Template.Spec, ops.Container, ops.DevImage, ops.StorageClass, true)
	if err != nil {
		return err
	}

	patchDevContainerToPodSpec(&generatedDeployment.Spec.Template.Spec, ops.Container, devContainer, sideCarContainer, devModeVolumes)

	// Create generated deployment
	if _, err = d.Client.CreateDeploymentAndWait(generatedDeployment); err != nil {
		return err
	}

	d.patchAfterDevContainerReplaced(ops.Container, generatedDeployment.Kind, generatedDeployment.Name)

	return waitingPodToBeReady(d.GetNocalhostDevContainerPod)
}

func (d *DuplicateDaemonSetController) RollBack(reset bool) error {
	lmap, err := d.getDuplicateLabelsMap()
	if err != nil {
		return err
	}

	ss, err := d.Client.Labels(lmap).ListDeployments()
	if err != nil {
		return err
	}

	if len(ss) != 1 {
		if !reset {
			return errors.New(fmt.Sprintf("Generated Deployment num is %d (not 1)?", len(ss)))
		} else if len(ss) == 0 {
			log.Warnf("Generated Deployment num is %d (not 1)?", len(ss))
			return nil
		}
	}

	return d.Client.DeleteDeployment(ss[0].Name, false)
}

// GetPodList
// In DevMode, return pod list of generated Deployment.
// Otherwise, return pod list of DaemonSet
func (d *DuplicateDaemonSetController) GetPodList() ([]corev1.Pod, error) {
	labelsMap, err := d.getDuplicateLabelsMap()
	if err != nil {
		return nil, err
	}
	return d.Client.Labels(labelsMap).ListPods()
}
