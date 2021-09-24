/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"context"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/log"
	"strings"
)

const (
	IdentifierKey         = "identifier"
	OriginWorkloadNameKey = "origin-workload-name"
	OriginWorkloadTypeKey = "origin-workload-type"
)

type DuplicateDeploymentController struct {
	*Controller
}

func (d *DuplicateDeploymentController) GetNocalhostDevContainerPod() (string, error) {
	labelsMap, err := d.getLabelsMap()
	if err != nil {
		return "", err
	}
	d.Client.Labels(labelsMap)
	pods, err := d.Client.ListPods()
	if err != nil {
		return "", err
	}
	podName, err := findDevPod(pods)
	if err != nil {
		return "", err
	}
	return podName, nil
}

func (d *DuplicateDeploymentController) getLabelsMap() (map[string]string, error) {
	p, err := d.GetAppProfile()
	if err != nil {
		return nil, err
	}
	if p.Identifier == "" {
		return nil, errors.New("Identifier can not be nil ")
	}

	labelsMap := map[string]string{
		IdentifierKey:         p.Identifier,
		OriginWorkloadNameKey: d.Name,
		OriginWorkloadTypeKey: string(d.Type),
	}
	return labelsMap, nil
}

// ReplaceImage Create a duplicate deployment instead of replacing image
func (d *DuplicateDeploymentController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {
	var err error
	d.Client.Context(ctx)

	dep, err := d.Client.GetDeployment(d.Name)
	if err != nil {
		return err
	}

	var rs int32 = 1
	dep.Spec.Replicas = &rs

	p, err := d.GetAppProfile()
	if err != nil {
		return err
	}
	if p.Identifier == "" {
		return errors.New("Identifier can not be nil ")
	}

	suffix := p.Identifier[0:5]
	labelsMap, err := d.getLabelsMap()
	if err != nil {
		return err
	}
	dep.Name = strings.Join([]string{dep.Name, suffix}, "-")
	dep.Labels = labelsMap
	dep.Status = appsv1.DeploymentStatus{}
	dep.Spec.Selector = &metav1.LabelSelector{MatchLabels: labelsMap}
	dep.Spec.Template.Labels = labelsMap

	devContainer, err := findContainerInDeploySpec(dep, ops.Container)
	if err != nil {
		return err
	}

	devContainer, sideCarContainer, devModeVolumes, err :=
		d.genContainersAndVolumes(devContainer, ops.Container, ops.StorageClass)
	if err != nil {
		return err
	}

	if ops.Container != "" {
		for index, c := range dep.Spec.Template.Spec.Containers {
			if c.Name == ops.Container {
				dep.Spec.Template.Spec.Containers[index] = *devContainer
				break
			}
		}
	} else {
		dep.Spec.Template.Spec.Containers[0] = *devContainer
	}

	// Add volumes to deployment spec
	if dep.Spec.Template.Spec.Volumes == nil {
		log.Debugf("Service %s has no volume", dep.Name)
		dep.Spec.Template.Spec.Volumes = make([]corev1.Volume, 0)
	}
	dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, devModeVolumes...)

	// delete user's SecurityContext
	dep.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{}

	// disable readiness probes
	for i := 0; i < len(dep.Spec.Template.Spec.Containers); i++ {
		dep.Spec.Template.Spec.Containers[i].LivenessProbe = nil
		dep.Spec.Template.Spec.Containers[i].ReadinessProbe = nil
		dep.Spec.Template.Spec.Containers[i].StartupProbe = nil
		dep.Spec.Template.Spec.Containers[i].SecurityContext = nil
	}

	dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, *sideCarContainer)

	// PriorityClass
	priorityClass := ops.PriorityClass
	if priorityClass == "" {
		svcProfile, _ := d.GetConfig()
		priorityClass = svcProfile.PriorityClass
	}
	if priorityClass != "" {
		log.Infof("Using priorityClass: %s...", priorityClass)
		dep.Spec.Template.Spec.PriorityClassName = priorityClass
	}

	log.Info("Create development container...")
	_, err = d.Client.CreateDeployment(dep)
	if err != nil {
		for i := 0; i < 10; i++ {
			if strings.Contains(err.Error(), "no PriorityClass") {
				log.Warnf("PriorityClass %s not found, disable it...", priorityClass)
				dep, err = d.Client.GetDeployment(d.GetName())
				if err != nil {
					return err
				}
				dep.Spec.Template.Spec.PriorityClassName = ""
				_, err = d.Client.UpdateDeployment(dep, true)
				if err != nil {
					if strings.Contains(err.Error(), "Operation cannot be fulfilled on") {
						log.Warn("Deployment has been modified, retrying...")
						continue
					}
					return err
				}
				break
			} else if strings.Contains(err.Error(), "Operation cannot be fulfilled on") {
				log.Warn("Deployment has been modified, retrying...")
				continue
			}
			return err
		}
	}
	return waitingPodToBeReady(d.GetNocalhostDevContainerPod)
}

func (d *DuplicateDeploymentController) RollBack(reset bool) error {
	//todo
	err := d.UpdateSvcProfile(func(svcProfileV2 *profile.SvcProfileV2) error {
		svcProfileV2.LocalDevMode = ""
		svcProfileV2.LocalDevModeStarted = false
		return nil
	})
	if err != nil {
		return err
	}

	panic("")
}

func (d *DuplicateDeploymentController) GetPodList() ([]corev1.Pod, error) {
	//todo
	panic("")
}
