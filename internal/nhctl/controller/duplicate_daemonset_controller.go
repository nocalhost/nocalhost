/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"context"
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/profile"
	"strings"
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

	p, err := d.GetAppProfile()
	if err != nil {
		return err
	}
	if p.Identifier == "" {
		return errors.New("Identifier can not be nil ")
	}

	suffix := p.Identifier[0:5]
	labelsMap, err := d.getDuplicateLabelsMap()
	if err != nil {
		return err
	}

	// Create a deployment from DaemonSet spec
	generatedDeployment := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   strings.Join([]string{d.Name, "daemonset", suffix}, "-"),
			Labels: labelsMap,
		},
		Spec: v1.DeploymentSpec{
			Template: ds.Spec.Template,
		},
	}
	generatedDeployment.Spec.Selector = &metav1.LabelSelector{MatchLabels: labelsMap}
	generatedDeployment.Spec.Template.Labels = labelsMap
	generatedDeployment.ResourceVersion = ""

	devContainer, err := findContainerInDeploySpec(generatedDeployment, ops.Container)
	if err != nil {
		return err
	}

	devContainer, sideCarContainer, devModeVolumes, err :=
		d.genContainersAndVolumes(devContainer, ops.Container, ops.DevImage, ops.StorageClass, true)
	if err != nil {
		return err
	}

	if ops.Container != "" {
		for index, c := range generatedDeployment.Spec.Template.Spec.Containers {
			if c.Name == ops.Container {
				generatedDeployment.Spec.Template.Spec.Containers[index] = *devContainer
				break
			}
		}
	} else {
		generatedDeployment.Spec.Template.Spec.Containers[0] = *devContainer
	}

	// Add volumes to deployment spec
	if generatedDeployment.Spec.Template.Spec.Volumes == nil {
		generatedDeployment.Spec.Template.Spec.Volumes = make([]corev1.Volume, 0)
	}
	generatedDeployment.Spec.Template.Spec.Volumes = append(generatedDeployment.Spec.Template.Spec.Volumes, devModeVolumes...)

	// delete user's SecurityContext
	generatedDeployment.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{}

	// disable readiness probes
	for i := 0; i < len(generatedDeployment.Spec.Template.Spec.Containers); i++ {
		generatedDeployment.Spec.Template.Spec.Containers[i].LivenessProbe = nil
		generatedDeployment.Spec.Template.Spec.Containers[i].ReadinessProbe = nil
		generatedDeployment.Spec.Template.Spec.Containers[i].StartupProbe = nil
		generatedDeployment.Spec.Template.Spec.Containers[i].SecurityContext = nil
	}

	generatedDeployment.Spec.Template.Spec.Containers =
		append(generatedDeployment.Spec.Template.Spec.Containers, *sideCarContainer)

	// Create generated deployment
	if _, err = d.Client.CreateDeploymentAndWait(generatedDeployment); err != nil {
		return err
	}

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
		return errors.New("Generated Deployment num is not 1?")
	}
	if err = d.Client.DeleteDeployment(ss[0].Name, false); err != nil {
		return err
	}
	return d.UpdateSvcProfile(func(svcProfileV2 *profile.SvcProfileV2) error {
		svcProfileV2.DevModeType = ""
		svcProfileV2.DuplicateDevMode = false
		return nil
	})
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
