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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/log"
	"strings"
	"time"
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
	pods, err := d.GetPodList()
	if err != nil {
		return "", err
	}
	return findDevPod(pods)
}

// ReplaceImage Create a duplicate deployment instead of replacing image
func (d *DuplicateDeploymentController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {
	var err error
	d.Client.Context(ctx)

	dep, err := d.Client.GetDeployment(d.Name)
	if err != nil {
		return err
	}

	if d.IsInReplaceDevMode() {
		osj, ok := dep.Annotations[OriginSpecJson]
		if ok {
			log.Info("Annotation nocalhost.origin.spec.json found, use it")
			dep.Spec = appsv1.DeploymentSpec{}
			if err = json.Unmarshal([]byte(osj), &dep.Spec); err != nil {
				return errors.Wrap(err, "")
			}
		} else {
			return errors.New("Annotation nocalhost.origin.spec.json not found?")
		}
	} else {
		osj, ok := dep.Annotations[OriginSpecJson]
		if ok {
			log.Info("Annotation nocalhost.origin.spec.json found, use it")
			oSpec := appsv1.DeploymentSpec{}
			if err = json.Unmarshal([]byte(osj), &oSpec); err == nil {
				dep.Spec = oSpec
			}
		}
	}

	var rs int32 = 1
	dep.Spec.Replicas = &rs

	//suffix := d.Identifier[0:5]
	labelsMap, err := d.getDuplicateLabelsMap()
	if err != nil {
		return err
	}
	dep.Name = d.getDuplicateResourceName()
	dep.Labels = labelsMap
	dep.Status = appsv1.DeploymentStatus{}
	dep.Spec.Selector = &metav1.LabelSelector{MatchLabels: labelsMap}
	dep.Spec.Template.Labels = labelsMap
	dep.ResourceVersion = ""

	devContainer, sideCarContainer, devModeVolumes, err :=
		d.genContainersAndVolumes(&dep.Spec.Template.Spec, ops.Container, ops.DevImage, ops.StorageClass, true)
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
		svcProfile := d.Config()
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

	for _, patch := range d.config.GetContainerDevConfigOrDefault(ops.Container).Patches {
		log.Infof("Patching %s", patch.Patch)
		if err = d.Client.Patch(d.Type.String(), dep.Name, patch.Patch, patch.Type); err != nil {
			log.WarnE(err, "")
		}
	}
	<-time.Tick(time.Second)

	return waitingPodToBeReady(d.GetNocalhostDevContainerPod)
}

func (d *DuplicateDeploymentController) RollBack(reset bool) error {
	lmap, err := d.getDuplicateLabelsMap()
	if err != nil {
		return err
	}
	d.Client.Labels(lmap)
	deploys, err := d.Client.ListDeployments()
	if err != nil {
		return err
	}

	if len(deploys) != 1 {
		if !reset {
			return errors.New(fmt.Sprintf("Duplicate Deployment num is %d (not 1)?", len(deploys)))
		} else if len(deploys) == 0 {
			log.Warnf("Duplicate Deployment num is %d (not 1)?", len(deploys))
			_ = d.UpdateSvcProfile(func(svcProfileV2 *profile.SvcProfileV2) error {
				svcProfileV2.DevModeType = ""
				return nil
			})
			return nil
		}
	}

	if err = d.Client.DeleteDeployment(deploys[0].Name, false); err != nil {
		return err
	}
	return d.UpdateSvcProfile(func(svcProfileV2 *profile.SvcProfileV2) error {
		svcProfileV2.DevModeType = ""
		return nil
	})
}

// GetPodList todo: Do not list pods already deleted - by hxx
func (d *DuplicateDeploymentController) GetPodList() ([]corev1.Pod, error) {
	labelsMap, err := d.getDuplicateLabelsMap()
	if err != nil {
		return nil, err
	}
	return d.Client.Labels(labelsMap).ListPods()
}
